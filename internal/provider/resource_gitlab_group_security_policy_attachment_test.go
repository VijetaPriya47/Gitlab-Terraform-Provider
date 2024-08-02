//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAcc_GitlabGroupSecurityPolicyAttachment_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	policyProject := testutil.CreateProject(t)
	secondSecurityPolicyProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupSecurityPolicyAttachment_CheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_security_policy_attachment" "this" {
					group          = %d
					policy_project = %d
				}`, group.ID, policyProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_security_policy_attachment.this", "group", strconv.Itoa(group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_security_policy_attachment.this", "policy_project", strconv.Itoa(policyProject.ID)),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_security_policy_attachment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the security policy
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_security_policy_attachment" "this" {
					group          = %d
					policy_project = %d
				}`, group.ID, secondSecurityPolicyProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_security_policy_attachment.this", "group", strconv.Itoa(group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_security_policy_attachment.this", "policy_project", strconv.Itoa(secondSecurityPolicyProject.ID)),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_security_policy_attachment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Destroy the security policy
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_security_policy_attachment" "this" {
					group          = %d
					policy_project = %d
				}`, group.ID, secondSecurityPolicyProject.ID),
				Destroy: true,
			},
		},
	})
}

func testAcc_GitlabGroupSecurityPolicyAttachment_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_group_security_policy_attachment" {
			id := rs.Primary.ID

			project, _, err := utils.ParseTwoPartID(id)
			if err != nil {
				return err
			}

			groupGid, err := api.GetGroupGIDFromID(context.Background(), testutil.TestGitlabClient, project)
			if err != nil {
				return err
			}

			query := fmt.Sprintf(`
			query {
				group(fullPath:"%s") {
					id,
					securityPolicyProject {id}
				}
			}
				`, groupGid.GroupFullPath)

			var response GetGroupSecurityPolicyProjectResponse
			_, err = api.SendGraphQLRequest(context.Background(), testutil.TestGitlabClient, api.GraphQLQuery{Query: query}, &response)
			if err != nil {
				return err
			}

			if response.Data.Group.SecurityPolicyProject != nil && response.Data.Group.SecurityPolicyProject.ID != "" {
				jsonString, _ := json.Marshal(response)
				tflog.Debug(context.Background(), "Security Policy Project was still present for the group in check destroy step.", map[string]interface{}{
					"response": string(jsonString),
				})

				return fmt.Errorf("security policy project still exists")
			}

			return nil
		}
	}
	return nil
}
