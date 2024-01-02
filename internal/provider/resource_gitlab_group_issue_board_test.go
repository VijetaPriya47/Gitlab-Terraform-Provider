//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupIssueBoard_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testMilestone := testutil.AddGroupMilestones(t, testGroup, 1)[0]
	testLabels := testutil.CreateGroupLabels(t, testGroup.ID, 2)
	//testUser := testutil.CreateUsers(t, 1)[0]

	// NOTE: there is no way to delete the last issue board, see
	// https://gitlab.com/gitlab-org/gitlab/-/issues/367395
	testutil.CreateGroupIssueBoard(t, testGroup.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupIssueBoardDestroy,
		Steps: []resource.TestStep{
			// Verify creation
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group   = "%d"
						name    = "Test Group Board"
					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Verify update with optional values (all optional attributes are EE only)
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Test Group Board"
						milestone_id = %d
						labels       = ["%s","%s"]
					}
				`, testGroup.ID, testMilestone.ID, testLabels[0].Name, testLabels[1].Name),
			},
			// Verify Import
			{
				SkipFunc:          testutil.IsRunningInCE,
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroupIssueBoard_AllOnCreateEE(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	// testUsers := testutil.CreateUsers(t, 2)

	// NOTE: there is no way to delete the last issue board, see
	// https://gitlab.com/gitlab-org/gitlab/-/issues/367395
	testutil.CreateGroupIssueBoard(t, testGroup.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupIssueBoardDestroy,
		Steps: []resource.TestStep{
			// Verify creation with all attributes set (some are only available in the update API)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Test Group Board"
					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Verify update with changed attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Renamed Group Board"
					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroupIssueBoard_Lists(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testLabels := testutil.CreateGroupLabels(t, testGroup.ID, 4)
	// testUsers := testutil.CreateUsers(t, 2)
	// testutil.AddGroupMembers(t, testGroup.ID, testUsers)

	// NOTE: there is no way to delete the last issue board, see
	// https://gitlab.com/gitlab-org/gitlab/-/issues/367395
	testutil.CreateGroupIssueBoard(t, testGroup.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupIssueBoardDestroy,
		Steps: []resource.TestStep{
			// Create Board with 2 lists with core features
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Test Group Board"

						lists {
							label_id = %d
							position = 1
						}

						lists {
							label_id = %d
							position = 0
						}
						
					}
				`, testGroup.ID, testLabels[0].ID, testLabels[2].ID),
			},
			// Verify import
			{
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update Board list labels
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Test Group Board"

						lists {
							label_id = %d
							position = 0
						}

						lists {
							label_id = %d
							position = 1
						}
					}
				`, testGroup.ID, testLabels[2].ID, testLabels[3].ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Force a destroy for the board so that it can be recreated as the same resource
			{
				SkipFunc: testutil.IsRunningInCE,
				Config:   ` `, // requires a space for empty config
			},
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Test Group Board"

						lists {
							label_id = %d
						}

						
					}
				`, testGroup.ID, testLabels[0].ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Test Board"

						lists {
							label_id = %d
						}

						
					}
				`, testGroup.ID, testLabels[1].ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroupIssueBoard_LabelPositions(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testLabels := testutil.CreateGroupLabels(t, testGroup.ID, 4)

	// NOTE: there is no way to delete the last issue board, see
	// https://gitlab.com/gitlab-org/gitlab/-/issues/367395
	testutil.CreateGroupIssueBoard(t, testGroup.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupIssueBoardDestroy,
		Steps: []resource.TestStep{
			// Create Board with 2 lists with core features
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Test Group Board"

						lists {
							label_id = %d
							position = 1
						}

						lists {
							label_id = %d
							position = 0
						}
						
					}
				`, testGroup.ID, testLabels[0].ID, testLabels[2].ID),
			},
			// Verify import
			{
				ResourceName:      "gitlab_group_issue_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update Board list labels
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_issue_board" "this" {
						group        = "%d"
						name         = "Test Group Board"

						lists {
							label_id = %d
							position = 0
						}

						lists {
							label_id = %d
							position = 1
						}
					}
				`, testGroup.ID, testLabels[0].ID, testLabels[2].ID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func resourceGitlabGroupIssueBoardParseID(id string) (string, int, error) {
	group, rawIssueBoardID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	issueBoardID, err := strconv.Atoi(rawIssueBoardID)
	if err != nil {
		return "", 0, err
	}

	return group, issueBoardID, nil
}

func testAccCheckGitlabGroupIssueBoardDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_issue_board" {
			continue
		}

		group, issueBoardID, err := resourceGitlabGroupIssueBoardParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		subject, _, err := testutil.TestGitlabClient.GroupIssueBoards.GetGroupIssueBoard(group, issueBoardID)
		if err == nil && subject != nil {
			return fmt.Errorf("gitlab_group_issue_board resource '%s' still exists", rs.Primary.ID)
		}

		if err != nil && !api.Is404(err) {
			return err
		}

		return nil
	}
	return nil
}
