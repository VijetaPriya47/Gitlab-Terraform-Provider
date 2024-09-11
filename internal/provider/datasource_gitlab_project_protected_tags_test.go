//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectProtectedTags_basic(t *testing.T) {
	// Create a project
	project := testutil.CreateProject(t)
	// Create protected tag on the project
	tags := testutil.CreateProtectedTags(t, project, 5)

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`			
				data "gitlab_project_protected_tags" "test" {
				  project = %d
				}
				`, project.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tags.test",
						"protected_tags.0.tag",
						tags[0].Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tags.test",
						"protected_tags.0.create_access_levels.0.access_level",
						"maintainer",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tags.test",
						"protected_tags.1.tag",
						tags[1].Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tags.test",
						"protected_tags.1.create_access_levels.0.access_level",
						"maintainer",
					),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectProtectedTags_customAccessLevels(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create a project
	project := testutil.CreateProject(t)

	myGroup := testutil.CreateGroups(t, 1)
	testutil.ProjectShareGroup(t, project.ID, myGroup[0].ID)

	tagName1 := fmt.Sprintf("TagProtect-%d", acctest.RandInt())
	tagName2 := fmt.Sprintf("TagProtect-%d", acctest.RandInt())

	testutil.CreateProtectedTagWithOptions(t, project, &gitlab.ProtectRepositoryTagsOptions{
		Name:              gitlab.Ptr(tagName1),
		CreateAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
		AllowedToCreate: &[]*gitlab.TagsPermissionOptions{
			{
				GroupID: gitlab.Ptr(myGroup[0].ID),
			},
		},
	})
	testutil.CreateProtectedTagWithOptions(t, project, &gitlab.ProtectRepositoryTagsOptions{
		Name:              gitlab.Ptr(tagName2),
		CreateAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
		AllowedToCreate: &[]*gitlab.TagsPermissionOptions{
			{
				GroupID: gitlab.Ptr(myGroup[0].ID),
			},
		},
	})

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project_protected_tags" "test" {
				  project = %d
				}
				`, project.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tags.test",
						"protected_tags.0.create_access_levels.0.access_level",
						"developer",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tags.test",
						"protected_tags.0.create_access_levels.1.group_id",
						strconv.Itoa(myGroup[0].ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tags.test",
						"protected_tags.1.create_access_levels.0.access_level",
						"developer",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_tags.test",
						"protected_tags.1.create_access_levels.1.group_id",
						strconv.Itoa(myGroup[0].ID),
					),
				),
			},
		},
	})
}
