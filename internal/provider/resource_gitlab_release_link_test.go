//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabReleaseLink_basic(t *testing.T) {
	rInt1, rInt2 := acctest.RandInt(), acctest.RandInt()
	project := testutil.CreateProject(t)
	releases := testutil.CreateReleases(t, project, 1)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabReleaseLinkDestroy,
		Steps: []resource.TestStep{
			{
				// create Release link with required values only
				Config: fmt.Sprintf(`
				resource "gitlab_release_link" "this" {
					project  = "%s"
					tag_name = "%s"
					name     = "test-%d"
					url      = "https://test/%d"
				}`, project.PathWithNamespace, releases[0].TagName, rInt1, rInt1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "link_id"),
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "direct_asset_url"),
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "external"),
				),
			},
			{
				// verify import
				ResourceName:      "gitlab_release_link.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// update some Release link attributes
				Config: fmt.Sprintf(`
				resource "gitlab_release_link" "this" {
					project   = "%d"
					tag_name  = "%s"
					name      = "test-%d"
					url       = "https://test/%d"
					filepath  = "/test/%d"
					link_type = "runbook"
				}`, project.ID, releases[0].TagName, rInt2, rInt2, rInt2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "link_id"),
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "direct_asset_url"),
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "external"),
				),
			},
			{
				// verify import
				ResourceName:      "gitlab_release_link.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabReleaseLink_migrateFromSDKToFramework(t *testing.T) {
	rInt1 := acctest.RandInt()
	project := testutil.CreateProject(t)
	releases := testutil.CreateReleases(t, project, 1)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabReleaseLinkDestroy,
		Steps: []resource.TestStep{
			// Create the pipeline in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.5",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
				resource "gitlab_release_link" "this" {
					project  = "%s"
					tag_name = "%s"
					name     = "test-%d"
					url      = "https://test/%d"
				}`, project.PathWithNamespace, releases[0].TagName, rInt1, rInt1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "link_id"),
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "direct_asset_url"),
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "external"),
				),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config: fmt.Sprintf(`
				resource "gitlab_release_link" "this" {
					project  = "%s"
					tag_name = "%s"
					name     = "test-%d"
					url      = "https://test/%d"
				}`, project.PathWithNamespace, releases[0].TagName, rInt1, rInt1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "link_id"),
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "direct_asset_url"),
					resource.TestCheckResourceAttrSet("gitlab_release_link.this", "external"),
				),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_release_link.this",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func testAccCheckGitlabReleaseLinkDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_release_link" {
			continue
		}
		project, tagName, linkID, err := resourceGitLabReleaseLinkParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		releaseLink, _, err := testutil.TestGitlabClient.ReleaseLinks.GetReleaseLink(project, tagName, linkID)
		if err == nil && releaseLink != nil {
			return errors.New("Release link still exists")
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
