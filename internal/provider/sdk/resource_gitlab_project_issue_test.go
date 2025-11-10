//go:build acceptance

package sdk

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectIssue_basic(t *testing.T) {
	var testIssue gitlab.Issue
	var updatedTestIssue gitlab.Issue

	testProject := testutil.CreateProject(t)
	testUser := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, testProject.ID, []*gitlab.User{testUser})
	testMilestone := testutil.AddProjectMilestones(t, testProject, 1)[0]

	currentUser, _, err := testutil.TestGitlabClient.Users.CurrentUser()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectIssueDestroy,
		Steps: []resource.TestStep{
			// create Issue with required values only
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_issue" "this" {
						// required
						project = "%s"
						title   = "Terraform test issue"
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIssueExists("gitlab_project_issue.this", &testIssue),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "iid", "1"),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "title", "Terraform test issue"),
					resource.TestCheckResourceAttrWith("gitlab_project_issue.this", "created_at", func(value string) error {
						expectedValue := testIssue.CreatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_issue.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_on_destroy"},
			},
			// update some Issue attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_issue" "this" {
						// required
						project = "%s"
						title   = "Terraform test issue"
					
						assignee_ids      = [%d]
						confidential      = true
						description       = "Terraform test issue description"
						issue_type        = "issue"
						labels            = ["foo", "bar"]
						milestone_id      = %d
						state 	          = "opened"
						discussion_locked = true
					}
				`, testProject.PathWithNamespace, testUser.ID, testMilestone.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIssueExists("gitlab_project_issue.this", &updatedTestIssue),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "iid", "1"),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "title", "Terraform test issue"),
					resource.TestCheckResourceAttrWith("gitlab_project_issue.this", "updated_at", func(value string) error {
						expectedValue := updatedTestIssue.UpdatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_issue.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_on_destroy"},
			},
			// go back to required values only
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_issue" "this" {
						// required
						project = "%s"
						title   = "Terraform test issue"
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIssueExists("gitlab_project_issue.this", &testIssue),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "iid", "1"),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "title", "Terraform test issue"),
					func(s *terraform.State) error {
						if !updatedTestIssue.UpdatedAt.Before(*testIssue.UpdatedAt) {
							return fmt.Errorf("expected issue to be updated, but it wasn't")
						}
						return nil
					},
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_issue.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_on_destroy"},
			},
			// close issue
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_issue" "this" {
						project  = "%s"
						title    = "Terraform test issue"
					
						state    = "%s"
					}
				`, testProject.PathWithNamespace, "closed"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIssueExists("gitlab_project_issue.this", &testIssue),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "state", "closed"),
					resource.TestCheckResourceAttrWith("gitlab_project_issue.this", "closed_at", func(value string) error {
						expectedValue := testIssue.ClosedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "closed_by_user_id", fmt.Sprintf("%d", currentUser.ID)),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_issue.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_on_destroy"},
			},
			// re-open issue
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_issue" "this" {
						project  = "%s"
						title    = "Terraform test issue"
					
						state    = "%s"
					}
				`, testProject.PathWithNamespace, "opened"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIssueExists("gitlab_project_issue.this", &testIssue),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "state", "opened"),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "closed_at", ""),
					resource.TestCheckResourceAttr("gitlab_project_issue.this", "closed_by_user_id", "0"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_issue.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_on_destroy"},
			},
		},
	})
}

func TestAccGitlabProjectIssue_basicEE(t *testing.T) {
	testutil.SkipIfCE(t)

	testProject := testutil.CreateProject(t)
	testUser := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, testProject.ID, []*gitlab.User{testUser})
	testMilestone := testutil.AddProjectMilestones(t, testProject, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectIssueDestroy,
		Steps: []resource.TestStep{
			// create Issue with EE features set
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_issue" "this" {
						// required
						project = "%s"
						title   = "Terraform test issue"
					
						assignee_ids      = [%d]
						confidential      = true
						description       = "Terraform test issue description"
						issue_type        = "issue"
						labels            = ["foo", "bar"]
						milestone_id      = %d
						weight            = 42
						state 	          = "opened"
						discussion_locked = true
					}
				`, testProject.PathWithNamespace, testUser.ID, testMilestone.ID),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_issue.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_on_destroy"},
			},
		},
	})
}

func TestAccGitlabProjectIssue_deleteOnDestroy(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectIssueDestroy,
		Steps: []resource.TestStep{
			// create Issue with required values only
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_issue" "this" {
						project = "%s"
						title   = "Terraform test issue"
					
						delete_on_destroy = true
					}
				`, testProject.PathWithNamespace),
			},
		},
	})
}

func testAccCheckGitlabProjectIssueExists(n string, issue *gitlab.Issue) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		project, issueIID, err := resourceGitLabProjectIssueParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing issue ID: %s", err)
		}

		gotIssue, _, err := testutil.TestGitlabClient.Issues.GetIssue(project, issueIID)
		if err != nil {
			return fmt.Errorf("Cannot get issue: %v", err)
		}

		tflog.Debug(context.Background(), fmt.Sprintf("[DEBUG] testAccCheckGitlabProjectIssueExists: %#v", gotIssue))

		*issue = *gotIssue
		return nil
	}
}

func testAccCheckGitlabProjectIssueDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_issue" {
			continue
		}

		project, issueIID, err := resourceGitLabProjectIssueParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		issue, _, err := testutil.TestGitlabClient.Issues.GetIssue(project, issueIID)
		if err == nil && issue != nil && issue.IID == issueIID {
			if val, ok := rs.Primary.Attributes["delete_on_destory"]; ok && val == "true" {
				return fmt.Errorf("Issue still exists")
			} else {
				if issue.State != "closed" {
					return fmt.Errorf("Issue still in state %s (should be closed)", issue.State)
				}
			}
		}
		if err != nil && !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
