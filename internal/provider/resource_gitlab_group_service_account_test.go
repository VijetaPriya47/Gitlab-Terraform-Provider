//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabGroupServiceAccount_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.FormatInt(group.ID, 10)

	name := acctest.RandString(10)
	username := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupServiceAccount_CheckDestroy(),
		Steps: []resource.TestStep{
			// Create a basic service account.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account" "this" {
					group 	 = %s
					name 	 = "%s"
					username = "%s"
					timeouts = {
						delete = "30m"
					}
				}
				`, groupID, name, username),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account.this", "group", groupID),
					resource.TestCheckResourceAttr("gitlab_group_service_account.this", "name", name),
					resource.TestCheckResourceAttr("gitlab_group_service_account.this", "username", username),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account.this", "email"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_service_account.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabGroupServiceAccount_customEmail(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.FormatInt(group.ID, 10)

	name := acctest.RandString(10)
	username := acctest.RandString(10)
	email := fmt.Sprintf("a%s@test.com", username)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupServiceAccount_CheckDestroy(),
		Steps: []resource.TestStep{
			// Create a basic service account.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account" "this" {
					group 	 = %s
					name 	 = "%s"
					username = "%s"
					email    = "%s"
					timeouts = {
						delete = "30m"
					}
				}
				`, groupID, name, username, email),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account.this", "group", groupID),
					resource.TestCheckResourceAttr("gitlab_group_service_account.this", "name", name),
					resource.TestCheckResourceAttr("gitlab_group_service_account.this", "username", username),
					resource.TestCheckResourceAttr("gitlab_group_service_account.this", "email", email),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_service_account.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabGroupServiceAccount_EnsureRecreate(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.FormatInt(group.ID, 10)
	name := acctest.RandString(10)
	username := acctest.RandString(10)
	name2 := acctest.RandString(10)
	username2 := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupServiceAccount_CheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account" "this" {
					group 	 = %s
					name     = "%s"
					username = "%s"
					timeouts = {
						delete = "30m"
					}
				}
				`, groupID, name, username),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_service_account.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account" "this" {
					group 	 = %s
					name     = "%s"
					username = "%s"
					timeouts = {
						delete = "30m"
					}
				}
				`, groupID, name2, username2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account.this", "name", name2),
				),
			},
		},
	})
}

func testAcc_GitlabGroupServiceAccount_CheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type == "gitlab_group_service_account" {
				groupID := rs.Primary.Attributes["group"]
				serviceAccount, err := findGitlabServiceAccount(testutil.TestGitlabClient, groupID, rs.Primary.ID)
				if err == nil {
					return fmt.Errorf("Found GitLab service account that should have been deleted: %s", gitlab.Stringify(serviceAccount))
				}
			}
		}
		return nil
	}
}
