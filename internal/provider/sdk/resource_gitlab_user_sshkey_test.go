//go:build acceptance

package sdk

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

var (
	testRSAPubKey                  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCi+ErxScCKIVqg2ZRJ6Mx2Yd/RTsh2DGyhUR8z8Iey4rpi1YOBlpTgjxxnSLy26J++Un/iWYDP8wMvEjXElkWz3z4I+Z3mfF3dv039FTCu+O17Mw20Ek4DJxdrKvOgul040sUG/ABVHo6DjqjokjoVJwzUrUmoOtbeMMD8hFN9bWdEVyTj18XQO8nvEe/VkbhCRhAlZC1l60fM07/7Tw83SV5UNAnBtOB+nfa3b24baO+Ijc4+PqYcBuUAF6DvhXW2gZPqf5wjDBJqlDlRTYDdHarMXZAKBpWfWj0gntbtEOM+Fnp6hS1HajaeveNSs6yQwgQEDN2boQnDuvXJ8Y7zW3YQKZp8z0uqWYJSIrYRKVEVYL7gDWL9NvdRV52d/RKPnE/BlL2chiAWBRCT8buQdjVtEPPoYbA1667PXZg6PI9yhCGEIjCj71XzPssA6VL/R7yUafsmNLsirWz9Uyh3HJWCcgNuO9mglP5nfFHIXSHQVhEUEYMfzv1iX5FrenU= terraform@gitlab.com"
	testRSAPubKeyUpdatedComment    = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCi+ErxScCKIVqg2ZRJ6Mx2Yd/RTsh2DGyhUR8z8Iey4rpi1YOBlpTgjxxnSLy26J++Un/iWYDP8wMvEjXElkWz3z4I+Z3mfF3dv039FTCu+O17Mw20Ek4DJxdrKvOgul040sUG/ABVHo6DjqjokjoVJwzUrUmoOtbeMMD8hFN9bWdEVyTj18XQO8nvEe/VkbhCRhAlZC1l60fM07/7Tw83SV5UNAnBtOB+nfa3b24baO+Ijc4+PqYcBuUAF6DvhXW2gZPqf5wjDBJqlDlRTYDdHarMXZAKBpWfWj0gntbtEOM+Fnp6hS1HajaeveNSs6yQwgQEDN2boQnDuvXJ8Y7zW3YQKZp8z0uqWYJSIrYRKVEVYL7gDWL9NvdRV52d/RKPnE/BlL2chiAWBRCT8buQdjVtEPPoYbA1667PXZg6PI9yhCGEIjCj71XzPssA6VL/R7yUafsmNLsirWz9Uyh3HJWCcgNuO9mglP5nfFHIXSHQVhEUEYMfzv1iX5FrenU= terraform2@foo.com"
	updatedRSAPubKey               = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDStVqW58VZ5afXFphIvu2JahndXslJZMkgWsNiYCNdk/NvrEbc4i7yZVoDPFQsbS9I6Ty1RMW7qy3KxJalMsVHcw8arCQFDxs/ka1NHGCUPl68t5ZxUOl900KRQ0lOzGnDQMqG/UUZdPw4CCmigTr6Z9ZBcD1fXAiUwbXR4tWrr5z9KWXC2HgF4WkIJUTIct7ilY1m9W0y79dI/+K8bZrurn3q2QK83pxqqWkLwvUsCxtlhMpwuyflyzyuz8xPZl2GlZgxeIpr68gsPHIzzWizibwFfbRYKCZO4wD0r7JCDOYs9KjcIPpCG6d3HUqijClgdQSBnLwHTdE04ZtdzO8akvy0hMzRCooI5TSc8IAHos53Gp9aaW92sPA8za+WRP6OSH6UsOW4N+iQc4jyl7/fckMSgIZlJouNqqV+P8iqIFJGs70Tj5L8G/m+P2lc3kcE4Vjmj+Fc0xG5+I/PsSOpcc6DfDfZdVDRe8yklYd/qC1jI89OCeqjxu3XcUGHj9s= terraform@gitlab.com"
	updatedRSAPubKeyWithoutComment = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDStVqW58VZ5afXFphIvu2JahndXslJZMkgWsNiYCNdk/NvrEbc4i7yZVoDPFQsbS9I6Ty1RMW7qy3KxJalMsVHcw8arCQFDxs/ka1NHGCUPl68t5ZxUOl900KRQ0lOzGnDQMqG/UUZdPw4CCmigTr6Z9ZBcD1fXAiUwbXR4tWrr5z9KWXC2HgF4WkIJUTIct7ilY1m9W0y79dI/+K8bZrurn3q2QK83pxqqWkLwvUsCxtlhMpwuyflyzyuz8xPZl2GlZgxeIpr68gsPHIzzWizibwFfbRYKCZO4wD0r7JCDOYs9KjcIPpCG6d3HUqijClgdQSBnLwHTdE04ZtdzO8akvy0hMzRCooI5TSc8IAHos53Gp9aaW92sPA8za+WRP6OSH6UsOW4N+iQc4jyl7/fckMSgIZlJouNqqV+P8iqIFJGs70Tj5L8G/m+P2lc3kcE4Vjmj+Fc0xG5+I/PsSOpcc6DfDfZdVDRe8yklYd/qC1jI89OCeqjxu3XcUGHj9s="
	testRSAPubKeyForCurrentUser    = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCzmWrvEwMdHU4znC22Frv6Teg7+ezSCje7AIrWZMQonds4REUw3SBhHrZq9QX6KC2tIKYo+kTlrCv4ww7rl0z9NFwcqRKmdq3Yx0nVO0N/ox3wpnKltWKwGt+QcAESiryQESfsRk85r6AQ6061XMj4/PlUXydesnwe/ehifjGTQ03Ps/rT/MiByjj2LmxWygfdRdHtEXaz8Tavi8VdZFYgKbUNHhaGk7Mbrb39Mb2kdoZgmO26hjeZYvN1kO99yUUX0UyXXHOVts5xDDsRyHn4gwuoBu09TWG16enQxRxs5UI6H3tkc1eIR5hE3KQ5WVdhc7lZBhoETzZUkQbAv6GuyXqy9/W77MlWUNl8KwjchHG7U/maVF7X5+oA66zoHzHHSp6/6fbC4NwwaF6d6HErySd+1mZ0Dh+shBltlhCoZz78OQYk+UV1w2c5M2ae4wef42zlWZh2PbfeX6G4SlsxSTxqbLKQks4JNKrwCeHCCAIX8jNQ91pikr/1G1tiIsp4MgjoD8ivUDG0ak1cE0GwPMjJiyf/sJ+SKm7YKqdcU6wgsKVGNXeXShf41xYTqDrQE0eJNIKc7+t3gUfKdvzDNCS3XvIeqMIkoEIVYztv1O0oy711hzF1UnnW3eTglQuedPwbA/0B5E1Hj9+xmMQ1/ugaSuYo8V8ddNNWOF6aSQ=="
	testKeyWithTrailingNewline     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMG5+BWfNRCNE9chUUooEwS/QeNMN5Z1RBdY1GQ0VqMa\n"
)

