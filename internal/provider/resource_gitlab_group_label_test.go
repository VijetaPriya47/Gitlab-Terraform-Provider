//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupLabel_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupLabelDestroy,
		Steps: []resource.TestStep{
			// Create a basic label
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_label" "foo" {
						group = "%d"
						name = "label-%d"
						color = "#FF0000"
					}
				`, group.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_label.foo", "group", fmt.Sprintf("%d", group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_label.foo", "name", fmt.Sprintf("label-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_group_label.foo", "color", "#FF0000"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_group_label.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the label
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_label" "foo" {
						group = "%d"
						name = "label-%d-updated"
						color = "#00FF00"
						description = "Group label description"
					}
				`, group.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_label.foo", "group", fmt.Sprintf("%d", group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_label.foo", "name", fmt.Sprintf("label-%d-updated", rInt)),
					resource.TestCheckResourceAttr("gitlab_group_label.foo", "color", "#00FF00"),
					resource.TestCheckResourceAttr("gitlab_group_label.foo", "description", "Group label description"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_group_label.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroupLabel_migrateFromSDKToFramework(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupLabelDestroy,
		Steps: []resource.TestStep{
			// Create the label in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.8",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_group_label" "foo" {
						group = "%s"
						name = "test-label"
						color = "#FF0000"
						description = "Group label description"
					}
				`, group.Path),
				Check: resource.TestCheckResourceAttrSet("gitlab_group_label.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_group_label" "foo" {
						group = "%s"
						name = "test-label"
						color = "#FF0000"
						description = "Group label description"
					}
				`, group.Path),
				Check: resource.TestCheckResourceAttrSet("gitlab_group_label.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_group_label.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func TestAccGitlabGroupLabel_schemaMigrationV0toV2(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	config := fmt.Sprintf(`
	resource "gitlab_group_label" "foo" {
		group = "%d"
		name = "test-label"
		color = "#FF0000"
		description = "Group label description"
	}
	`, group.ID)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 15.7", // Before v1 schema, has a v0 state.
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
			},
		},
	})
}

func TestAccGitlabGroupLabel_schemaMigrationV1toV2(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	config := fmt.Sprintf(`
	resource "gitlab_group_label" "foo" {
		group = "%d"
		name = "test-label"
		color = "#FF0000"
		description = "Group label description"
	}
	`, group.ID)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.3.0", // Before SDK -> Framework migration, has a v1 state.
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
			},
		},
	})
}

func testAccCheckGitlabGroupLabelDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_label" {
			continue
		}

		group, labelID, err := (&gitlabGroupLabelResourceModel{}).ResourceGitlabGroupLabelParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.GroupLabels.GetGroupLabel(group, labelID)
		if err == nil {
			return fmt.Errorf("Group Label %d in group %s still exists", labelID, group)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
