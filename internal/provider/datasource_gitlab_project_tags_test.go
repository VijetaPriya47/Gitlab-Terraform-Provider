//go:build acceptance

package provider

import (
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectTags_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	tags := testutil.CreateTags(t, project, 3)
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_tags" "foo" {
						project  = "%s"
						order_by = "name"
						sort     = "asc"
					}
				`, project.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.#", "3"),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.0.name", tags[0].Name),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.1.name", tags[1].Name),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.2.name", tags[2].Name),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.0.message", tags[0].Message),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.1.message", tags[1].Message),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.2.message", tags[2].Message),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.0.protected", fmt.Sprintf("%t", tags[0].Protected)),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.1.protected", fmt.Sprintf("%t", tags[1].Protected)),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.2.protected", fmt.Sprintf("%t", tags[2].Protected)),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.0.target", tags[0].Target),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.1.target", tags[1].Target),
					resource.TestCheckResourceAttr("data.gitlab_project_tags.foo", "tags.2.target", tags[2].Target),
				),
			},
		},
	})
}
