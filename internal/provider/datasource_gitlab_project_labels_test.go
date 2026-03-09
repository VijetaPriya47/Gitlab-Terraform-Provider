//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectLabels_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testutil.CreateProjectLabelsWithCustomOptions(t, testProject.ID, []*gitlab.CreateLabelOptions{
		{
			Name:        gitlab.Ptr("test-label-1"),
			Color:       gitlab.Ptr("#FF0000"),
			Description: gitlab.Ptr("First test label"),
		},
		{
			Name:        gitlab.Ptr("test-label-2"),
			Color:       gitlab.Ptr("#00FF00"),
			Description: gitlab.Ptr("Second test label"),
		},
		{
			Name:  gitlab.Ptr("test-label-3"),
			Color: gitlab.Ptr("#0000FF"),
		},
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_labels" "test" {
						project = %d
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "id"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.#", "3"),
					// Verify all labels have required attributes
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.id"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.color"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.text_color"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.is_project_label"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.1.id"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.1.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.1.color"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.2.id"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.2.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.2.color"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectLabels_projectByPath(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testutil.CreateProjectLabelsWithCustomOptions(t, testProject.ID, []*gitlab.CreateLabelOptions{
		{
			Name:  gitlab.Ptr("path-test-label"),
			Color: gitlab.Ptr("#FFFF00"),
		},
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_labels" "test" {
						project = "%s"
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "id"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.#", "1"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.id"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.0.name", "path-test-label"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.0.color", "#FFFF00"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectLabels_empty(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_labels" "test" {
						project = %d
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "id"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectLabels_allAttributes(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testutil.CreateProjectLabelsWithCustomOptions(t, testProject.ID, []*gitlab.CreateLabelOptions{
		{
			Name:        gitlab.Ptr("full-featured-label"),
			Color:       gitlab.Ptr("#123456"),
			Description: gitlab.Ptr("A label with all attributes"),
		},
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_labels" "test" {
						project = %d
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.#", "1"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.id"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.0.name", "full-featured-label"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.0.color", "#123456"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.text_color"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.0.description", "A label with all attributes"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.0.is_project_label", "true"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.open_issues_count"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.closed_issues_count"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.open_merge_requests_count"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "labels.0.subscribed"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectLabels_pagination(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Create more than 20 labels to test pagination (default page size is 20)
	var labelOpts []*gitlab.CreateLabelOptions
	for i := range 25 {
		labelOpts = append(labelOpts, &gitlab.CreateLabelOptions{
			Name:  gitlab.Ptr(fmt.Sprintf("pagination-label-%02d", i)),
			Color: gitlab.Ptr("#000000"),
		})
	}
	testutil.CreateProjectLabelsWithCustomOptions(t, testProject.ID, labelOpts)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_labels" "test" {
						project = %d
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_project_labels.test", "id"),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_labels.test", "labels.#", "25"),
				),
			},
		},
	})
}
