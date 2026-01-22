//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectIssueLink_basic(t *testing.T) {

	sourceProject := testutil.CreateProject(t)
	targetProject := testutil.CreateProject(t)

	sourceIssues := testutil.CreateProjectIssues(t, sourceProject.ID, 1)
	targetIssues := testutil.CreateProjectIssues(t, targetProject.ID, 1)

	sourceIssue := sourceIssues[0]
	targetIssue := targetIssues[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIssueLinkDestroy,
		Steps: []resource.TestStep{
			// Create issue link
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_issue_link" "this" {
					project			  = "%d"
					issue_iid         = "%d"
					target_project_id = "%d"
					target_issue_iid  = "%d"
					link_type         = "relates_to"
				}`, sourceProject.ID, sourceIssue.IID, targetProject.ID, targetIssue.IID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_issue_link.this", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_issue_link.this", "issue_link_id"),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "project", fmt.Sprintf("%d", sourceProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "issue_iid", fmt.Sprintf("%d", sourceIssue.IID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "target_project_id", fmt.Sprintf("%d", targetProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "target_issue_iid", fmt.Sprintf("%d", targetIssue.IID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "link_type", "relates_to"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_issue_link.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectIssueLink_ensureReplacement(t *testing.T) {
	testutil.SkipIfCE(t)

	sourceProject := testutil.CreateProject(t)
	targetProject := testutil.CreateProject(t)

	sourceIssues := testutil.CreateProjectIssues(t, sourceProject.ID, 1)
	targetIssues := testutil.CreateProjectIssues(t, targetProject.ID, 1)

	sourceIssue := sourceIssues[0]
	targetIssue := targetIssues[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIssueLinkDestroy,
		Steps: []resource.TestStep{
			// Create initial resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_issue_link" "this" {
					project			  = "%d"
					issue_iid         = "%d"
					target_project_id = "%d"
					target_issue_iid  = "%d"
					link_type         = "relates_to"
				}`, sourceProject.ID, sourceIssue.IID, targetProject.ID, targetIssue.IID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_issue_link.this", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_issue_link.this", "issue_link_id"),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "project", fmt.Sprintf("%d", sourceProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "issue_iid", fmt.Sprintf("%d", sourceIssue.IID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "target_project_id", fmt.Sprintf("%d", targetProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "target_issue_iid", fmt.Sprintf("%d", targetIssue.IID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "link_type", "relates_to"),
				),
			},
			// Change link_type - should trigger replacement, not update
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_issue_link" "this" {
					project			  = "%d"
					issue_iid         = "%d"
					target_project_id = "%d"
					target_issue_iid  = "%d"
					link_type         = "blocks"
				}`, sourceProject.ID, sourceIssue.IID, targetProject.ID, targetIssue.IID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gitlab_project_issue_link.this", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_issue_link.this", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_issue_link.this", "issue_link_id"),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "project", fmt.Sprintf("%d", sourceProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "issue_iid", fmt.Sprintf("%d", sourceIssue.IID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "target_project_id", fmt.Sprintf("%d", targetProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "target_issue_iid", fmt.Sprintf("%d", targetIssue.IID)),
					resource.TestCheckResourceAttr("gitlab_project_issue_link.this", "link_type", "blocks"),
				),
			},
		},
	})
}

func testAccCheckGitlabProjectIssueLinkDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_issue_link" {
			continue
		}

		project, issueIID, issueLinkID, err := parseIssueLinkID(rs.Primary.ID)
		if err != nil {
			return err
		}

		issueLink, _, err := testutil.TestGitlabClient.IssueLinks.GetIssueLink(project, issueIID, issueLinkID, nil)
		if err == nil && issueLink != nil {
			return errors.New("Issue link still exists")
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
