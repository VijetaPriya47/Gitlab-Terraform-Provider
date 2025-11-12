//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectMirrorPublicKey_SSH(t *testing.T) {
	// Create a project and configure the SSH mirror
	project := testutil.CreateProject(t)
	projectMirror := testutil.CreateProjectMirrorWithOptions(t, project, &gitlab.AddProjectMirrorOptions{
		URL:        gitlab.Ptr("ssh://username@example.com/gitlab/example.git"),
		AuthMethod: gitlab.Ptr("ssh_public_key"),
	})

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`				
				data "gitlab_project_mirror_public_key" "test" {
				  project_id = %d
				  mirror_id  = %d
				}
				`, project.ID, projectMirror.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_mirror_public_key.test", "mirror_id", strconv.Itoa(projectMirror.ID)),
					resource.TestCheckResourceAttrSet("data.gitlab_project_mirror_public_key.test", "public_key"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectMirrorPublicKey_ErrorWithHTTP(t *testing.T) {
	// Create a project and configure the HTTP mirror
	project := testutil.CreateProject(t)
	projectMirror := testutil.CreateProjectMirrorWithOptions(t, project, &gitlab.AddProjectMirrorOptions{
		URL:        gitlab.Ptr("https://username:password@example.com/gitlab/example.git"),
		AuthMethod: gitlab.Ptr("password"),
	})

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`				
				data "gitlab_project_mirror_public_key" "test" {
				  project_id = %d
				  mirror_id  = %d
				}
				`, project.ID, projectMirror.ID),
				ExpectError: regexp.MustCompile("Unable to read project mirror public key"),
			},
		},
	})
}