func TestAccGitlabUserSSHKey_basic(t *testing.T) {
	var key gitlab.SSHKey
	testUser := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabUserSSHKeyDestroy,
		Steps: []resource.TestStep{
			// Create a user + sshkey
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_sshkey" "foo_key" {
						title   = "foo-key"
						key     = "%s"
						user_id = %d
					}
				`, testRSAPubKey, testUser.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserSSHKeyExists("gitlab_user_sshkey.foo_key", &key),
					testAccCheckGitlabUserSSHKeyAttributes(&key, &testAccGitlabUserSSHKeyExpectedAttributes{
						Title: "foo-key",
						Key:   testRSAPubKey,
					}),
				),
			},
			// Only update key comment (which is a no-op plan)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_sshkey" "foo_key" {
						title   = "foo-key"
						key     = "%s"
						user_id = %d
					}
				`, testRSAPubKeyUpdatedComment, testUser.ID),
				PlanOnly: true,
			},
			// Update the key and title
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_sshkey" "foo_key" {
						title      = "key"
						key        = "%s"
						user_id    = %d
						expires_at = "3016-01-21T00:00:00Z"
					}
				`, updatedRSAPubKey, testUser.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserSSHKeyExists("gitlab_user_sshkey.foo_key", &key),
					testAccCheckGitlabUserSSHKeyAttributes(&key, &testAccGitlabUserSSHKeyExpectedAttributes{
						Title:     "key",
						Key:       updatedRSAPubKey,
						ExpiresAt: "3016-01-21T00:00:00Z",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user_sshkey.foo_key",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Change pub key to one without a comment
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_sshkey" "foo_key" {
						title   = "foo-key"
						key     = "%s"
						user_id = %d
					}
				`, updatedRSAPubKeyWithoutComment, testUser.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserSSHKeyExists("gitlab_user_sshkey.foo_key", &key),
					testAccCheckGitlabUserSSHKeyAttributes(&key, &testAccGitlabUserSSHKeyExpectedAttributes{
						Title: "foo-key",
						Key:   updatedRSAPubKeyWithoutComment,
					}),
				),
			},
			{
				ResourceName:      "gitlab_user_sshkey.foo_key",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabUserSSHKey_currentuser(t *testing.T) {
	var key gitlab.SSHKey

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabUserSSHKeyDestroy,
		Steps: []resource.TestStep{
			// Create a user
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_sshkey" "current_user" {
					  title = "current-user"
					  key = "%s"
					}
				`, testRSAPubKeyForCurrentUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserSSHKeyExists("gitlab_user_sshkey.current_user", &key),
					testAccCheckGitlabUserSSHKeyAttributes(&key, &testAccGitlabUserSSHKeyExpectedAttributes{
						Title: "current-user",
						Key:   testRSAPubKeyForCurrentUser,
					}),
				),
			},
		},
	})
}

func TestAccGitlabUserSSHKey_ignoreTrailingWhitespaces(t *testing.T) {
	testUser := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabUserSSHKeyDestroy,
		Steps: []resource.TestStep{
			// Create a user + sshkey
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_sshkey" "this" {
						user_id = %d
						title   = "test"
						key     = <<EOF
						%s
						EOF
					}
				`, testUser.ID, testKeyWithTrailingNewline),
			},
			// Check for no-op plan
			{
				Config: fmt.Sprintf(`
					resource "gitlab_user_sshkey" "this" {
						user_id = %d
						title   = "test"
						key     = <<EOF
						%s
						EOF
					}
				`, testUser.ID, testKeyWithTrailingNewline),
				PlanOnly: true,
			},
			// Verify Import
			{
				ResourceName:      "gitlab_user_sshkey.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabUserSSHKeyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_user_sshkey" {
			continue
		}

		userID, keyID, err := resourceGitlabUserSSHKeyParseID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to parse user ssh key resource ID: %s", err)
		}

		keys, _, err := testutil.TestGitlabClient.Users.ListSSHKeysForUser(userID, &gitlab.ListSSHKeysForUserOptions{})
		if err != nil {
			return err
		}

		var gotKey *gitlab.SSHKey

		for _, k := range keys {
			if k.ID == keyID {
				gotKey = k
				break
			}
		}
		if gotKey != nil {
			return fmt.Errorf("SSH Key still exists")
		}

		return nil
	}
	return nil
}

