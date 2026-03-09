//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectPackageDependencyProxy_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectPackageDependencyProxyDestroy,
		Steps: []resource.TestStep{
			// Create with enabled and URL
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_package_dependency_proxy" "test" {
					project                        = "%d"
					enabled                        = true
					maven_external_registry_url    = "https://repo.maven.apache.org/maven2/"
				}
			`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_package_dependency_proxy.test", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_package_dependency_proxy.test", "maven_external_registry_url", "https://repo.maven.apache.org/maven2/"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_package_dependency_proxy.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"maven_external_registry_password", // doesn't come back in the response
					"project",                          // can be path or ID, can't be verified as a result
				},
			},
			// Update to include credentials
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_package_dependency_proxy" "test" {
					project                           = "%d"
					enabled                           = true
					maven_external_registry_url       = "https://repo.maven.apache.org/maven2/"
					maven_external_registry_username  = "testuser"
					maven_external_registry_password  = "testpassword"
				}
			`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_package_dependency_proxy.test", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_package_dependency_proxy.test", "maven_external_registry_username", "testuser"),
				),
			},
			// Verify import after update
			{
				ResourceName:      "gitlab_project_package_dependency_proxy.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"maven_external_registry_password",
					"project",
				},
			},
			// Update to change URL
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_package_dependency_proxy" "test" {
					project                           = "%d"
					enabled                           = true
					maven_external_registry_url       = "https://jcenter.bintray.com/"
					maven_external_registry_username  = "newuser"
					maven_external_registry_password  = "newpassword"
				}
			`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_package_dependency_proxy.test", "maven_external_registry_url", "https://jcenter.bintray.com/"),
					resource.TestCheckResourceAttr("gitlab_project_package_dependency_proxy.test", "maven_external_registry_username", "newuser"),
				),
			},
		},
	})
}

func TestAccGitlabProjectPackageDependencyProxy_validation(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectPackageDependencyProxyDestroy,
		Steps: []resource.TestStep{
			// Username without password should fail
			{
				Config: `
				resource "gitlab_project_package_dependency_proxy" "test" {
					project                          = "1234"
					enabled                          = true
					maven_external_registry_url      = "https://repo.maven.apache.org/maven2/"
					maven_external_registry_username = "testuser"
				}`,
				ExpectError: regexp.MustCompile("maven_external_registry_password must be set"),
			},
			// Password without username should fail
			{
				Config: `
				resource "gitlab_project_package_dependency_proxy" "test" {
					project                          = "1234"
					enabled                          = true
					maven_external_registry_url      = "https://repo.maven.apache.org/maven2/"
					maven_external_registry_password = "testpassword"
				}`,
				ExpectError: regexp.MustCompile("maven_external_registry_username must be set"),
			},
		},
	})
}

func testAccCheckGitlabProjectPackageDependencyProxyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_package_dependency_proxy" {
			continue
		}

		projectID := rs.Primary.ID
		project, _, err := testutil.TestGitlabClient.Projects.GetProject(projectID, nil)
		if err != nil {
			// Project was deleted, which is fine
			if api.Is404(err) {
				return nil
			}
			return err
		}

		graphQLcall := fmt.Sprintf(`
query {
  project(fullPath: "%s") {
    dependencyProxyPackagesSetting {
      enabled
    }
	errors
  }
}
`, project.PathWithNamespace)

		var response *readProjectDependencyProxyPackagesGraphQLResponse
		_, err = testutil.TestGitlabClient.GraphQL.Do(gitlab.GraphQLQuery{Query: graphQLcall}, &response)
		if err != nil {
			return err
		}

		// Check if there are any errors in the response
		if response != nil && len(response.Data.Project.Errors) > 0 {
			return fmt.Errorf("%s", strings.Join(response.Data.Project.Errors, "\n"))
		}

		// proxy is still enabled - should have been disabled
		if response != nil && response.Data.Project.DependencyProxyPackagesSetting.Enabled {
			return fmt.Errorf("Project Package Dependency Proxy still enabled for project %s", project.PathWithNamespace)
		}
	}
	return nil
}
