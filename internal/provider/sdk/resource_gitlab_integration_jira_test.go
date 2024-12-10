//go:build acceptance
// +build acceptance

package sdk

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabIntegrationJira_basic(t *testing.T) {
	var jiraService gitlab.JiraService
	jiraResourceName := "gitlab_integration_jira.jira"

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
				  comment_on_event_enabled = false
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists(jiraResourceName, &jiraService),
					resource.TestCheckResourceAttr(jiraResourceName, "url", "https://test.com"),
					resource.TestCheckResourceAttr(jiraResourceName, "username", "user1"),
					resource.TestCheckResourceAttr(jiraResourceName, "password", "mypass"),
					resource.TestCheckResourceAttr(jiraResourceName, "commit_events", "true"),
					resource.TestCheckResourceAttr(jiraResourceName, "merge_requests_events", "false"),
					resource.TestCheckResourceAttr(jiraResourceName, "comment_on_event_enabled", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      jiraResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
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
				  comment_on_event_enabled = true
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists(jiraResourceName, &jiraService),
					resource.TestCheckResourceAttr(jiraResourceName, "url", "https://testurl.com"),
					resource.TestCheckResourceAttr(jiraResourceName, "api_url", "https://testurl.com/rest"),
					resource.TestCheckResourceAttr(jiraResourceName, "username", "user2"),
					resource.TestCheckResourceAttr(jiraResourceName, "password", "mypass_update"),
					resource.TestCheckResourceAttr(jiraResourceName, "jira_issue_transition_id", "3"),
					resource.TestCheckResourceAttr(jiraResourceName, "commit_events", "false"),
					resource.TestCheckResourceAttr(jiraResourceName, "merge_requests_events", "true"),
					resource.TestCheckResourceAttr(jiraResourceName, "comment_on_event_enabled", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      jiraResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
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
				  comment_on_event_enabled = false
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists(jiraResourceName, &jiraService),
					resource.TestCheckResourceAttr(jiraResourceName, "url", "https://test.com"),
					resource.TestCheckResourceAttr(jiraResourceName, "api_url", "https://testurl.com/rest"),
					resource.TestCheckResourceAttr(jiraResourceName, "username", "user1"),
					resource.TestCheckResourceAttr(jiraResourceName, "password", "mypass"),
					resource.TestCheckResourceAttr(jiraResourceName, "commit_events", "true"),
					resource.TestCheckResourceAttr(jiraResourceName, "merge_requests_events", "false"),
					resource.TestCheckResourceAttr(jiraResourceName, "comment_on_event_enabled", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      jiraResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
		},
	})
}

func TestAcc_GitlabIntegrationJira_projectKey(t *testing.T) {
	var jiraService gitlab.JiraService
	jiraResourceName := "gitlab_integration_jira.jira"
	project := testutil.CreateProject(t)

	importSkipAttributes := []string{
		"password",
	}

	isVersionUnder17, err := api.IsGitLabVersionLessThan(context.Background(), testutil.TestGitlabClient, "17.0")()
	if err != nil {
		t.Fatal("Failed to read GitLab version")
	}

	// We need to skip import validation on project key when we're below 17.0
	if isVersionUnder17 {
		importSkipAttributes = append(importSkipAttributes, "project_key")
	}

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
					  comment_on_event_enabled = false
					}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists(jiraResourceName, &jiraService),
				),
			},
			// Verify Import
			{
				ResourceName:            jiraResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importSkipAttributes,
			},
		},
	})
}

func TestAcc_GitlabIntegrationJira_authType_basicAuth(t *testing.T) {
	var jiraService gitlab.JiraService
	jiraResourceName := "gitlab_service_jira.jira"

	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create a project and a jira service
			{
				Config: fmt.Sprintf(`
				resource "gitlab_service_jira" "jira" {
				  project  = "%d"
				  url      = "https://test.com"
					jira_auth_type = 0
				  username = "user1"
				  password = "mypass"
				  commit_events = true
				  merge_requests_events    = false
				  comment_on_event_enabled = false
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists(jiraResourceName, &jiraService),
					resource.TestCheckResourceAttr(jiraResourceName, "url", "https://test.com"),
					resource.TestCheckResourceAttr(jiraResourceName, "jira_auth_type", "0"),
					resource.TestCheckResourceAttr(jiraResourceName, "username", "user1"),
					resource.TestCheckResourceAttr(jiraResourceName, "password", "mypass"),
					resource.TestCheckResourceAttr(jiraResourceName, "commit_events", "true"),
					resource.TestCheckResourceAttr(jiraResourceName, "merge_requests_events", "false"),
					resource.TestCheckResourceAttr(jiraResourceName, "comment_on_event_enabled", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      jiraResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
		},
	})
}

func TestAcc_GitlabIntegrationJira_authType_tokenAuth(t *testing.T) {
	var jiraService gitlab.JiraService
	jiraResourceName := "gitlab_service_jira.jira"

	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create a project and a jira service
			{
				Config: fmt.Sprintf(`
				resource "gitlab_service_jira" "jira" {
				  project  = "%d"
				  url      = "https://test.com"
					jira_auth_type = 1
				  password = "mypass"
          use_inherited_settings = false
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists(jiraResourceName, &jiraService),
					resource.TestCheckResourceAttr(jiraResourceName, "url", "https://test.com"),
					resource.TestCheckResourceAttr(jiraResourceName, "jira_auth_type", "1"),
					resource.TestCheckResourceAttr(jiraResourceName, "password", "mypass"),
					resource.TestCheckResourceAttr(jiraResourceName, "use_inherited_settings", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      jiraResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
		},
	})
}

func TestAcc_GitlabIntegrationJira_backwardsCompatibility(t *testing.T) {
	var jiraService gitlab.JiraService
	jiraResourceName := "gitlab_service_jira.jira"

	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create a project and a jira service
			{
				Config: fmt.Sprintf(`
				resource "gitlab_service_jira" "jira" {
				  project  = "%d"
				  url      = "https://test.com"
				  username = "user1"
				  password = "mypass"
				  commit_events = true
				  merge_requests_events    = false
				  comment_on_event_enabled = false
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationJiraExists(jiraResourceName, &jiraService),
					resource.TestCheckResourceAttr(jiraResourceName, "url", "https://test.com"),
					resource.TestCheckResourceAttr(jiraResourceName, "username", "user1"),
					resource.TestCheckResourceAttr(jiraResourceName, "password", "mypass"),
					resource.TestCheckResourceAttr(jiraResourceName, "commit_events", "true"),
					resource.TestCheckResourceAttr(jiraResourceName, "merge_requests_events", "false"),
					resource.TestCheckResourceAttr(jiraResourceName, "comment_on_event_enabled", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      jiraResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
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
