//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabInstanceServiceAccount_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	name := acctest.RandString(10)
	username := acctest.RandString(10)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
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

func TestAcc_GitlabInstanceServiceAccount_EnsureRecreate(t *testing.T) {
	testutil.SkipIfCE(t)

	name := acctest.RandString(10)
	username := acctest.RandString(10)
	name2 := acctest.RandString(10)
	username2 := acctest.RandString(10)
	resource.ParallelTest(t, resource.TestCase{
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

func testAcc_GitlabInstanceServiceAccount_CheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type == "gitlab_instance_service_account" {
				serviceAccountID, err := strconv.Atoi(rs.Primary.ID)

				if err != nil {
					return fmt.Errorf("Could not convert id to int")
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
