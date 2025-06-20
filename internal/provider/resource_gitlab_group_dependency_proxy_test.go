//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupDependencyProxy_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupDependencyProxyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_dependency_proxy" "foo" {
					group     = "%d"
					
					enabled   = true
					identity  = "someidentity"
					secret    = "somesecret"
				}
			`, group.ID),
			},
			{
				ResourceName:      "gitlab_group_dependency_proxy.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"secret", // doesn't come back in the response
					"group",  // can be path or ID, can't be imported as a result.
				},
			},
			// Update to different identity
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_dependency_proxy" "foo" {
					group     = "%d"
					
					enabled   = true
					identity  = "newidentity"
					secret    = "somesecret"
				}
			`, group.ID),
			},
			{
				ResourceName:      "gitlab_group_dependency_proxy.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"secret", // doesn't come back in the response
					"group",  // can be path or ID, can't be imported as a result.
				},
			},
		},
	})
}

func TestAccGitlabGroupDependencyProxy_validation(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupDependencyProxyDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "gitlab_group_dependency_proxy" "foo" {
					group     = "1234"
					
					enabled   = false
					identity  = "someidentity"
				}`,
				ExpectError: regexp.MustCompile("Identity cannot be set when proxy is disabled"),
			},
			{
				Config: `
				resource "gitlab_group_dependency_proxy" "foo" {
					group     = "1234"
					
					enabled   = false
					secret    = "somesecret"
				}`,
				ExpectError: regexp.MustCompile("Secret cannot be set when proxy is disabled"),
			},
			{
				Config: `
				resource "gitlab_group_dependency_proxy" "foo" {
					group     = "1234"
					
					enabled   = true
					secret    = "somesecret"
				}`,
				ExpectError: regexp.MustCompile("Identity must be set when proxy is enabled"),
			},
			{
				Config: `
				resource "gitlab_group_dependency_proxy" "foo" {
					group     = "1234"
					
					enabled   = true
					identity  = "someidentity"
				}`,
				ExpectError: regexp.MustCompile("Secret must be set when proxy is enabled"),
			},
		},
	})
}

func testAccCheckGitlabGroupDependencyProxyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_dependency_proxy" {
			continue
		}

		groupID := rs.Primary.ID
		group, _, err := testutil.TestGitlabClient.Groups.GetGroup(groupID, nil)
		if err != nil {
			return err
		}

		response, err := readGroupDependencyProxySettings(context.Background(), testutil.TestGitlabClient, group)
		if err != nil && !api.Is404(err) {
			return err
		}

		// proxy is still enabled. Should have been removed.
		if response.Data.Group.DependencyProxySettings.Enabled {
			return fmt.Errorf("Group Dependency Proxy still enabled for group %s", group.FullPath)
		}

		return nil
	}
	return nil
}
