//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabMemberRole_basic(t *testing.T) {

	testutil.SkipIfCE(t)

	testMemberRole := testutil.CreateMemberRole(t)

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_member_role" "test" {
						id = "%s"
					}
						`, testMemberRole.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_member_role.test", "name", testMemberRole.Name),
					resource.TestCheckResourceAttr("data.gitlab_member_role.test", "base_access_level", testMemberRole.BaseAccessLevel.StringValue),
					resource.TestCheckResourceAttr("data.gitlab_member_role.test", "description", testMemberRole.Description),
					resource.TestCheckResourceAttr("data.gitlab_member_role.test", "enabled_permissions.#", "1"),
				),
			},
		},
	})
}

func TestAccDataGitlabMemberRole_InvalidRole(t *testing.T) {

	testutil.SkipIfCE(t)

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "gitlab_member_role" "test" {
						id = "42"
					}
						`,
				ExpectError: regexp.MustCompile("Member role does not exist"),
			},
		},
	})
}
