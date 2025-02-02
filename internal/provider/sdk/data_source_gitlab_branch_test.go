//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabBranch_basic(t *testing.T) {
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch" "foo" {
						name = "testbranch-%[1]d"
						ref = "main"
						project = "%s"
					}
					
					data "gitlab_branch" "foo" {
						name = "${gitlab_branch.foo.name}"
						project = "%s"
					}
				`, rInt, project.PathWithNamespace, project.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabBranch("gitlab_branch.foo", "data.gitlab_branch.foo"),
				),
			},
		},
	})
}

func testAccDataSourceGitlabBranch(src, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		branch := s.RootModule().Resources[src]
		branchAttr := branch.Primary.Attributes

		search := s.RootModule().Resources[n]
		searchAttr := search.Primary.Attributes

		testAttributes := []string{
			"id",
			"name",
			"web_url",
			"default",
			"project",
			"can_push",
			"merged",
			"commit",
			"parent_ids",
			"protected",
			"developer_can_merge",
			"developer_can_push",
		}

		for _, attribute := range testAttributes {
			if searchAttr[attribute] != branchAttr[attribute] {
				return fmt.Errorf("expected branch's parameter `%s` to be: %s, but got: `%s`", attribute, branchAttr[attribute], searchAttr[attribute])
			}
		}
		return nil
	}
}
