//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
)

func TestAccGitlabProjectIntegrationExternalWiki_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	externalWikiURL1 := "https://example.com/external-wiki-1"
	externalWikiURL2 := "https://example.com/external-wiki-2"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationExternalWikiDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with external_wiki_url
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_external_wiki" "test" {
						project          = %d
						external_wiki_url = "%s"
					}
				`, testProject.ID, externalWikiURL1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_external_wiki.test", "external_wiki_url", externalWikiURL1),
					resource.TestCheckResourceAttr("gitlab_project_integration_external_wiki.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.test", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.test", "title"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.test", "slug"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.test", "created_at"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_project_integration_external_wiki.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update to externalWikiURL2
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_external_wiki" "test" {
						project          = %d
						external_wiki_url = "%s"
					}
				`, testProject.ID, externalWikiURL2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_external_wiki.test", "external_wiki_url", externalWikiURL2),
					resource.TestCheckResourceAttr("gitlab_project_integration_external_wiki.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.test", "updated_at"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_project_integration_external_wiki.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 5: Update back to externalWikiURL1
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_external_wiki" "test" {
						project          = %d
						external_wiki_url = "%s"
					}
				`, testProject.ID, externalWikiURL1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_external_wiki.test", "external_wiki_url", externalWikiURL1),
					resource.TestCheckResourceAttr("gitlab_project_integration_external_wiki.test", "active", "true"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_project_integration_external_wiki.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectIntegrationExternalWiki_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	externalWikiURL1 := "https://example.com/external-wiki-deprecated-1"
	externalWikiURL2 := "https://example.com/external-wiki-deprecated-2"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationExternalWikiDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with deprecated resource name
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_external_wiki" "test" {
						project          = %d
						external_wiki_url = "%s"
					}
				`, testProject.ID, externalWikiURL1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_external_wiki.test", "external_wiki_url", externalWikiURL1),
					resource.TestCheckResourceAttr("gitlab_integration_external_wiki.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.test", "id"),
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.test", "title"),
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.test", "slug"),
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.test", "created_at"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_integration_external_wiki.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update to externalWikiURL2
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_external_wiki" "test" {
						project          = %d
						external_wiki_url = "%s"
					}
				`, testProject.ID, externalWikiURL2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_external_wiki.test", "external_wiki_url", externalWikiURL2),
					resource.TestCheckResourceAttr("gitlab_integration_external_wiki.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.test", "updated_at"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_integration_external_wiki.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectIntegrationExternalWiki_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	externalWikiURL := "https://example.com/external-wiki-migration"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectIntegrationExternalWikiDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with SDK provider (version 18.8.1)
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.8.1",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_external_wiki" "test" {
						project          = %d
						external_wiki_url = "%s"
					}
				`, testProject.ID, externalWikiURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.test", "id"),
				),
			},
			// Step 2: Migrate to Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_external_wiki" "test" {
						project          = %d
						external_wiki_url = "%s"
					}
				`, testProject.ID, externalWikiURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.test", "id"),
				),
			},
			// Step 3: Import verification with Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_integration_external_wiki.test",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

// TestAcc_GitlabProjectIntegrationExternalWiki_stateMove verifies that the moved block works
// when migrating from gitlab_integration_external_wiki to gitlab_project_integration_external_wiki.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAcc_GitlabProjectIntegrationExternalWiki_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)
	externalWikiURL := "https://example.com/external-wiki"

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationExternalWikiDestroy,
		Steps: []resource.TestStep{
			// Create an External Wiki integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_external_wiki" "old" {
					project           = %d
					external_wiki_url = "%s"
				}
				`, testProject.ID, externalWikiURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.old", "id"),
					resource.TestCheckResourceAttr("gitlab_integration_external_wiki.old", "external_wiki_url", externalWikiURL),
					resource.TestCheckResourceAttr("gitlab_integration_external_wiki.old", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.old", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.old", "title"),
					resource.TestCheckResourceAttrSet("gitlab_integration_external_wiki.old", "slug"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_external_wiki" "new" {
					project           = %d
					external_wiki_url = "%s"
				}

				moved {
					from = gitlab_integration_external_wiki.old
					to   = gitlab_project_integration_external_wiki.new
				}
				`, testProject.ID, externalWikiURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.new", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_external_wiki.new", "external_wiki_url", externalWikiURL),
					resource.TestCheckResourceAttr("gitlab_project_integration_external_wiki.new", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.new", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.new", "title"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_external_wiki.new", "slug"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:      "gitlab_project_integration_external_wiki.new",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationExternalWikiDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_external_wiki" && rs.Type != "gitlab_project_integration_external_wiki" {
			continue
		}

		project := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Services.GetExternalWikiService(project)
		if err == nil {
			if service != nil && service.Active {
				return fmt.Errorf("External Wiki integration for project %s is still active", project)
			}
		}
	}
	return nil
}
