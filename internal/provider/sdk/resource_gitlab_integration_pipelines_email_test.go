//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabIntegrationPipelinesEmail_basic(t *testing.T) {
	var pipelinesEmailService gitlab.PipelinesEmailService
	rInt := acctest.RandInt()
	pipelinesEmailResourceName := "gitlab_integration_pipelines_email.email"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationPipelinesEmailDestroy,
		Steps: []resource.TestStep{
			// Create a project and a pipelines email integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name         = "foo-%d"
						description  = "Terraform acceptance tests"
					}
					
					resource "gitlab_integration_pipelines_email" "email" {
						project    = gitlab_project.foo.id
						recipients = ["test@example.com"]
					}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationPipelinesEmailExists(pipelinesEmailResourceName, &pipelinesEmailService),
					testRecipients(&pipelinesEmailService, []string{"test@example.com"}),
					resource.TestCheckResourceAttr(pipelinesEmailResourceName, "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr(pipelinesEmailResourceName, "branches_to_be_notified", "default"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_pipelines_email.email",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the pipelinesEmail integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name         = "foo-%d"
						description  = "Terraform acceptance tests"
					}
					
					resource "gitlab_integration_pipelines_email" "email" {
						project                      = gitlab_project.foo.id
						recipients                   = ["test@example.com", "test2@example.com"]
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
					}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationPipelinesEmailExists(pipelinesEmailResourceName, &pipelinesEmailService),
					testRecipients(&pipelinesEmailService, []string{"test@example.com", "test2@example.com"}),
					resource.TestCheckResourceAttr(pipelinesEmailResourceName, "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr(pipelinesEmailResourceName, "branches_to_be_notified", "all"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_pipelines_email.email",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the pipelinesEmail integration to get back to previous settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name         = "foo-%d"
						description  = "Terraform acceptance tests"
					}
					
					resource "gitlab_integration_pipelines_email" "email" {
						project    = gitlab_project.foo.id
						recipients = ["test@example.com"]
					}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationPipelinesEmailExists(pipelinesEmailResourceName, &pipelinesEmailService),
					testRecipients(&pipelinesEmailService, []string{"test@example.com"}),
					resource.TestCheckResourceAttr(pipelinesEmailResourceName, "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr(pipelinesEmailResourceName, "branches_to_be_notified", "default"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_pipelines_email.email",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabIntegrationPipelinesEmail_backwardsCompatbility(t *testing.T) {
	var pipelinesEmailService gitlab.PipelinesEmailService
	rInt := acctest.RandInt()
	pipelinesEmailResourceName := "gitlab_service_pipelines_email.email"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationPipelinesEmailDestroy,
		Steps: []resource.TestStep{
			// Create a project and a pipelines email integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
					name         = "foo-%d"
					description  = "Terraform acceptance tests"
				}
				
				resource "gitlab_service_pipelines_email" "email" {
					project                      = gitlab_project.foo.id
					recipients                   = ["test@example.com"]
				}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationPipelinesEmailExists(pipelinesEmailResourceName, &pipelinesEmailService),
					testRecipients(&pipelinesEmailService, []string{"test@example.com"}),
					resource.TestCheckResourceAttr(pipelinesEmailResourceName, "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr(pipelinesEmailResourceName, "branches_to_be_notified", "default"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_service_pipelines_email.email",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabIntegrationPipelinesEmailExists(n string, service *gitlab.PipelinesEmailService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return fmt.Errorf("No project ID is set")
		}
		pipelinesEmailService, _, err := testutil.TestGitlabClient.Services.GetPipelinesEmailService(project)
		if err != nil {
			return fmt.Errorf("PipelinesEmail integration does not exist in project %s: %v", project, err)
		}
		*service = *pipelinesEmailService

		return nil
	}
}

func testRecipients(service *gitlab.PipelinesEmailService, expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res_string := service.Properties.Recipients
		res := strings.Split(res_string, ",")
		if len(res) != len(expected) {
			return fmt.Errorf("'recipients' field does not have the correct size expected: %d, found: %d", len(expected), len(res))
		}
		sort.Strings(res)
		sort.Strings(expected)
		for i, r := range res {
			e := expected[i]
			if r != e {
				return fmt.Errorf("expected: %s, found: %s", r, e)
			}

		}
		return nil
	}
}

func testAccCheckGitlabIntegrationPipelinesEmailDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}

		gotRepo, resp, err := testutil.TestGitlabClient.Projects.GetProject(rs.Primary.ID, nil)
		if err == nil {
			if gotRepo != nil && fmt.Sprintf("%d", gotRepo.ID) == rs.Primary.ID {
				if gotRepo.MarkedForDeletionAt == nil {
					return fmt.Errorf("Repository still exists")
				}
			}
		}
		if resp != nil && resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}
