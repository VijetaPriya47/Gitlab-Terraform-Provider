//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabRelease_basic(t *testing.T) {
	// Create a project and release
	project := testutil.CreateProject(t)
	release := testutil.CreateReleases(t, project, 1)[0]

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_release" "this" {
						project_id = %d
						tag_name = "%s"
					}
					`,
					project.ID,
					release.TagName,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_release.this", "project_id", strconv.Itoa(project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_release.this", "tag_name", release.TagName),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "name"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "created_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "released_at"),

					resource.TestCheckResourceAttr("data.gitlab_release.this", "assets.sources.#", "4"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.sources.0.format"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.sources.0.url"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.sources.1.url"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.sources.2.url"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.sources.3.url"),

					resource.TestCheckResourceAttr("data.gitlab_release.this", "assets.links.#", "2"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.links.0.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.links.0.url"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.links.0.id"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.links.0.link_type"),
					resource.TestCheckResourceAttrSet("data.gitlab_release.this", "assets.links.1.name"),
				),
			},
		},
	})
}
