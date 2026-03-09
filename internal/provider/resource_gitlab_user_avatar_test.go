//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabUserAvatar_basic(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	token := testutil.CreatePersonalAccessTokenWithScopes(t, user, []string{"api"})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic user avatar
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, token.UserID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(token.UserID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
			// Verify no change in the plan when avatar doesn't change
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar.png"
				}
				`, token.UserID, token.Token),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update the user avatar.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar-update.png"
				}
				`, token.UserID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(token.UserID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar-update.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
		},
	})
}

func TestAccGitlabUserAvatar_WithHash(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	token := testutil.CreatePersonalAccessTokenWithScopes(t, user, []string{"api"})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic user avatar
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar.png"
					avatar_hash = filesha256("${path.module}/testdata/avatar.png")
				}
				`, token.UserID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(token.UserID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar.png")),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_hash"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
			// Verify no change in the plan when avatar doesn't change
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar.png"
					avatar_hash = filesha256("${path.module}/testdata/avatar.png")
				}
				`, token.UserID, token.Token),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Verify a change in the plan when avatar doesn't change, but the hash does
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar.png"
					avatar_hash = filesha256("${path.module}/testdata/avatar-update.png")
				}
				`, token.UserID, token.Token),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Update the user avatar.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar-update.png"
					avatar_hash = filesha256("${path.module}/testdata/avatar-update.png")
				}
				`, token.UserID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(token.UserID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar-update.png")),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_hash"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
		},
	})
}

func TestAccGitlabUserAvatar_BadScopes(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	insufficientToken := testutil.CreatePersonalAccessTokenWithScopes(t, user, []string{
		"read_user",
		"read_api",
		"read_repository",
		"write_repository",
		"read_registry",
		"write_registry",
		"read_virtual_registry",
		"write_virtual_registry",
		"sudo",
		"admin_mode",
		"create_runner",
		"manage_runner",
		"ai_features",
		"k8s_proxy",
		"self_rotate",
		"read_service_ping",
	})

	unauthorizedToken := testutil.CreatePersonalAccessTokenWithScopes(t, user, []string{
		"read_user",
		"read_repository",
		"write_repository",
		"read_registry",
		"write_registry",
		"read_virtual_registry",
		"write_virtual_registry",
		"sudo",
		"admin_mode",
		"create_runner",
		"manage_runner",
		"ai_features",
		"k8s_proxy",
		"self_rotate",
		"read_service_ping",
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic user avatar, but expect an error due to insufficient scope
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, insufficientToken.UserID, insufficientToken.Token),
				ExpectError: regexp.MustCompile("insufficient_scope"),
			},
			// Create a basic user avatar, but expect an error due to being unauthorized
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, unauthorizedToken.UserID, unauthorizedToken.Token),
				ExpectError: regexp.MustCompile("Unauthorized"),
			},
		},
	})
}

// This test validates the user check in the ModifyPlan function when the token
// is not provided and the user is not an administrator.
func TestAccGitlabUserAvatar_OmittedTokenNotAdminUser(t *testing.T) {
	testutil.SkipIfSaaS(t)

	users := testutil.CreateUsers(t, 2)
	token := testutil.CreatePersonalAccessTokenWithScopes(t, users[0], []string{"api"})

	// Compile the expected error regex for access denied
	accessDeniedRegex, err := regexp.Compile("Access Denied")
	if err != nil {
		t.Errorf("Unable to format expected error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Set avatar for user when using non-administrator token and no token for the user is provided
			{
				// lintignore:AT004  // we need the provider configuration here to test with a different user token
				Config: fmt.Sprintf(`
					provider "gitlab" {
						token = "%s"
					}

					resource "gitlab_user_avatar" "foo" {
						user_id = %d
						avatar  = "${path.module}/testdata/avatar.png"
					}
				`, token.Token, users[1].ID),
				ExpectError: accessDeniedRegex,
			},
		},
	})
}

func TestAccGitlabUserAvatar_SetUserAvatarUsingAdminToken(t *testing.T) {
	testutil.SkipIfSaaS(t)

	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a user avatar as admin
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, user.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("gitlab_user_avatar.foo", "token"),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(user.ID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
			// Verify no change in the plan when avatar doesn't change
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id	= %d
					avatar	= "${path.module}/testdata/avatar.png"
				}
				`, user.ID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update the user avatar.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id	= %d
					avatar	= "${path.module}/testdata/avatar-update.png"
				}
				`, user.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("gitlab_user_avatar.foo", "token"),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(user.ID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar-update.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
		},
	})
}

func TestAccGitlabUserAvatar_ProjectAccessToken(t *testing.T) {
	project := testutil.CreateProject(t)
	token := testutil.CreateProjectAccessToken(t, project.ID, "acc-tests", []string{"api"}, gitlab.DeveloperPermissions, nil)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic user avatar
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, token.UserID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(token.UserID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
			// Verify no change in the plan when avatar doesn't change
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar.png"
				}
				`, token.UserID, token.Token),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update the user avatar.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar-update.png"
				}
				`, token.UserID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(token.UserID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar-update.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
		},
	})
}

