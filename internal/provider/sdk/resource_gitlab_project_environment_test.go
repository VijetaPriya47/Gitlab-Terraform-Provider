//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectEnvironment_basic(t *testing.T) {
	rInt := acctest.RandInt()
	testProject := testutil.CreateProject(t)

	env1 := gitlab.Environment{
		Name: fmt.Sprintf("ProjectEnvironment-%d", rInt),
	}

	env2 := gitlab.Environment{
		Name:        fmt.Sprintf("ProjectEnvironment-%d", rInt),
		ExternalURL: "https://example.com",
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectEnvironmentDestroy,
		Steps: []resource.TestStep{
			// Create an Environment with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "this" {
						project     = %d
						name        = "ProjectEnvironment-%d"
						description = "A test project"	
					
						stop_before_destroy = true
					}
				`, testProject.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectEnvironmentExists("gitlab_project_environment.this", &env1),
					testAccCheckGitlabProjectEnvironmentAttributes(&env1, &testAccGitlabProjectEnvironmentExpectedAttributes{
						Name:        fmt.Sprintf("ProjectEnvironment-%d", rInt),
						State:       "available",
						Tier:        "other",
						Description: "A test project",
					}),
					resource.TestCheckResourceAttrWith("gitlab_project_environment.this", "created_at", func(value string) error {
						expectedValue := env1.CreatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_environment.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
			// Update the Environment
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "this" {
						project      = %d
						name         = "ProjectEnvironment-%d"
						external_url = "https://example.com"
						tier         = "production"
						description  = "A different description"
					}
				`, testProject.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectEnvironmentExists("gitlab_project_environment.this", &env2),
					testAccCheckGitlabProjectEnvironmentAttributes(&env2, &testAccGitlabProjectEnvironmentExpectedAttributes{
						Name:        fmt.Sprintf("ProjectEnvironment-%d", rInt),
						State:       "available",
						ExternalURL: "https://example.com",
						Tier:        "production",
						Description: "A different description",
					}),
					resource.TestCheckResourceAttrWith("gitlab_project_environment.this", "created_at", func(value string) error {
						expectedValue := env2.CreatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("gitlab_project_environment.this", "updated_at", func(value string) error {
						expectedValue := env2.UpdatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_environment.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
			// Update the Environment to get back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "this" {
						project     = %d
						name        = "ProjectEnvironment-%d"
						description = "A test project"
					
						stop_before_destroy = true
					}
				`, testProject.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectEnvironmentExists("gitlab_project_environment.this", &env1),
					testAccCheckGitlabProjectEnvironmentAttributes(&env1, &testAccGitlabProjectEnvironmentExpectedAttributes{
						Name:        fmt.Sprintf("ProjectEnvironment-%d", rInt),
						State:       "available",
						Tier:        "production",
						Description: "A test project",
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_environment.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
		},
	})
}

func TestAccGitlabProjectEnvironment_stopBeforeDestroyDisabled(t *testing.T) {
	rInt := acctest.RandInt()
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectEnvironmentDestroy,
		Steps: []resource.TestStep{
			// Create environment with `stop_before_destroy = false`
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "this" {
						project = %d
						name    = "ProjectEnvironment-%d"
						
						stop_before_destroy = false
					}
				`, testProject.ID, rInt),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "this" {
						project = %d
						name    = "ProjectEnvironment-%d"
						
						stop_before_destroy = false
					}
				`, testProject.ID, rInt),
				ExpectError: regexp.MustCompile("Environment must be in a stopped state before deletion"),
				Destroy:     true,
			},
			// Update stop flag
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "this" {
						project = %d
						name    = "ProjectEnvironment-%d"
					
						stop_before_destroy = true
					}
				`, testProject.ID, rInt),
			},
		},
	})
}

