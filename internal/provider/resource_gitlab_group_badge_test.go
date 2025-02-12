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

func TestAccGitlabGroupBadge_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	rInt := acctest.RandInt()
	rInt2 := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupBadgeDestroy,
		Steps: []resource.TestStep{
			// Create a basic badge
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_badge" "foo" {
						group     = "%d"
						link_url  = "https://example.com/badge-%d"
						image_url = "https://example.com/badge-%d.svg"
					}
				`, group.ID, rInt, rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_badge.foo", "group", fmt.Sprintf("%d", group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_badge.foo", "link_url", fmt.Sprintf("https://example.com/badge-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_group_badge.foo", "image_url", fmt.Sprintf("https://example.com/badge-%d.svg", rInt)),
				),
			},
			{
				ResourceName:      "gitlab_group_badge.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the badge
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_badge" "foo" {
						group     = "%d"
						link_url  = "https://example.com/new-badge-%d"
						image_url = "https://example.com/new-badge-%d.svg"
						name      = "badge-updated"
					}
				`, group.ID, rInt, rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_badge.foo", "group", fmt.Sprintf("%d", group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_badge.foo", "link_url", fmt.Sprintf("https://example.com/new-badge-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_group_badge.foo", "image_url", fmt.Sprintf("https://example.com/new-badge-%d.svg", rInt)),
					resource.TestCheckResourceAttr("gitlab_group_badge.foo", "name", "badge-updated"),
				),
			},
			{
				ResourceName:      "gitlab_group_badge.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Create a fully setup badge
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_badge" "foo2" {
						group     = "%d"
						link_url  = "https://example.com/badge-%d"
						image_url = "https://example.com/badge-%d.svg"
						name      = "badge2"
					}
				`, group.ID, rInt2, rInt2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_badge.foo2", "group", fmt.Sprintf("%d", group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_badge.foo2", "link_url", fmt.Sprintf("https://example.com/badge-%d", rInt2)),
					resource.TestCheckResourceAttr("gitlab_group_badge.foo2", "image_url", fmt.Sprintf("https://example.com/badge-%d.svg", rInt2)),
					resource.TestCheckResourceAttr("gitlab_group_badge.foo2", "name", "badge2"),
					resource.TestCheckResourceAttrSet("gitlab_group_badge.foo2", "rendered_link_url"),
					resource.TestCheckResourceAttrSet("gitlab_group_badge.foo2", "rendered_image_url"),
				),
			},
			{
				ResourceName:      "gitlab_group_badge.foo2",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroupBadge_migrateFromSDKToFramework(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupBadgeDestroy,
		Steps: []resource.TestStep{
			// Create the badge in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.8",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_group_badge" "foo" {
						group = "%s"
						link_url = "https://example.com/badge"
						image_url = "https://example.com/badge.svg"
					}
				`, group.Path),
				Check: resource.TestCheckResourceAttrSet("gitlab_group_badge.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_group_badge" "foo" {
						group = "%s"
						link_url = "https://example.com/badge"
						image_url = "https://example.com/badge.svg"
					}
				`, group.Path),
				Check: resource.TestCheckResourceAttrSet("gitlab_group_badge.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_group_badge.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func testAccCheckGitlabGroupBadgeDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_badge" {
			continue
		}

		group, badgeID, err := (&gitlabGroupBadgeResourceModel{}).ResourceGitlabGroupBadgeParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.GroupBadges.GetGroupBadge(group, badgeID)
		if err == nil {
			return fmt.Errorf("Group Badge %d in group %s still exists", badgeID, group)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
