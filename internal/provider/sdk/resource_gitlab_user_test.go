//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabUser_basic(t *testing.T) {
	var user gitlab.User
	rInt := acctest.RandInt()
	password := acctest.RandString(16)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabUserDestroy,
		Steps: []resource.TestStep{
			// Create a user
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "active",
					}),
					// If there is no value in the config, this should be true
					resource.TestCheckResourceAttr("gitlab_user.foo", "skip_confirmation", "true"),
				),
			},
			{
				ResourceName:      "gitlab_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_confirmation",
				},
			},
			// Create a user with blocked state
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				  state            = "blocked"
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "blocked",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_confirmation",
				},
			},
			// Update the user to change the name, email, projects_limit and more
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "bar %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@tttt.com"
				  is_admin         = true
				  projects_limit   = 10
				  can_create_group = true
				  is_external      = false
				  note             = "note%d"
				}
				  `, rInt, rInt, password, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@tttt.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("bar %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  10,
						Admin:          true,
						CanCreateGroup: true,
						External:       false,
						Note:           fmt.Sprintf("note%d", rInt),
						State:          "active",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_confirmation",
				},
			},
			// Update the user to change the state to blocked
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "bar %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@tttt.com"
				  is_admin         = true
				  projects_limit   = 10
				  can_create_group = true
				  is_external      = false
				  note             = "note%d"
				  state            = "blocked"
				}
				  `, rInt, rInt, password, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@tttt.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("bar %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  10,
						Admin:          true,
						CanCreateGroup: true,
						External:       false,
						Note:           fmt.Sprintf("note%d", rInt),
						State:          "blocked",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_confirmation",
				},
			},
			// Update the user to put the name back
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "active",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_confirmation",
				},
			},
			// Update the user to disable skip confirmation
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name              = "foo %d"
				  username          = "listest%d"
				  password          = "%s"
				  email             = "listest%d@ssss.com"
				  is_admin          = false
				  projects_limit    = 0
				  can_create_group  = false
				  is_external       = false
				  skip_confirmation = false
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "active",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_confirmation",
				},
			},
			// Update the user to initial config
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "active",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_confirmation",
				},
			},
			// Deactivate the user
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				  state            = "deactivated"
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "deactivated",
					}),
				),
			},
			// Re-activate the user
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "active",
					}),
				),
			},
			// Block the user
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				  state            = "blocked"
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "blocked",
					}),
				),
			},
			// Deactivate the user from blocked state
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				  state            = "deactivated"
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "deactivated",
					}),
				),
			},
			// Block the user from deactivate state
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				  state            = "blocked"
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "blocked",
					}),
				),
			},
			// Unblock the user
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  password         = "%s"
				  email            = "listest%d@ssss.com"
				  is_admin         = false
				  projects_limit   = 0
				  can_create_group = false
				  is_external      = false
				}
				  `, rInt, rInt, password, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserExists("gitlab_user.foo", &user),
					testAccCheckGitlabUserAttributes(&user, &testAccGitlabUserExpectedAttributes{
						Email:          fmt.Sprintf("listest%d@ssss.com", rInt),
						Username:       fmt.Sprintf("listest%d", rInt),
						Name:           fmt.Sprintf("foo %d", rInt),
						NamespaceID:    user.NamespaceID,
						ProjectsLimit:  0,
						Admin:          false,
						CanCreateGroup: false,
						External:       false,
						State:          "active",
					}),
				),
			},
		},
	})
}

// Test that the fix for suppressing skip_confirmation works appropriately, and doesn't suppress create
func TestAccGitlabUser_user_skip_confirmation(t *testing.T) {
	var user gitlab.User
	rInt := acctest.RandInt()
	password := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "example_user" {
					name               = "Example User"
					username           = "exampleuser"
					email              = "user%d@example.com"
					is_admin           = true
					projects_limit     = 0
					can_create_group   = false
					is_external        = false
					note               = "Ipsum Lorem."
					password           = "%s"
					skip_confirmation  = false
				  }
				`, rInt, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user.example_user", "skip_confirmation", "false"),
					testAccCheckGitlabUserExists("gitlab_user.example_user", &user),
				),
			},
			{
				ResourceName:      "gitlab_user.example_user",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"skip_confirmation",
				},
			},
			{
				Config: fmt.Sprintf(`
				  resource "gitlab_user" "example_user" {
					name               = "Example User"
					username           = "exampleuser"
					email              = "user%d@example.com"
					is_admin           = true
					projects_limit     = 0
					can_create_group   = false
					is_external        = false
					note               = "Ipsum Lorem."
					password           = "%s"
					skip_confirmation  = true 
				  }
				  resource "gitlab_user" "example_user_new" {
					name               = "Example User"
					username           = "exampleusernew"
					email              = "user-new%d@example.com"
					is_admin           = true
					projects_limit     = 0
					can_create_group   = false
					is_external        = false
					note               = "Ipsum Lorem."
					password           = "%s"
					skip_confirmation  = true
				  }
				`, rInt, password, rInt, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user.example_user_new", "skip_confirmation", "true"),
					// Even though "Skip_confirmation" is set to true above, our diff should be ignored
					resource.TestCheckResourceAttr("gitlab_user.example_user", "skip_confirmation", "false"),
				),
			},
		},
	})
}

