//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupEpicBoard_AllOnCreateEE(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	// testUsers := testutil.CreateUsers(t, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupEpicBoardDestroy,
		Steps: []resource.TestStep{
			// Verify creation with all attributes set (some are only available in the update API)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_epic_board" "this" {
						group        = "%d"
						name         = "Test Group Board"
					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_epic_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Verify update with changed attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_epic_board" "this" {
						group        = "%d"
						name         = "Renamed Group Board"
					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_epic_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroupEpicBoard_Lists(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testLabels := testutil.CreateGroupLabels(t, testGroup.ID, 4)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupEpicBoardDestroy,
		Steps: []resource.TestStep{
			// Create Board with 2 lists with core features
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_epic_board" "this" {
						group        = "%d"
						name         = "Test Group Board"

						lists {
							label_id = %d
						}

						lists {
							label_id = %d
						}
					}
				`, testGroup.ID, testLabels[0].ID, testLabels[2].ID),
			},
			// Verify import
			{
				ResourceName:      "gitlab_group_epic_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update Board list labels
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_epic_board" "this" {
						group        = "%d"
						name         = "Test Group Board"

						lists {
							label_id = %d
						}

						lists {
							label_id = %d
						}
					}
				`, testGroup.ID, testLabels[2].ID, testLabels[3].ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_epic_board.this",
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
					resource "gitlab_group_epic_board" "this" {
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
				ResourceName:      "gitlab_group_epic_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
					resource "gitlab_group_epic_board" "this" {
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
				ResourceName:      "gitlab_group_epic_board.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func resourceGitlabGroupEpicBoardParseID(id string) (string, int64, error) {
	group, rawIssueBoardID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	epicBoardID, err := strconv.ParseInt(rawIssueBoardID, 10, 64)
	if err != nil {
		return "", 0, err
	}

	return group, epicBoardID, nil
}

func testAccCheckGitlabGroupEpicBoardDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_epic_board" {
			continue
		}

		group, epicBoardID, err := resourceGitlabGroupEpicBoardParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		subject, _, err := testutil.TestGitlabClient.GroupEpicBoards.GetGroupEpicBoard(group, epicBoardID)
		if err == nil && subject != nil {
			return fmt.Errorf("gitlab_group_epic_board resource '%s' still exists", rs.Primary.ID)
		}

		if err != nil && !api.Is404(err) {
			return err
		}

		return nil
	}
	return nil
}
