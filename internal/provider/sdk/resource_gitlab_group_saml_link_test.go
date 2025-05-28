//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupSamlLink_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupSamlLinkDestroy,
		Steps: []resource.TestStep{
			// Create a group SAML link as a developer
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_saml_link" "this" {
						group   		= "%d"
						access_level 	= "developer"
						saml_group_name = "test_saml_group"

					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_saml_link.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group SAML link to change the access level
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_saml_link" "this" {
						group   		= "%d"
						access_level 	= "maintainer"
						saml_group_name = "test_saml_group"

					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_saml_link.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group SAML link to change the access level to the newer type `planner`
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_saml_link" "this" {
						group   		= "%d"
						access_level 	= "planner"
						saml_group_name = "test_saml_group"

					}
				`, testGroup.ID),
			},
		},
	})
}

func TestAccGitlabGroupSamlLink_customRole(t *testing.T) {
	// Group level custom roles don't work on self managed, so we can't test them without a SaaS project.
	// See https://gitlab.com/gitlab-org/gitlab/-/issues/439284 for more details
	t.Skip()

	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupSamlLinkDestroy,
		Steps: []resource.TestStep{
			// Create a group SAML link as a developer with a custom role
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_saml_link" "this" {
						group   	= "%d"
						access_level 	= "developer"
						saml_group_name = "test_saml_group"
						member_role_id  = 123
					}
				`, testGroup.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_saml_link.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group SAML link to change the custom role
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_saml_link" "this" {
						group   	= "%d"
						access_level 	= "developer"
						saml_group_name = "test_saml_group"
						member_role_id  = 456
					}
				`, testGroup.ID),
			},
		},
	})
}

func testAccCheckGitlabGroupSamlLinkDestroy(s *terraform.State) error {
	for _, resourceState := range s.RootModule().Resources {
		if resourceState.Type != "gitlab_group_saml_link" {
			continue
		}

		group, samlGroupName, err := utils.ParseTwoPartID(resourceState.Primary.ID)
		if err != nil {
			return err
		}

		samlGroupLink, _, err := testutil.TestGitlabClient.Groups.GetGroupSAMLLink(group, samlGroupName)
		if err == nil {
			if samlGroupLink != nil {
				return fmt.Errorf("SAML Group Link still exists")
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
