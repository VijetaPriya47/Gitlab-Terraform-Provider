//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabIntegrationJira_basic(t *testing.T) {
	var jiraService gitlab.JiraService
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create a project and a jira service
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_jira" "jira" {
				  project  = "%d"
				  url      = "https://test.com"
				  username = "user1"
				  password = "mypass"
				  commit_events = true
				  merge_requests_events    = false
				  jira_issue_transition_automatic = true
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists("gitlab_integration_jira.jira", &jiraService),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "username", "user1"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "commit_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_transition_automatic", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
					"comment_on_event_enabled", // ignored due to a bug in GitLab 17.9
				},
			},
			// Update the jira service
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_jira" "jira" {
				  project  = "%d"
				  url      = "https://testurl.com"
				  api_url  = "https://testurl.com/rest"
				  username = "user2"
				  password = "mypass_update"
				  jira_issue_transition_id = "3"
				  commit_events = false
				  merge_requests_events    = true
				  jira_issue_regex = "TEST-[0-9]+"
				  issues_enabled = true
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists("gitlab_integration_jira.jira", &jiraService),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "url", "https://testurl.com"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "api_url", "https://testurl.com/rest"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "username", "user2"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "password", "mypass_update"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_transition_automatic", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_transition_id", "3"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "commit_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "merge_requests_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_regex", "TEST-[0-9]+"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "issues_enabled", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
					"comment_on_event_enabled", // ignored due to a bug in GitLab 17.9
				},
			},
			// Update the jira service to get back to previous settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_jira" "jira" {
				  project  = "%d"
				  url      = "https://test.com"
				  username = "user1"
				  password = "mypass"
				  commit_events = true
				  merge_requests_events    = false
				  jira_issue_transition_automatic = true
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists("gitlab_integration_jira.jira", &jiraService),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "api_url", "https://testurl.com/rest"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "username", "user1"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "commit_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "use_inherited_settings", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_regex", ""),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "issues_enabled", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_issue_transition_automatic", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"jira_issue_transition_automatic",
					"comment_on_event_enabled", // ignored due to a bug in GitLab 17.9
				},
			},
		},
	})
}

func TestAcc_GitlabIntegrationJira_projectKey(t *testing.T) {
	var jiraService gitlab.JiraService
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create a project and a jira service
			{
				Config: fmt.Sprintf(
					`resource "gitlab_integration_jira" "jira" {
					  project  = "%d"
					  url      = "https://test.com"
					  username = "user1"
					  password = "mypass"
					  project_key = "TEST"
					  commit_events = true
					  merge_requests_events    = false
					}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists("gitlab_integration_jira.jira", &jiraService),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"comment_on_event_enabled", // ignored due to a bug in GitLab 17.9
				},
			},
		},
	})
}

func TestAcc_GitlabIntegrationJira_authType_basicAuth(t *testing.T) {
	var jiraService gitlab.JiraService
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create a project and a jira service
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_jira" "jira" {
				  project  = "%d"
				  url      = "https://test.com"
					jira_auth_type = 0
				  username = "user1"
				  password = "mypass"
				  commit_events = true
				  merge_requests_events    = false
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists("gitlab_integration_jira.jira", &jiraService),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_auth_type", "0"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "username", "user1"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "commit_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "merge_requests_events", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"comment_on_event_enabled", // ignored due to a bug in GitLab 17.9
				},
			},
		},
	})
}

func TestAcc_GitlabIntegrationJira_authType_tokenAuth(t *testing.T) {
	var jiraService gitlab.JiraService
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create a project and a jira service
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_jira" "jira" {
				  project  = "%d"
				  url      = "https://test.com"
				  jira_auth_type = 1
				  password = "mypass"
                  use_inherited_settings = false
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists("gitlab_integration_jira.jira", &jiraService),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "url", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "jira_auth_type", "1"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "password", "mypass"),
					resource.TestCheckResourceAttr("gitlab_integration_jira.jira", "use_inherited_settings", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_jira.jira",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"comment_on_event_enabled", // ignored due to a bug in GitLab 17.9
				},
			},
		},
	})
}

func testAccCheckGitlabIntegrationJiraExists(n string, service *gitlab.JiraService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return fmt.Errorf("No project ID is set")
		}
		jiraService, _, err := testutil.TestGitlabClient.Services.GetJiraService(project)
		if err != nil {
			return fmt.Errorf("Jira integration does not exist in project %s: %v", project, err)
		}
		*service = *jiraService

		return nil
	}
}

func testAccCheckGitlabIntegrationJiraDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_jira" {
			continue
		}

		project := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Services.GetJiraService(project)
		if err == nil && service.Active {
			return fmt.Errorf("Jira Integration in project %s still exists", project)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
