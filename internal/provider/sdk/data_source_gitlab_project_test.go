//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProject_basic(t *testing.T) {
	projectname := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: testAccDataGitlabProjectConfigByPathWithNamespace(projectname),
				Check: testAccDataSourceGitlabProject("gitlab_project.test", "data.gitlab_project.foo",
					[]string{"id", "name", "path", "visibility", "description"}),
			},
			{
				Config: testAccDataGitlabProjectConfig(projectname),
				Check: testAccDataSourceGitlabProject("gitlab_project.test", "data.gitlab_project.foo",
					[]string{"id", "name", "path", "visibility", "description"}),
			},
			{
				SkipFunc: testutil.IsRunningInCE,
				Config:   testAccDataGitlabProjectConfigPushRules(projectname),
				Check: testAccDataSourceGitlabProject("gitlab_project.test", "data.gitlab_project.foo",
					[]string{"push_rules.0.author_email_regex"}),
			},
		},
	})
}

func TestAccDataGitlabProject_withoutPushRulesAccess(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testUser := testutil.CreateUsers(t, 1)[0]
	testToken := testutil.CreatePersonalAccessToken(t, testUser)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				// lintignore:AT004  // we need the provider configuration here
				Config: fmt.Sprintf(`
				provider "gitlab" {
				  token = "%s"
				}

				data "gitlab_project" "test" {
				  path_with_namespace = "%s"
				}
				`, testToken.Token, testProject.PathWithNamespace),
				Check: resource.TestCheckResourceAttr("data.gitlab_project.test", "id", fmt.Sprintf("%d", testProject.ID)),
			},
		},
	})
}

func TestAccDataGitlabProject_sharedWithGroup(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testGroup := testutil.CreateGroups(t, 1)[0]
	testutil.ProjectShareGroup(t, testProject.ID, testGroup.ID)

	projectData, _, err := testutil.TestGitlabClient.Projects.GetProject(testProject.ID, nil)
	if err != nil {
		t.Fatalf("could not refresh %v project's updated data, %v", testProject.PathWithNamespace, err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project" "test" {
					id = "%d"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project.test",
						"shared_with_groups.#",
						"1"),
					resource.TestCheckResourceAttr(
						"data.gitlab_project.test",
						"shared_with_groups.0.group_id",
						fmt.Sprintf("%d", projectData.SharedWithGroups[0].GroupID)),
					resource.TestCheckResourceAttr(
						"data.gitlab_project.test",
						"shared_with_groups.0.group_full_path",
						projectData.SharedWithGroups[0].GroupFullPath),
					resource.TestCheckResourceAttr(
						"data.gitlab_project.test",
						"shared_with_groups.0.group_access_level",
						fmt.Sprintf("%d", projectData.SharedWithGroups[0].GroupAccessLevel)),
					resource.TestCheckResourceAttr(
						"data.gitlab_project.test",
						"shared_with_groups.0.group_name",
						projectData.SharedWithGroups[0].GroupName),
				),
			},
		},
	})
}

func TestAccDataGitlabProject_pathWithNamespaceAsIdExpectError(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project" "test" {
					id = "%s"
				}
				`, testProject.PathWithNamespace),
				ExpectError: regexp.MustCompile("`id` must be an integer string and not a path."),
			},
		},
	})
}

// Create a test that populates the CI Restrict Pipeline value using testUtil,
// then uses a terraform `gitlab_project` datasource to read and validate that it matches
func TestAccDataGitlabProject_CIRestrictPipeline(t *testing.T) {
	// Requires EE
	testutil.SkipIfCE(t)

	// Requires GitLab 16.8 or greater
	testutil.RunIfAtLeast(t, "16.8")

	// Create a new project using testutil, and update it's pipelines cancellation
	// to "developer"
	client := testutil.TestGitlabClient
	project := testutil.CreateProject(t)
	var devAccessLevel gitlab.AccessControlValue = "developer"
	_, _, err := client.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{
		CIRestrictPipelineCancellationRole: &devAccessLevel,
	})
	if err != nil {
		t.Fatalf("Error updating project: %v", err)
	}

	// Create the terraform test
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					 data "gitlab_project" "this" {
						id = %d
					 }
					`, project.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project.this", "ci_restrict_pipeline_cancellation_role", "developer"),
				),
			},
		},
	})
}

// Create a test that populates the CI Pipeline variables minimum
// override role value using testUtil, then uses a terraform
// `gitlab_project` datasource to read and validate that it matches
func TestAccDataGitlabProject_CIPipelineVariablesMinimumOverrideRole(t *testing.T) {
	// Requires GitLab 17.1 or greater
	testutil.RunIfAtLeast(t, "17.1")

	// Create a new project using testutil, and update it's pipelines cancellation
	// to "developer"
	client := testutil.TestGitlabClient
	project := testutil.CreateProject(t)
	role := gitlab.CIPipelineVariablesNoOneAllowedRole
	_, _, err := client.Projects.EditProject(project.ID, &gitlab.EditProjectOptions{
		CIPipelineVariablesMinimumOverrideRole: &role,
	})
	if err != nil {
		t.Fatalf("Error updating project: %v", err)
	}

	// Create the terraform test
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					 data "gitlab_project" "this" {
						id = %d
					 }
					`, project.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project.this", "ci_pipeline_variables_minimum_override_role", "no_one_allowed"),
				),
			},
		},
	})
}

func testAccDataSourceGitlabProject(resourceName, dataSourceName string, testAttributes []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		project := s.RootModule().Resources[resourceName]
		projectResource := project.Primary.Attributes

		search := s.RootModule().Resources[dataSourceName]
		searchResource := search.Primary.Attributes

		if searchResource["id"] == "" {
			return fmt.Errorf("Expected to get a project ID from Gitlab")
		}

		for _, attribute := range testAttributes {
			if searchResource[attribute] != projectResource[attribute] {
				return fmt.Errorf("Expected the project %s to be: %s, but got: %s", attribute, projectResource[attribute], searchResource[attribute])
			}
		}
		return nil
	}
}

func testAccDataGitlabProjectConfig(projectname string) string {
	return fmt.Sprintf(`
resource "gitlab_project" "test"{
	name = "%s"
	path = "%s"
	description = "Terraform acceptance tests"
	visibility_level = "public"
}

data "gitlab_project" "foo" {
	id = "${gitlab_project.test.id}"
}
	`, projectname, projectname)
}

func testAccDataGitlabProjectConfigByPathWithNamespace(projectname string) string {
	return fmt.Sprintf(`
resource "gitlab_project" "test"{
	name = "%s"
	path = "%s"
	description = "Terraform acceptance tests"
	visibility_level = "public"
}

data "gitlab_project" "foo" {
	path_with_namespace = gitlab_project.test.path_with_namespace
}
	`, projectname, projectname)
}

func testAccDataGitlabProjectConfigPushRules(projectName string) string {
	return fmt.Sprintf(`
resource "gitlab_project" "test"{
	name = "%[1]s"
	path = "%[1]s"
	description = "Terraform acceptance tests"
	visibility_level = "public"
    push_rules {
        author_email_regex = "foo"
    }
}

data "gitlab_project" "foo" {
	id = gitlab_project.test.id
}
	`, projectName)
}
