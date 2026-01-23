//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabProjectMilestone_basic(t *testing.T) {
	rInt1, rInt2 := acctest.RandInt(), acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMilestoneDestroy,
		Steps: []resource.TestStep{
			{
				// create Milestone with required values only
				Config: fmt.Sprintf(`
				resource "gitlab_project_milestone" "this" {
					project = "%v"
					title   = "test-%d"
				}`, project.PathWithNamespace, rInt1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMilestoneExists("gitlab_project_milestone.this"),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "title", fmt.Sprintf("test-%d", rInt1)),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "state", "active"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "milestone_id"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "project_id"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "updated_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "web_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "expired"),
				),
			},
			{
				// verify import
				ResourceName:      "gitlab_project_milestone.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// update some Milestone attributes
				Config: fmt.Sprintf(`
				resource "gitlab_project_milestone" "this" {
					project     = "%[1]d"
					title       = "test-%[2]d"
					description = "test-%[2]d"
					start_date  = "2022-04-10"
					due_date    = "2022-04-15"
					state       = "closed"
				}`, project.ID, rInt2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMilestoneExists("gitlab_project_milestone.this"),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "title", fmt.Sprintf("test-%d", rInt2)),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "description", fmt.Sprintf("test-%d", rInt2)),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "start_date", "2022-04-10"),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "due_date", "2022-04-15"),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "state", "closed"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "milestone_id"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "project_id"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "updated_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "web_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_milestone.this", "expired"),
				),
			},
			{
				// verify import
				ResourceName:      "gitlab_project_milestone.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectMilestone_attributeValidation(t *testing.T) {
	rInt1 := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMilestoneDestroy,
		Steps: []resource.TestStep{
			{
				// create Milestone with invalid due_date
				Config: fmt.Sprintf(`
				resource "gitlab_project_milestone" "this" {
					project  = "%v"
					title    = "test-%d"
					due_date = "2026/01/01"
				}`, project.PathWithNamespace, rInt1),
				ExpectError: regexp.MustCompile("Invalid Date Format"),
			},
			{
				// create Milestone with invalid start_date
				Config: fmt.Sprintf(`
				resource "gitlab_project_milestone" "this" {
					project  = "%v"
					title    = "test-%d"
					start_date = "2026/02/01"
				}`, project.PathWithNamespace, rInt1),
				ExpectError: regexp.MustCompile("Invalid Date Format"),
			},
			{
				// create Milestone with invalid state
				Config: fmt.Sprintf(`
				resource "gitlab_project_milestone" "this" {
					project = "%v"
					title   = "test-%d"
					state   = "madeup"
				}`, project.PathWithNamespace, rInt1),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
		},
	})
}

func TestAccGitlabProjectMilestone_migrateFromSDKToFramework(t *testing.T) {
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectMilestoneDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.8",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
				resource "gitlab_project_milestone" "this" {
					project = "%[1]d"
					title   = "test-%[2]d"
				}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMilestoneExists("gitlab_project_milestone.this"),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "title", fmt.Sprintf("test-%d", rInt)),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
				resource "gitlab_project_milestone" "this" {
					project = "%[1]d"
					title   = "test-%[2]d"
				}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectMilestoneExists("gitlab_project_milestone.this"),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_milestone.this", "title", fmt.Sprintf("test-%d", rInt)),
				),
			},
		},
	})
}

func testAccCheckGitlabProjectMilestoneExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		project, rawMilestoneID, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		milestoneID, err := strconv.ParseInt(rawMilestoneID, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse milestone ID: %v", err)
		}

		milestone, _, err := testutil.TestGitlabClient.Milestones.GetMilestone(project, milestoneID)
		if err != nil {
			return err
		}

		if milestone == nil {
			return fmt.Errorf("Milestone does not exist")
		}

		return nil
	}
}

func testAccCheckGitlabProjectMilestoneDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_milestone" {
			continue
		}

		project, rawMilestoneID, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		milestoneID, err := strconv.ParseInt(rawMilestoneID, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse milestone ID: %v", err)
		}

		milestone, _, err := testutil.TestGitlabClient.Milestones.GetMilestone(project, milestoneID)
		if err == nil && milestone != nil {
			return errors.New("Milestone still exists")
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
