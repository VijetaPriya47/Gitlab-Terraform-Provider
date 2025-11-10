//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabUserIdentity_basic(t *testing.T) {
	rInt := acctest.RandInt()
	rInt2 := acctest.RandInt()
	rInt3 := acctest.RandInt()
	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabUserIdentityDestroy,
		Steps: []resource.TestStep{
			// Create two user identities
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_identity" "google" {
						user_id = %d
						external_provider = "google"
						external_uid = "%d"
					}
					
					resource "gitlab_user_identity" "salesforce" {
						user_id = %d
						external_provider = "salesforce"
						external_uid = "%d"
					}
				`, user.ID, rInt, user.ID, rInt3),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_identity.google", "id", fmt.Sprintf("%d:google", user.ID)),
					resource.TestCheckResourceAttr("gitlab_user_identity.google", "user_id", fmt.Sprintf("%d", user.ID)),
					resource.TestCheckResourceAttr("gitlab_user_identity.google", "external_provider", "google"),
					resource.TestCheckResourceAttr("gitlab_user_identity.google", "external_uid", fmt.Sprintf("%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_user_identity.salesforce", "id", fmt.Sprintf("%d:salesforce", user.ID)),
					resource.TestCheckResourceAttr("gitlab_user_identity.salesforce", "user_id", fmt.Sprintf("%d", user.ID)),
					resource.TestCheckResourceAttr("gitlab_user_identity.salesforce", "external_provider", "salesforce"),
					resource.TestCheckResourceAttr("gitlab_user_identity.salesforce", "external_uid", fmt.Sprintf("%d", rInt3)),
				),
			},
			// Verify Imports
			{
				ResourceName:      "gitlab_user_identity.google",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "gitlab_user_identity.salesforce",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the identities
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_identity" "google" {
						user_id = %d
						external_provider = "google"
						external_uid = "%d"
					}

					resource "gitlab_user_identity" "salesforce" {
						user_id = %d
						external_provider = "salesforce"
						external_uid = "%d"
					}
				`, user.ID, rInt2, user.ID, rInt3),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_identity.google", "id", fmt.Sprintf("%d:google", user.ID)),
					resource.TestCheckResourceAttr("gitlab_user_identity.google", "user_id", fmt.Sprintf("%d", user.ID)),
					resource.TestCheckResourceAttr("gitlab_user_identity.google", "external_provider", "google"),
					resource.TestCheckResourceAttr("gitlab_user_identity.google", "external_uid", fmt.Sprintf("%d", rInt2)),
					resource.TestCheckResourceAttr("gitlab_user_identity.salesforce", "id", fmt.Sprintf("%d:salesforce", user.ID)),
					resource.TestCheckResourceAttr("gitlab_user_identity.salesforce", "user_id", fmt.Sprintf("%d", user.ID)),
					resource.TestCheckResourceAttr("gitlab_user_identity.salesforce", "external_provider", "salesforce"),
					resource.TestCheckResourceAttr("gitlab_user_identity.salesforce", "external_uid", fmt.Sprintf("%d", rInt3)),
				),
			},
			// Verify Imports
			{
				ResourceName:      "gitlab_user_identity.google",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "gitlab_user_identity.salesforce",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabUserIdentityDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_user_identity" {
			continue
		}

		userID, provider, err := resourceGitlabUserIdentityParseID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("[ERROR] cannot get User ID and Provider from input: %v", rs.Primary.ID)
		}

		user, _, err := testutil.TestGitlabClient.Users.GetUser(userID, gitlab.GetUsersOptions{})
		if err != nil {
			return err
		}

		for _, identity := range user.Identities {
			if identity.Provider == provider {
				return fmt.Errorf("GitLab User Identity %s still exists", provider)
			}
		}
		return nil
	}

	return nil
}
