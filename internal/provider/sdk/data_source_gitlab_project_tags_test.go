//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectTags_basic(t *testing.T) {
	countTags := 3
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_tag" "foo" {
						count   = "%[1]d"
						
						name    = "${count.index}"
						ref     = "main"
						project = "%s"
						message = "Tag ${count.index}"
					}
					
					data "gitlab_project_tags" "foo" {
						project  = "%s"
						order_by = "name"
						sort     = "asc"
						
						depends_on = [
							gitlab_project_tag.foo,
						]
					}
				`, countTags, project.PathWithNamespace, project.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabProjectTags("gitlab_project_tag.foo", "data.gitlab_project_tags.foo", countTags),
				),
			},
		},
	})
}

func testAccDataSourceGitlabProjectTags(src string, n string, countTags int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		testAttributes := []string{
			"commit",
			"release",
			"name",
			"message",
			"protected",
			"target",
		}

		for numberTag := range make([]int, countTags) {
			search := s.RootModule().Resources[n]
			searchAttrs := search.Primary.Attributes

			tag := s.RootModule().Resources[fmt.Sprintf("%s.%d", src, numberTag)]
			tagAttrs := tag.Primary.Attributes

			for _, attribute := range testAttributes {
				tagAttr := tagAttrs[attribute]
				searchAttr := searchAttrs[fmt.Sprintf("tags.%d.%s", numberTag, attribute)]
				if searchAttr != tagAttr {
					return fmt.Errorf("Expected the tag `%s` with parameter `%s` to be: `%s`, but got: `%s`", tagAttrs["name"], attribute, tagAttr, searchAttr)
				}
			}
		}

		return nil
	}
}