func TestAccGitlabUser_password_reset(t *testing.T) {
	var user gitlab.User
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabUserDestroy,
		Steps: []resource.TestStep{
			// Test that either password or reset_password is needed
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  email            = "listest%d@ssss.com"
				}
				  `, rInt, rInt, rInt),
				ExpectError: regexp.MustCompile("At least one of either password or reset_password must be defined"),
			},
			// Create a user without a password
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name             = "foo %d"
				  username         = "listest%d"
				  email            = "listest%d@ssss.com"
				  reset_password   = true
				}
				  `, rInt, rInt, rInt),
				Check: testAccCheckGitlabUserExists("gitlab_user.foo", &user),
			},
			{
				ResourceName:      "gitlab_user.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"reset_password",
					"skip_confirmation",
				},
			},
		},
	})
}

func TestAccGitlabUser_external_provider(t *testing.T) {
	var user gitlab.User
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabUserDestroy,
		Steps: []resource.TestStep{
			// Test error if extern_uid set and external_provider isn't
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name           = "foo %d"
				  username       = "listest%d"
				  email          = "listest%d@ssss.com"
				  reset_password = true
				  extern_uid     = "%d"
				}
				  `, rInt, rInt, rInt, rInt),
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
			// Test error if external_provider set and extern_uid isn't
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name              = "foo %d"
				  username          = "listest%d"
				  email             = "listest%d@ssss.com"
				  reset_password    = true
				  external_provider = "google"
				}
				  `, rInt, rInt, rInt),
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
			// Create a user with both set
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user" "foo" {
				  name              = "foo %d"
				  username          = "listest%d"
				  email             = "listest%d@ssss.com"
				  reset_password    = true
				  extern_uid        = "%d"
				  external_provider = "google"
				}
				  `, rInt, rInt, rInt, rInt),
				Check: testAccCheckGitlabUserExists("gitlab_user.foo", &user),
			},
		},
	})
}

func testAccCheckGitlabUserExists(n string, user *gitlab.User) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		userID := rs.Primary.ID
		if userID == "" {
			return fmt.Errorf("No user ID is set")
		}
		id, _ := strconv.Atoi(userID)

		gotUser, _, err := testutil.TestGitlabClient.Users.GetUser(id, gitlab.GetUsersOptions{})
		if err != nil {
			return err
		}
		*user = *gotUser
		return nil
	}
}

type testAccGitlabUserExpectedAttributes struct {
	Email          string
	Username       string
	Name           string
	NamespaceID    int
	ProjectsLimit  int
	Admin          bool
	CanCreateGroup bool
	External       bool
	Note           string
	State          string
}

func testAccCheckGitlabUserAttributes(user *gitlab.User, want *testAccGitlabUserExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if user.Name != want.Name {
			return fmt.Errorf("got name %q; want %q", user.Name, want.Name)
		}

		if user.Username != want.Username {
			return fmt.Errorf("got username %q; want %q", user.Username, want.Username)
		}

		if user.Email != want.Email {
			return fmt.Errorf("got email %q; want %q", user.Email, want.Email)
		}

		if user.CanCreateGroup != want.CanCreateGroup {
			return fmt.Errorf("got can_create_group %t; want %t", user.CanCreateGroup, want.CanCreateGroup)
		}

		if user.External != want.External {
			return fmt.Errorf("got is_external %t; want %t", user.External, want.External)
		}

		if user.Note != want.Note {
			return fmt.Errorf("got note %q; want %q", user.Note, want.Note)
		}

		if user.IsAdmin != want.Admin {
			return fmt.Errorf("got is_admin %t; want %t", user.IsAdmin, want.Admin)
		}

		if user.ProjectsLimit != want.ProjectsLimit {
			return fmt.Errorf("got projects_limit %d; want %d", user.ProjectsLimit, want.ProjectsLimit)
		}

		if user.State != want.State {
			return fmt.Errorf("got state %q; want %q", user.State, want.State)
		}

		return nil
	}
}

func testAccCheckGitlabUserDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_user" {
			continue
		}

		id, _ := strconv.Atoi(rs.Primary.ID)

		user, _, err := testutil.TestGitlabClient.Users.GetUser(id, gitlab.GetUsersOptions{})
		if err == nil {
			if user != nil && fmt.Sprintf("%d", user.ID) == rs.Primary.ID {
				return fmt.Errorf("User still exists")
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