func TestAccGitlabUserAvatar_ProjectAccessTokenBadScopes(t *testing.T) {
	project := testutil.CreateProject(t)

	insufficientToken := testutil.CreateProjectAccessToken(t, project.ID, "insufficient", []string{
		"read_api",
		"read_registry",
		"write_registry",
		"read_repository",
		"write_repository",
		"create_runner",
		"manage_runner",
		"ai_features",
		"k8s_proxy",
		"read_observability",
		"write_observability",
		"self_rotate",
	}, gitlab.DeveloperPermissions, nil)

	unauthorizedToken := testutil.CreateProjectAccessToken(t, project.ID, "unauthorized", []string{
		"read_registry",
		"write_registry",
		"read_repository",
		"write_repository",
		"create_runner",
		"manage_runner",
		"ai_features",
		"k8s_proxy",
		"read_observability",
		"write_observability",
		"self_rotate",
	}, gitlab.DeveloperPermissions, nil)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic user avatar, but expect an error due to insufficient scope
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, insufficientToken.UserID, insufficientToken.Token),
				ExpectError: regexp.MustCompile("insufficient_scope"),
			},
			// Create a basic user avatar, but expect an error due to being unauthorized
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, unauthorizedToken.UserID, unauthorizedToken.Token),
				ExpectError: regexp.MustCompile("Unauthorized"),
			},
		},
	})
}

func TestAccGitlabUserAvatar_GroupAccessToken(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	token := testutil.CreateGroupAccessToken(t, group.ID, "acc-tests", []string{"api"}, gitlab.DeveloperPermissions)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic user avatar
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, token.UserID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(token.UserID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
			// Verify no change in the plan when avatar doesn't change
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar.png"
				}
				`, token.UserID, token.Token),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update the user avatar.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar-update.png"
				}
				`, token.UserID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(token.UserID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar-update.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
		},
	})
}

func TestAccGitlabUserAvatar_GroupAccessTokenBadScopes(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	insufficientToken := testutil.CreateGroupAccessToken(t, group.ID, "insufficient", []string{
		"read_api",
		"read_registry",
		"write_registry",
		"read_repository",
		"write_repository",
		"create_runner",
		"manage_runner",
		"ai_features",
		"k8s_proxy",
		"read_observability",
		"write_observability",
		"self_rotate",
	}, gitlab.DeveloperPermissions)

	unauthorizedToken := testutil.CreateGroupAccessToken(t, group.ID, "unauthorized", []string{
		"read_registry",
		"write_registry",
		"read_repository",
		"write_repository",
		"create_runner",
		"manage_runner",
		"ai_features",
		"k8s_proxy",
		"read_observability",
		"write_observability",
		"self_rotate",
	}, gitlab.DeveloperPermissions)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic user avatar, but expect an error due to insufficient scope
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, insufficientToken.UserID, insufficientToken.Token),
				ExpectError: regexp.MustCompile("insufficient_scope"),
			},
			// Create a basic user avatar, but expect an error due to being unauthorized
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, unauthorizedToken.UserID, unauthorizedToken.Token),
				ExpectError: regexp.MustCompile("Unauthorized"),
			},
		},
	})
}

func TestAccGitlabUserAvatar_GroupServiceAccount(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, strconv.FormatInt(group.ID, 10))[0]
	token := testutil.CreateGroupServiceAccountAccessToken(t, group.ID, serviceAccount.ID, fmt.Sprintf("acctest-%d", acctest.RandInt()), []string{"api"})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic service account avatar
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, serviceAccount.ID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(serviceAccount.ID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
			// Verify no change in the plan when avatar doesn't change
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar.png"
				}
				`, token.UserID, token.Token),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update the avatar.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id     = %d
					token       = "%s"
					avatar      = "${path.module}/testdata/avatar-update.png"
				}
				`, serviceAccount.ID, token.Token),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "token", token.Token),
					resource.TestCheckResourceAttr("gitlab_user_avatar.foo", "user_id", strconv.FormatInt(serviceAccount.ID, 10)),
					resource.TestCheckResourceAttrSet("gitlab_user_avatar.foo", "avatar_url"),
					resource.TestMatchResourceAttr("gitlab_user_avatar.foo", "avatar_url", regexp.MustCompile(".+/avatar-update.png")),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_user_avatar.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "avatar"},
			},
		},
	})
}

func TestAccGitlabUserAvatar_GroupServiceAccountBadScopes(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, strconv.FormatInt(group.ID, 10))[0]

	insufficientToken := testutil.CreateGroupServiceAccountAccessToken(t, group.ID, serviceAccount.ID, "insufficient", []string{
		"read_api",
		"read_registry",
		"write_registry",
		"read_repository",
		"write_repository",
		"create_runner",
		"manage_runner",
		"ai_features",
		"k8s_proxy",
		"read_observability",
		"write_observability",
		"self_rotate",
	})

	unauthorizedToken := testutil.CreateGroupServiceAccountAccessToken(t, group.ID, serviceAccount.ID, "unauthorized", []string{
		"read_registry",
		"write_registry",
		"read_repository",
		"write_repository",
		"create_runner",
		"manage_runner",
		"ai_features",
		"k8s_proxy",
		"read_observability",
		"write_observability",
		"self_rotate",
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a basic user avatar, but expect an error due to insufficient scope
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, insufficientToken.UserID, insufficientToken.Token),
				ExpectError: regexp.MustCompile("insufficient_scope"),
			},
			// Create a basic user avatar, but expect an error due to being unauthorized
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_avatar" "foo" {
					user_id = %d
					token   = "%s"
					avatar  = "${path.module}/testdata/avatar.png"
				}
				`, unauthorizedToken.UserID, unauthorizedToken.Token),
				ExpectError: regexp.MustCompile("Unauthorized"),
			},
		},
	})
}
