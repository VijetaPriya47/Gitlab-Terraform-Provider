//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectProtectedTag_basic(t *testing.T) {
	// Create a project
	project := testutil.CreateProject(t)
	// Create protected tag on the project
	tag := testutil.CreateProtectedTags(t, project, 1)

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`				
				data "gitlab_project_protected_tag" "test" {
				  project = %d
				  tag     = "%s"
				}
				`, project.ID, tag[0].Name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tag.test",
						"tag",
						tag[0].Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tag.test",
						"create_access_levels.0.access_level",
						"maintainer",
					),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectProtectedTag_customAccessLevels(t *testing.T) {
	// Create a project
	project := testutil.CreateProject(t)

	tagName := fmt.Sprintf("TagProtect-%d", acctest.RandInt())

	testutil.CreateProtectedTagWithOptions(t, project, &gitlab.ProtectRepositoryTagsOptions{
		Name:              gitlab.Ptr(tagName),
		CreateAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	})

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project_protected_tag" "test" {
				  project = %d
				  tag     = "%s"
				}
				`, project.ID, tagName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tag.test",
						"tag",
						tagName,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tag.test",
						"create_access_levels.0.access_level",
						"developer",
					),
				),
			},
		},
	})
}