func TestAccGitlabProjectEnvironment_ClusterAgent(t *testing.T) {
	testName := acctest.RandString(10)
	testProject := testutil.CreateProject(t)
	testAgents := testutil.CreateClusterAgents(t, testProject.ID, 2)
	testAgent1 := testAgents[0]
	testAgent2 := testAgents[1]
	testutil.SetupUserAccess(t, testProject, testAgent1)
	testutil.SetupUserAccess(t, testProject, testAgent2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "test" {
						project              = %d
						name                 = "%s"
						cluster_agent_id     = %d
						kubernetes_namespace = "default"
						flux_resource_path   = "some-path"

						stop_before_destroy = true
					}
				`, testProject.ID, testName, testAgent1.ID),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_environment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
			// Clear agent related attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "test" {
						project              = %d
						name                 = "%s"

						stop_before_destroy = true
					}
				`, testProject.ID, testName),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_environment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
			// Re-assign cluster agent
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "test" {
						project              = %d
						name                 = "%s"
						cluster_agent_id     = %d

						stop_before_destroy = true
					}
				`, testProject.ID, testName, testAgent2.ID),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_environment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
			// Re-assign kubernetes namespace
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "test" {
						project              = %d
						name                 = "%s"
						cluster_agent_id     = %d
						kubernetes_namespace = "default"

						stop_before_destroy = true
					}
				`, testProject.ID, testName, testAgent2.ID),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_environment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
			// Re-assign flux resource path
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "test" {
						project              = %d
						name                 = "%s"
						cluster_agent_id     = %d
						kubernetes_namespace = "default"
						flux_resource_path   = "some-path"

						stop_before_destroy = true
					}
				`, testProject.ID, testName, testAgent1.ID),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_environment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
		},
	})
}

func TestAccGitlabProjectEnvironment_AutoStopSetting(t *testing.T) {
	testName := acctest.RandString(10)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "auto_stop" {
						project              = %d
						name                 = "%s"
						auto_stop_setting    = "always"

						stop_before_destroy = true
					}
				`, testProject.ID, testName),
			},
			{
				ResourceName:            "gitlab_project_environment.auto_stop",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_environment" "auto_stop" {
						project              = %d
						name                 = "%s"
						auto_stop_setting    = "with_action"

						stop_before_destroy = true
					}
				`, testProject.ID, testName),
			},
			{
				ResourceName:            "gitlab_project_environment.auto_stop",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"stop_before_destroy"},
			},
		},
	})
}

func testAccCheckGitlabProjectEnvironmentExists(n string, env *gitlab.Environment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project, environment, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error in Splitting Project ID and Environment Name")
		}

		environmentID, err := strconv.Atoi(environment)
		if err != nil {
			return fmt.Errorf("error converting environment ID to int: %v", err)
		}

		if e, _, err := testutil.TestGitlabClient.Environments.GetEnvironment(project, environmentID); err != nil {
			return err
		} else {
			*env = *e
		}
		return nil
	}
}

type testAccGitlabProjectEnvironmentExpectedAttributes struct {
	Name        string
	ExternalURL string
	State       string
	Tier        string
	Description string
}

func testAccCheckGitlabProjectEnvironmentAttributes(env *gitlab.Environment, want *testAccGitlabProjectEnvironmentExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if env.Name != want.Name {
			return fmt.Errorf("got name %q; want %q", env.Name, want.Name)
		}

		if env.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", env.Description, want.Description)
		}

		if env.ExternalURL != want.ExternalURL {
			return fmt.Errorf("got external URL %q; want %q", env.ExternalURL, want.ExternalURL)
		}

		if env.Tier != want.Tier {
			return fmt.Errorf("got tier %q; want %q", env.Tier, want.Tier)
		}

		if env.State != want.State {
			return fmt.Errorf("got State %q; want %q", env.State, want.State)
		}

		return nil
	}
}

func testAccCheckGitlabProjectEnvironmentDestroy(s *terraform.State) error {
	var project string
	var environmentIDString string
	var environmentIDInt int
	var err error
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "gitlab_project":
			project = rs.Primary.ID
		case "gitlab_project_environment":
			project, environmentIDString, err = utils.ParseTwoPartID(rs.Primary.ID)
			if err != nil {
				return fmt.Errorf("[ERROR] cannot get project and environmentID from input: %v", rs.Primary.ID)
			}

			environmentIDInt, err = strconv.Atoi(environmentIDString)
			if err != nil {
				return fmt.Errorf("[ERROR] cannot convert environment ID to int: %v", err)
			}
		}
	}

	env, _, err := testutil.TestGitlabClient.Environments.GetEnvironment(project, environmentIDInt)
	if err == nil {
		if env != nil {
			return fmt.Errorf("[ERROR] project Environment %v still exists", environmentIDInt)
		}
	} else {
		if !api.Is404(err) {
			return err
		}
	}

	return nil
}