func testAccCheckGitlabUserSSHKeyExists(n string, key *gitlab.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		userID, keyID, err := resourceGitlabUserSSHKeyParseID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to parse user ssh key resource ID: %s", err)
		}

		keys, _, err := testutil.TestGitlabClient.Users.ListSSHKeysForUser(userID, &gitlab.ListSSHKeysForUserOptions{})
		if err != nil {
			return err
		}

		var gotKey *gitlab.SSHKey

		for _, k := range keys {
			if k.ID == keyID {
				gotKey = k
				break
			}
		}
		if gotKey == nil {
			return fmt.Errorf("Could not find sshkey %d for user %d", keyID, userID)
		}

		*key = *gotKey
		return nil
	}
}

type testAccGitlabUserSSHKeyExpectedAttributes struct {
	Title     string
	Key       string
	CreatedAt string
	ExpiresAt string
}

func testAccCheckGitlabUserSSHKeyAttributes(key *gitlab.SSHKey, want *testAccGitlabUserSSHKeyExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if key.Title != want.Title {
			return fmt.Errorf("got title %q; want %q", key.Title, want.Title)
		}

		k := strings.Join(strings.Split(key.Key, " ")[:2], " ")
		wk := strings.Join(strings.Split(want.Key, " ")[:2], " ")

		if k != wk {
			return fmt.Errorf("got key %q; want %q", k, wk)
		}

		return nil
	}
}
