//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceGitlabUser_basic(t *testing.T) {
	rString := fmt.Sprintf("%s", acctest.RandString(5)) // nolint // TODO: Resolve this golangci-lint issue: S1025: the argument is already a string, there's no need to use fmt.Sprintf (gosimple)
	password := acctest.RandString(16)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Get user using its email
			{
				Config: testAccDataGitlabUserConfigEmail(rString, password),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabUser("gitlab_user.foo", "data.gitlab_user.foo"),
				),
			},
			// Get user using its ID
			{
				Config: testAccDataGitlabUserConfigUserID(rString, password),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabUser("gitlab_user.foo2", "data.gitlab_user.foo2"),
				),
			},
			// Get user using its username
			{
				Config: testAccDataGitlabUserConfigUsername(rString, password),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabUser("gitlab_user.foo", "data.gitlab_user.foo"),
				),
			},
		},
	})
}

func testAccDataSourceGitlabUser(src, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		user := s.RootModule().Resources[src]
		userResource := user.Primary.Attributes

		search := s.RootModule().Resources[n]
		searchResource := search.Primary.Attributes

		testAttributes := []string{
			"username",
			"email",
			"name",
			"is_admin",
			"can_create_group",
			"projects_limit",
		}

		for _, attribute := range testAttributes {
			if searchResource[attribute] != userResource[attribute] {
				return fmt.Errorf("Expected user's parameter `%s` to be: %s, but got: `%s`", attribute, userResource[attribute], searchResource[attribute])
			}
		}

		return nil
	}
}

func testAccDataGitlabUserConfigEmail(rString string, password string) string {
	return fmt.Sprintf(`
resource "gitlab_user" "foo" {
  name     = "foo%s"
  username = "listest%s"
  password = "%s"
  email    = "listest%s@ssss.com"
  is_admin = false
}

resource "gitlab_user" "foo2" {
  name     = "foo2%s"
  username = "listest2%s"
  password = "%s"
  email    = "listest2%s@ssss.com"
}

data "gitlab_user" "foo" {
  email = "${gitlab_user.foo.email}"
}
`, rString, rString, password, rString, rString, rString, password, rString)
}

func testAccDataGitlabUserSameSuffixEmail(rString string, password string) string {
	return fmt.Sprintf(`
resource "gitlab_user" "foo" {
  name     = "foo%s"
  username = "listest%s"
  password = "%s"
  email    = "listest@ssss.com"
  is_admin = false
}

resource "gitlab_user" "foo2" {
  name     = "foo2%s"
  username = "listest2%s"
  password = "%s"
  email    = "%slistest@ssss.com"
}

data "gitlab_user" "foo" {
  email = "${gitlab_user.foo.email}"
}
`, rString, rString, password, rString, rString, password, rString)
}

func testAccDataGitlabUserConfigUserID(rString string, password string) string {
	return fmt.Sprintf(`
resource "gitlab_user" "foo" {
  name     = "foo%s"
  username = "listest%s"
  password = "%s"
  email    = "listest%s@ssss.com"
  is_admin = false
}

resource "gitlab_user" "foo2" {
  name     = "foo2%s"
  username = "listest2%s"
  password = "%s"
  email    = "listest2%s@ssss.com"
}

data "gitlab_user" "foo2" {
  user_id = "${gitlab_user.foo2.id}"
}
`, rString, rString, password, rString, rString, rString, password, rString)
}

func testAccDataGitlabUserConfigUsername(rString string, password string) string {
	return fmt.Sprintf(`
resource "gitlab_user" "foo" {
  name     = "foo%s"
  username = "listest%s"
  password = "%s"
  email    = "listest%s@ssss.com"
  is_admin = false
}

resource "gitlab_user" "foo2" {
  name     = "foo2%s"
  username = "listest2%s"
  password = "%s"
  email    = "listest2%s@ssss.com"
}

data "gitlab_user" "foo" {
  username = "${gitlab_user.foo.username}"
}
`, rString, rString, password, rString, rString, rString, password, rString)
}

func TestAccDataSourceGitlabUser_ExactEmail(t *testing.T) {
	rString := acctest.RandString(5)
	password := acctest.RandString(16)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Get user using its email
			{
				Config: testAccDataGitlabUserSameSuffixEmail(rString, password),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabUser("gitlab_user.foo", "data.gitlab_user.foo"),
				),
			},
		},
	})
}
