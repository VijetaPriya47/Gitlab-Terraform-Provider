//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectTag_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	tag := testutil.CreateTags(t, project, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_tag" "foo" {
						name    = "%s"
						project = "%s"
					}
				`, tag.Name, project.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_tag.foo", "name", tag.Name),
					resource.TestCheckResourceAttr("data.gitlab_project_tag.foo", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_project_tag.foo", "message", tag.Message),
					resource.TestCheckResourceAttr("data.gitlab_project_tag.foo", "protected", fmt.Sprintf("%t", tag.Protected)),
					resource.TestCheckResourceAttr("data.gitlab_project_tag.foo", "target", tag.Target),
				),
			},
		},
	})
}
