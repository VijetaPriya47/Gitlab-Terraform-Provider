//go:build acceptance
// +build acceptance

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabCurrentUser_basic(t *testing.T) {
	// Get the user so we can get the verified email that's randomly generated for them
	user, _, err := testutil.TestGitlabClient.Users.GetUser(1, gitlab.GetUsersOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// The root user has no public email by default, set the public email so it shows up properly.
	_, _, err = testutil.TestGitlabClient.Users.ModifyUser(1, &gitlab.ModifyUserOptions{
		// The public email MUST match an email on record for the user, or it gets a bad request.
		PrivateProfile: gitlab.Ptr(false),
		PublicEmail:    gitlab.Ptr(user.Email),
	})
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "gitlab_current_user" "this" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "id", "1"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "global_id", "gid://gitlab/User/1"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "name", "Administrator"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "username", "root"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "bot", "false"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "namespace_id", "1"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "global_namespace_id", "gid://gitlab/Namespaces::UserNamespace/1"),
					// Check only if this attribute is _set_, since other tests may modify it and we can't create a clean user for this test.
					resource.TestCheckResourceAttrSet("data.gitlab_current_user.this", "group_count"),
					// Check that public_email is set; we can't change it because it only allows verified email changes now.
					resource.TestCheckResourceAttrSet("data.gitlab_current_user.this", "public_email"),
				),
			},
		},
	})
}
