//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabIntegrationExternalWiki_backwardsCompatibleToService(t *testing.T) {
	testProject := testutil.CreateProject(t)

	var externalWikiService gitlab.ExternalWikiService

	externalWikiURL1 := "http://mynumberonewiki.com"
	externalWikiResourceName := "gitlab_service_external_wiki.this"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabServiceExternalWikiDestroy,
		Steps: []resource.TestStep{
			// Create an External Wiki service
			{
				Config: fmt.Sprintf(`
				resource "gitlab_service_external_wiki" "this" {
					project           = %[1]d
					external_wiki_url = "%[2]s"
				}
				`, testProject.ID, externalWikiURL1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationExternalWikiExists(externalWikiResourceName, &externalWikiService),
					resource.TestCheckResourceAttr(externalWikiResourceName, "external_wiki_url", externalWikiURL1),
					resource.TestCheckResourceAttr(externalWikiResourceName, "external_wiki_url", externalWikiURL1),
					resource.TestCheckResourceAttr(externalWikiResourceName, "active", "true"),
					resource.TestCheckResourceAttrWith(externalWikiResourceName, "created_at", func(value string) error {
						expectedValue := externalWikiService.CreatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_service_external_wiki.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabIntegrationExternalWiki_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	var externalWikiService gitlab.ExternalWikiService

	externalWikiURL1 := "http://mynumberonewiki.com"
	externalWikiURL2 := "http://mynumbertwowiki.com"
	externalWikiResourceName := "gitlab_integration_external_wiki.this"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabServiceExternalWikiDestroy,
		Steps: []resource.TestStep{
			// Create an External Wiki service
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_external_wiki" "this" {
						project           = %[1]d
						external_wiki_url = "%[2]s"
					}
				`, testProject.ID, externalWikiURL1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationExternalWikiExists(externalWikiResourceName, &externalWikiService),
					resource.TestCheckResourceAttr(externalWikiResourceName, "external_wiki_url", externalWikiURL1),
					resource.TestCheckResourceAttr(externalWikiResourceName, "external_wiki_url", externalWikiURL1),
					resource.TestCheckResourceAttr(externalWikiResourceName, "active", "true"),
					resource.TestCheckResourceAttrWith(externalWikiResourceName, "created_at", func(value string) error {
						expectedValue := externalWikiService.CreatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_integration_external_wiki.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the External Wiki service
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_external_wiki" "this" {
						project           = %[1]d
						external_wiki_url = "%[2]s"
					}
				`, testProject.ID, externalWikiURL2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationExternalWikiExists(externalWikiResourceName, &externalWikiService),
					resource.TestCheckResourceAttr(externalWikiResourceName, "external_wiki_url", externalWikiURL2),
					resource.TestCheckResourceAttrWith(externalWikiResourceName, "created_at", func(value string) error {
						expectedValue := externalWikiService.CreatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith(externalWikiResourceName, "updated_at", func(value string) error {
						expectedValue := externalWikiService.UpdatedAt.Format(time.RFC3339)
						if value != expectedValue {
							return fmt.Errorf("should be equal to %s", expectedValue)
						}
						return nil
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_integration_external_wiki.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the External Wiki service to get back to previous settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_external_wiki" "this" {
						project           = %[1]d
						external_wiki_url = "%[2]s"
					}
				`, testProject.ID, externalWikiURL1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationExternalWikiExists(externalWikiResourceName, &externalWikiService),
					resource.TestCheckResourceAttr(externalWikiResourceName, "external_wiki_url", externalWikiURL1),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_integration_external_wiki.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabIntegrationExternalWikiExists(resourceIdentifier string, service *gitlab.ExternalWikiService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceIdentifier]
		if !ok {
			return fmt.Errorf("Not Found: %s", resourceIdentifier)
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return fmt.Errorf("No project ID is set")
		}

		externalWikiService, _, err := testutil.TestGitlabClient.Services.GetExternalWikiService(project)
		if err != nil {
			return fmt.Errorf("External Wiki Integration does not exist in project %s: %v", project, err)
		}
		*service = *externalWikiService

		return nil
	}
}

func testAccCheckGitlabServiceExternalWikiDestroy(s *terraform.State) error {
	var project string

	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_integration_external_wiki" {
			project = rs.Primary.ID

			externalWikiService, _, err := testutil.TestGitlabClient.Services.GetExternalWikiService(project)
			if err == nil {
				if externalWikiService != nil && externalWikiService.Active != false {
					return fmt.Errorf("[ERROR] External Wiki Integration %v still exists", project)
				}
			} else {
				return err
			}
		}
	}
	return nil
}
