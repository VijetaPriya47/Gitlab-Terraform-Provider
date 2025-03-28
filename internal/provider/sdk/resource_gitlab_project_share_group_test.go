//go:build acceptance
// +build acceptance

package sdk

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func testResourceGitlabProjectShareGroupStateDataV0() map[string]interface{} {
	// Explicitly left as "project_id" as opposed to "project" for backwards compatibility.
	return map[string]interface{}{
		"project_id":   "1",
		"group_id":     "2",
		"access_level": "maintainer",
	}
}

func testResourceGitlabProjectShareGroupStateDataV1() map[string]interface{} {
	v0 := testResourceGitlabProjectShareGroupStateDataV0()
	return map[string]interface{}{
		"project_id":   v0["project_id"],
		"group_id":     v0["group_id"],
		"group_access": v0["access_level"],
	}
}

func TestResourceGitlabProjectShareGroupStateUpgradeV0(t *testing.T) {
	expected := testResourceGitlabProjectShareGroupStateDataV1()
	actual, err := resourceGitlabProjectShareGroupStateUpgradeV0(context.Background(), testResourceGitlabProjectShareGroupStateDataV0(), nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}

func TestAccGitlabProjectShareGroup_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectShareGroupDestroy,
		Steps: []resource.TestStep{
			// Share a new project with a new group.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_share_group" "test" {
						project      = "%d"
						group_id     = "%d"
						group_access = "guest"
					}
				`, project.ID, group.ID),
				Check: testAccCheckGitlabProjectSharedWithGroup(project.ID, group.ID, gitlab.GuestPermissions),
			},
			// Update the access level.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_share_group" "test" {
						project      = "%d"
						group_id     = "%d"
						group_access = "reporter"
					}
				`, project.ID, group.ID),
				Check: testAccCheckGitlabProjectSharedWithGroup(project.ID, group.ID, gitlab.ReporterPermissions),
			},
			// Delete the gitlab_project_share_group resource.
			{
				Config: ` `, // Space is required
				Check:  testAccCheckGitlabProjectIsNotShared(project.ID),
			},
		},
	})
}

func TestAccGitlabProjectShareGroup_modifiedOutsideTerraform(t *testing.T) {
	// Create the project and groups to use
	project := testutil.CreateProject(t)
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectShareGroupDestroy,
		Steps: []resource.TestStep{
			// Share a new project with a new group.
			{
				Config: fmt.Sprintf(`
				  resource "gitlab_project_share_group" "test" {
					project      = %d
					group_id     = %d
					group_access = "reporter"
				  }
				`, project.ID, group.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_share_group.test", "project"),
				),
			},
			{
				// Remove the group outside of terraform before we do our `Plan` run
				PreConfig: func() {
					// Remove the project share group since we're simulating removing it between steps.
					_, err := testutil.TestGitlabClient.Projects.DeleteSharedProjectFromGroup(project.ID, group.ID, nil)
					if err != nil {
						t.Fatalf("Failed to remove the project share outside terraform")
					}
				},

				// Then run our plan
				Config: fmt.Sprintf(`
				  resource "gitlab_project_share_group" "test" {
					project      = %d
					group_id     = %d
					group_access = "reporter"
				  }
				`, project.ID, group.ID),
				ExpectNonEmptyPlan: true,
				PlanOnly:           true,
			},
		},
	})
}

func testAccCheckGitlabProjectSharedWithGroup(projectID int, groupID int, accessLevel gitlab.AccessLevelValue) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		project, _, err := testutil.TestGitlabClient.Projects.GetProject(projectID, nil)
		if err != nil {
			return err
		}

		group, _, err := testutil.TestGitlabClient.Groups.GetGroup(groupID, nil)
		if err != nil {
			return err
		}

		for _, share := range project.SharedWithGroups {
			if share.GroupID == group.ID {
				if gitlab.AccessLevelValue(share.GroupAccessLevel) != accessLevel {
					return fmt.Errorf("groupAccessLevel was %d (wanted %d)", share.GroupAccessLevel, accessLevel)
				}
				return nil
			}
		}

		return errors.New("project is not shared with group")
	}
}

func testAccCheckGitlabProjectIsNotShared(projectID int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		project, _, err := testutil.TestGitlabClient.Projects.GetProject(projectID, nil)
		if err != nil {
			return err
		}

		if len(project.SharedWithGroups) != 0 {
			return fmt.Errorf("project is shared with %d groups (wanted 0)", len(project.SharedWithGroups))
		}

		return nil
	}
}

func testAccCheckGitlabProjectShareGroupDestroy(s *terraform.State) error {
	var projectId string
	var groupId int
	var err error

	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_share_group" {
			projectId, groupId, err = projectAndGroupIdFromId(rs.Primary.ID)
			if err != nil {
				return fmt.Errorf("[ERROR] cannot get project ID and group ID from input: %v", rs.Primary.ID)
			}

			proj, _, err := testutil.TestGitlabClient.Projects.GetProject(projectId, nil)
			if err != nil {
				return err
			}

			for _, v := range proj.SharedWithGroups {
				if groupId == v.GroupID {
					return fmt.Errorf("GitLab Project Share %d still exists", groupId)
				}
			}
		}
	}

	return nil
}
