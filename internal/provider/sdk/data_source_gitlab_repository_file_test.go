//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabRepositoryFile_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_repository_file" "foo" {
						project = "%s"
						file_path = "README.md"
						ref = "main"
					}
				`, project.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_repository_file.foo", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_repository_file.foo", "file_path", "README.md"),
					resource.TestCheckResourceAttr("data.gitlab_repository_file.foo", "ref", "main"),
					resource.TestCheckResourceAttr("data.gitlab_repository_file.foo", "encoding", "base64"),
				),
			},
		},
	})
}
