//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
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

// An explicit tests for the block-style import instead of CLI import (which is tested
// in other tests with the `ImportState` commands). The block import causes replacement
// issues if run in the same apply as other actions, since changing "Group" post-plan
// forces a replace.
// see https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/merge_requests/2607
// nolint - "import" is fine in the test name.
func TestAccGitlabGroupDependencyProxy_import(t *testing.T) {

	group := testutil.CreateGroups(t, 1)[0]
	graphQLcall := fmt.Sprintf(`
mutation {
  updateDependencyProxySettings(input: {
    groupPath: "%s",
    enabled: false,
	identity: "someidentity",
	secret: "somesecret"
  }){
   errors
   dependencyProxySetting{
      enabled,
      identity
   }
  }
}`, group.FullPath)

	// Make the GraphQL call
	var response *updateGroupDependencyProxyGraphQLResponse
	_, err := testutil.TestGitlabClient.GraphQL.Do(gitlab.GraphQLQuery{Query: graphQLcall}, &response)
	if err != nil {
		t.Fatalf("Failed to create initial Dependency Proxy settings via GraphQL: %v", err)
	}

	// Run the test with TF version 1.6.0 to ensure the `import` block is
	// supported
	framework.RunTestWithVersion(t, "1.6.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_6_0),
		},
		CheckDestroy: testAccCheckGitlabGroupDependencyProxyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				import {
					to = gitlab_group_dependency_proxy.this
				    id = "%[1]d"
				}

				resource "gitlab_group_dependency_proxy" "this" {
					group    = "%[1]d"

					enabled  = true
					identity = "someidentity"
					secret   = "somesecret"
				}`, group.ID),
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

		response, err := readGroupDependencyProxySettings(testutil.TestGitlabClient, group)
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
