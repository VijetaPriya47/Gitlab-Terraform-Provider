//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectSecureFile_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	secureFiles := testutil.CreateProjectSecureFile(t, project.ID, 25)
	secureFile6 := secureFiles[6]
	secureFile24 := secureFiles[24]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Get secure file by name
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_secure_file" "name" {
						project = "%d"
						name = "%s"
					}
				`, project.ID, secureFile6.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.name", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.name", "secure_file_id", fmt.Sprintf("%d", secureFile6.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.name", "name", secureFile6.Name),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.name", "checksum", secureFile6.Checksum),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.name", "checksum_algorithm", secureFile6.ChecksumAlgorithm),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.name", "created_at", secureFile6.CreatedAt.String()),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.name", "content", "secure file content 6"),
				),
			},
			// Check error when Name doesn't match
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_secure_file" "name" {
						project = "%d"
						name = "no-exist"
					}
				`, project.ID),
				ExpectError: regexp.MustCompile("Secure File not found"),
			},
			// Get Secure File by ID
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_secure_file" "id" {
						project = "%d"
						secure_file_id = %d
					}
				`, project.ID, secureFile24.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.id", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.id", "secure_file_id", fmt.Sprintf("%d", secureFile24.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.id", "name", secureFile24.Name),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.id", "checksum", secureFile24.Checksum),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.id", "checksum_algorithm", secureFile24.ChecksumAlgorithm),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.id", "created_at", secureFile24.CreatedAt.String()),
					resource.TestCheckResourceAttr("data.gitlab_project_secure_file.id", "content", "secure file content 24"),
				),
			},
			// Check error when id doesn't exist
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_secure_file" "id" {
						project = "%d"
						secure_file_id = 404
					}
				`, project.ID),
				ExpectError: regexp.MustCompile("Error fetching secure file details"),
			},
		},
	})
}
