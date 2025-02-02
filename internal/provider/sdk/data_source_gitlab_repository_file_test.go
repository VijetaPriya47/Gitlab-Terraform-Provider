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

func TestAccDataGitlabRepositoryFile_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_repository_file" "foo" {
						project = "%s"
						file_path = "testfile-meow"
						branch = "main"
						content = base64encode("Meow goes the cat")
						commit_message = "feat: Meow"
					}
					
					data "gitlab_repository_file" "foo" {
						project = gitlab_repository_file.foo.project
						file_path = gitlab_repository_file.foo.file_path
						ref = "main"
					}
				`, project.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabRepositoryFile("gitlab_repository_file.foo", "data.gitlab_repository_file.foo"),
				),
			},
		},
	})
}

func testAccDataSourceGitlabRepositoryFile(src, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		file := s.RootModule().Resources[src]
		fileAttr := file.Primary.Attributes

		search := s.RootModule().Resources[n]
		searchAttr := search.Primary.Attributes

		testAttributes := []string{
			"project",
			"file_path",
			"size",
			"encoding",
			"content",
			"execute_filemode",
			"ref",
			"blob_id",
			"commit_id",
			"content_sha256",
			"last_commit_id",
		}

		for _, attribute := range testAttributes {
			if searchAttr[attribute] != fileAttr[attribute] {
				return fmt.Errorf("expected file's parameter `%s` to be: %s, but got: `%s`", attribute, fileAttr[attribute], searchAttr[attribute])
			}
		}
		return nil
	}
}
