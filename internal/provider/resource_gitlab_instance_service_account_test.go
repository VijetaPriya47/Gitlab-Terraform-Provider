//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabInstanceServiceAccount_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	name := acctest.RandString(10)
	username := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabInstanceServiceAccount_CheckDestroy(),
		Steps: []resource.TestStep{
			// Create a basic service account.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_instance_service_account" "this" {
					name 	 = "%s"
					username = "%s"
				}
				`, name, username),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this", "name", name),
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this", "username", username),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_instance_service_account.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabInstanceServiceAccount_defaults(t *testing.T) {
	testutil.SkipIfCE(t)

	name := acctest.RandString(10)
	username := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabInstanceServiceAccount_CheckDestroy(),
		Steps: []resource.TestStep{
			// Create a basic service account with just defaults.
			{
				Config: `resource "gitlab_instance_service_account" "this" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_instance_service_account.this", "name"),
					resource.TestCheckResourceAttrSet("gitlab_instance_service_account.this", "username"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_instance_service_account.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Create a basic service account with just username.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_instance_service_account" "this2" {
					username = "%s"
				}
				`, username),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_instance_service_account.this2", "name"),
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this2", "username", username),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_instance_service_account.this2",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Create a basic service account with just name.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_instance_service_account" "this3" {
					name 	 = "%s"
				}
				`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this3", "name", name),
					resource.TestCheckResourceAttrSet("gitlab_instance_service_account.this3", "username"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_instance_service_account.this3",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabInstanceServiceAccount_EnsureRecreate(t *testing.T) {
	testutil.SkipIfCE(t)

	name := acctest.RandString(10)
	username := acctest.RandString(10)
	name2 := acctest.RandString(10)
	username2 := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabInstanceServiceAccount_CheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_instance_service_account" "this" {
					name     = "%s"
					username = "%s"
				}
				`, name, username),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_instance_service_account.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_instance_service_account" "this" {
					name     = "%s"
					username = "%s"
				}
				`, name2, username2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this", "name", name2),
				),
			},
		},
	})
}

func TestAcc_GitlabInstanceServiceAccount_WithEmail(t *testing.T) {
	testutil.SkipIfCE(t)

	name := acctest.RandString(10)
	username := acctest.RandString(10)
	email := fmt.Sprintf("%s@example.com", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabInstanceServiceAccount_CheckDestroy(),
		Steps: []resource.TestStep{
			// Create a service account with an email.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_instance_service_account" "this" {
					name     = "%s"
					username = "%s"
					email    = "%s"
				}
				`, name, username, email),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this", "name", name),
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this", "username", username),
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this", "email", email),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_instance_service_account.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabInstanceServiceAccount_CreateWithoutEmail(t *testing.T) {
	testutil.SkipIfCE(t)

	name := acctest.RandString(10)
	username := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabInstanceServiceAccount_CheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_instance_service_account" "this" {
					name     = "%s"
					username = "%s"
				}
				`, name, username),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this", "name", name),
					resource.TestCheckResourceAttr("gitlab_instance_service_account.this", "username", username),
					// Check that email is set and matches the expected pattern for generated emails
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["gitlab_instance_service_account.this"]
						if !ok {
							return fmt.Errorf("Not found: gitlab_instance_service_account.this")
						}
						email := rs.Primary.Attributes["email"]
						if email == "" {
							return fmt.Errorf("Expected generated email, got empty string")
						}
						// Check for noreply pattern
						if !strings.Contains(email, "@noreply.") {
							return fmt.Errorf("Expected generated noreply email, got: %s", email)
						}
						return nil
					},
				),
			},
			{
				ResourceName:      "gitlab_instance_service_account.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAcc_GitlabInstanceServiceAccount_CheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type == "gitlab_instance_service_account" {
				serviceAccountID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
				if err != nil {
					return fmt.Errorf("Could not convert id to int64")
				}

				serviceAccount, _, err := testutil.TestGitlabClient.Users.GetUser(serviceAccountID, gitlab.GetUsersOptions{})
				if err == nil {
					return fmt.Errorf("Found GitLab service account that should have been deleted: %s", gitlab.Stringify(serviceAccount))
				}
			}
		}
		return nil
	}
}
