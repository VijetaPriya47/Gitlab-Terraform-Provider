//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabPagesDomain_basic(t *testing.T) {

	// Set up project environment.
	project := testutil.CreateProject(t)
	domain := fmt.Sprintf("%d.example.com", project.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabPagesDomain_CheckDestroy(project.ID, domain),
		Steps: []resource.TestStep{
			// Create a basic pages domain.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pages_domain" "this" {
					project     = %d
					domain      = "%s"

					auto_ssl_enabled = true
				}`, project.ID, domain),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "project"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "domain"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "auto_ssl_enabled"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "verified"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "url"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_pages_domain.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add optional attributes
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pages_domain" "this" {
					project     = %d
					domain      = "%s"

					auto_ssl_enabled = false
					key              = file("${path.module}/testdata/key.pem")
					certificate      = file("${path.module}/testdata/cert.pem")
				}`, project.ID, domain),

				// Check computed attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "expired"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "certificate"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "key"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_pages_domain.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
			// Replace with a new domain
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pages_domain" "this" {
					project     = %d
					domain      = "replaced-%s"

					auto_ssl_enabled = false
					key              = file("${path.module}/testdata/key.pem")
					certificate      = file("${path.module}/testdata/cert.pem")
				}`, project.ID, domain),

				// Check computed attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "project"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "domain"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "auto_ssl_enabled"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "verified"),
					resource.TestCheckResourceAttrSet("gitlab_pages_domain.this", "url"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_pages_domain.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}

func TestAcc_GitlabPagesDomain_conflictingError(t *testing.T) {

	// Set up project environment.
	project := testutil.CreateProject(t)
	domain := fmt.Sprintf("%d.example.com", project.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabPagesDomain_CheckDestroy(project.ID, domain),
		Steps: []resource.TestStep{
			// Create a basic pages domain.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pages_domain" "this" {
					project     = %d
					domain      = "%s"

					// This will fail on purpose
					auto_ssl_enabled = true
					certificate      = file("${path.module}/testdata/cert.pem")
				}`, project.ID, domain),

				ExpectError: regexp.MustCompile(`"certificate" can't be included when "auto_ssl_enabled" is set to true`),
			},
		},
	})
}

func testAcc_GitlabPagesDomain_CheckDestroy(projectID int64, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, err := testutil.TestGitlabClient.PagesDomains.GetPagesDomain(projectID, domain)
		if err == nil {
			return errors.New("Pages Domain still exists")
		}
		if !api.Is404(err) {
			return fmt.Errorf("Error calling API to get the Pages Domain: %w", err)
		}
		return nil
	}
}
