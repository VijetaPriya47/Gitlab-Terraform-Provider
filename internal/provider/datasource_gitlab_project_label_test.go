//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectLabel_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testLabel := testutil.CreateProjectLabelsWithCustomOptions(t, testProject.ID, []*gitlab.CreateLabelOptions{
		{
			Name:        gitlab.Ptr("test-label"),
			Color:       gitlab.Ptr("#FF0000"),
			Description: gitlab.Ptr("Test label for acceptance tests"),
		},
	})[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_label" "test" {
						project  = %d
						label_id = %d
					}
				`, testProject.ID, testLabel.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "id", fmt.Sprintf("%d:%d", testProject.ID, testLabel.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "label_id", fmt.Sprintf("%d", testLabel.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "name", "test-label"),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "color", "#FF0000"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_label.test", "text_color"),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "description", "Test label for acceptance tests"),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "is_project_label", "true"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_label.test", "open_issues_count"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_label.test", "closed_issues_count"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_label.test", "open_merge_requests_count"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectLabel_projectByPath(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testLabel := testutil.CreateProjectLabelsWithCustomOptions(t, testProject.ID, []*gitlab.CreateLabelOptions{
		{
			Name:  gitlab.Ptr("path-test-label"),
			Color: gitlab.Ptr("#FFFF00"),
		},
	})[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_label" "test" {
						project  = "%s"
						label_id = %d
					}
				`, testProject.PathWithNamespace, testLabel.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "id", fmt.Sprintf("%s:%d", testProject.PathWithNamespace, testLabel.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "label_id", fmt.Sprintf("%d", testLabel.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "name", "path-test-label"),
					resource.TestCheckResourceAttr("data.gitlab_project_label.test", "color", "#FFFF00"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectLabel_notFound(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_label" "test" {
						project  = %d
						label_id = 99999
					}
				`, testProject.ID),
				ExpectError: regexp.MustCompile("Failed to get project label"),
			},
		},
	})
}
