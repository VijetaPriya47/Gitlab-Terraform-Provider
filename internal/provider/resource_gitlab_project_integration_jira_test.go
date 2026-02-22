//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
)

func TestAccGitlabProjectIntegrationJira_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with basic configuration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "jira" {
						project                         = %d
						url                             = "https://test.com"
						username                        = "user1"
						password                        = "mypass"
						commit_events                   = true
						merge_requests_events           = false
						jira_issue_transition_automatic = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "username", "user1"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "commit_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "jira_issue_transition_automatic", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_jira.jira", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_jira.jira", "title"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_jira.jira", "created_at"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_project_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
				},
			},
			// Step 3: Update to different values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "jira" {
						project                  = %d
						url                      = "https://testurl.com"
						api_url                  = "https://testurl.com/rest"
						username                 = "user2"
						password                 = "mypass_update"
						jira_issue_transition_id = "3"
						commit_events            = false
						merge_requests_events    = true
						jira_issue_regex         = "TEST-[0-9]+"
						issues_enabled           = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "url", "https://testurl.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "api_url", "https://testurl.com/rest"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "username", "user2"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "password", "mypass_update"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "jira_issue_transition_id", "3"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "commit_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "merge_requests_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "jira_issue_regex", "TEST-[0-9]+"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "issues_enabled", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_jira.jira", "updated_at"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_project_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
				},
			},
			// Step 5: Update back to original values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "jira" {
						project                         = %d
						url                             = "https://test.com"
						username                        = "user1"
						password                        = "mypass"
						commit_events                   = true
						merge_requests_events           = false
						jira_issue_transition_automatic = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "username", "user1"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "commit_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "jira_issue_transition_automatic", "true"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_project_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationJira_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with deprecated resource name
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_jira" "jira" {
						project                         = %d
						url                             = "https://test.com"
						username                        = "user1"
						password                        = "mypass"
						commit_events                   = true
						merge_requests_events           = false
						jira_issue_transition_automatic = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "username", "user1"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "commit_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_transition_automatic", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_jira.jira", "id"),
					resource.TestCheckResourceAttrSet("gitlab_integration_jira.jira", "title"),
					resource.TestCheckResourceAttrSet("gitlab_integration_jira.jira", "created_at"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
				},
			},
			// Step 3: Update to different values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_jira" "jira" {
						project                  = %d
						url                      = "https://testurl.com"
						api_url                  = "https://testurl.com/rest"
						username                 = "user2"
						password                 = "mypass_update"
						jira_issue_transition_id = "3"
						commit_events            = false
						merge_requests_events    = true
						jira_issue_regex         = "TEST-[0-9]+"
						issues_enabled           = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "url", "https://testurl.com"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "api_url", "https://testurl.com/rest"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "username", "user2"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "password", "mypass_update"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_transition_id", "3"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "commit_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "merge_requests_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_regex", "TEST-[0-9]+"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "issues_enabled", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_jira.jira", "updated_at"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
				},
			},
			// Step 5: Update back to original values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_jira" "jira" {
						project                         = %d
						url                             = "https://test.com"
						username                        = "user1"
						password                        = "mypass"
						commit_events                   = true
						merge_requests_events           = false
						jira_issue_transition_automatic = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "username", "user1"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "commit_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_transition_automatic", "true"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationJira_projectKey(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with project_keys
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "jira" {
						project               = %d
						url                   = "https://test.com"
						username              = "user1"
						password              = "mypass"
						project_keys          = ["TEST"]
						commit_events         = true
						merge_requests_events = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "project_keys.#", "1"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "project_keys.0", "TEST"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_project_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationJira_authType_basicAuth(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with Basic Auth
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "jira" {
						project               = %d
						url                   = "https://test.com"
						jira_auth_type        = 0
						username              = "user1"
						password              = "mypass"
						commit_events         = true
						merge_requests_events = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "jira_auth_type", "0"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "username", "user1"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "commit_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "merge_requests_events", "false"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_project_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationJira_authType_tokenAuth(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with Token Auth
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "jira" {
						project                = %d
						url                    = "https://test.com"
						jira_auth_type         = 1
						password               = "mypass"
						use_inherited_settings = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "jira_auth_type", "1"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_project_integration_jira.jira", "use_inherited_settings", "false"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_project_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationJira_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with SDK provider (version 18.9.0)
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.9.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "jira" {
						project  = %d
						url      = "https://test.com"
						username = "user1"
						password = "mypass"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_jira.jira", "id"),
				),
			},
			// Step 2: Migrate to Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "jira" {
						project  = %d
						url      = "https://test.com"
						username = "user1"
						password = "mypass"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_jira.jira", "id"),
				),
			},
			// Step 3: Import verification with Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_integration_jira.jira",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
		},
	})
}

// TestAcc_GitlabProjectIntegrationJira_stateMove verifies that the moved block works
// when migrating from gitlab_integration_jira to gitlab_project_integration_jira.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAcc_GitlabProjectIntegrationJira_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create a Jira integration using the old resource
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_jira" "old" {
						project  = %d
						url      = "https://test.com"
						username = "user1"
						password = "mypass"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_jira.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jira" "new" {
						project  = %d
						url      = "https://test.com"
						username = "user1"
						password = "mypass"
					}

					moved {
						from = gitlab_integration_jira.old
						to   = gitlab_project_integration_jira.new
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_jira.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:            "gitlab_project_integration_jira.new",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationJiraDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_jira" && rs.Type != "gitlab_project_integration_jira" {
			continue
		}

		project := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Services.GetJiraService(project)
		if err == nil {
			if service != nil && service.Active {
				return fmt.Errorf("Jira integration for project %s is still active", project)
			}
		}
		if !api.Is404(err) && err != nil {
			return err
		}
	}
	return nil
}
