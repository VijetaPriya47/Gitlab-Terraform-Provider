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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAcc_GitlabGroupSecurityPolicyAttachment_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	securityPolicyProject := testutil.CreateProject(t)
	secondSecurityPolicyProject := testutil.CreateProject(t)
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectSecurityPolicyAttachment_CheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_security_policy_attachment" "this" {
					project          = %d
					policy_project = %d
				}`, project.ID, securityPolicyProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_security_policy_attachment.this", "project", strconv.Itoa(project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_security_policy_attachment.this", "policy_project", strconv.Itoa(securityPolicyProject.ID)),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_security_policy_attachment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the security policy
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_security_policy_attachment" "this" {
					project          = %d
					policy_project = %d
				}`, project.ID, secondSecurityPolicyProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_security_policy_attachment.this", "project", strconv.Itoa(project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_security_policy_attachment.this", "policy_project", strconv.Itoa(secondSecurityPolicyProject.ID)),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_security_policy_attachment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Destroy the security policy
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_security_policy_attachment" "this" {
					project          = %d
					policy_project = %d
				}`, project.ID, secondSecurityPolicyProject.ID),
				Destroy: true,
			},
		},
	})
}

func testAcc_GitlabProjectSecurityPolicyAttachment_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_security_policy_attachment" {
			id := rs.Primary.ID

			project, _, err := utils.ParseTwoPartID(id)
			if err != nil {
				return err
			}

			projectIds, err := api.GetProjectGIDFromID(context.Background(), testutil.TestGitlabClient, project)
			if err != nil {
				return err
			}

			query := fmt.Sprintf(`
			query {
				project(fullPath:"%s") {
					id,
					securityPolicyProject {id}
				}
			}
				`, projectIds.ProjectFullPath)

			var response GetSecurityPolicyProjectResponse
			_, err = api.SendGraphQLRequest(context.Background(), testutil.TestGitlabClient, api.GraphQLQuery{Query: query}, &response)
			if err != nil {
				return err
			}

			if response.Data.Project.SecurityPolicyProject != nil && response.Data.Project.SecurityPolicyProject.ID != "" {
				jsonString, _ := json.Marshal(response)
				tflog.Debug(context.Background(), "Security Policy Project was still present in check destroy step.", map[string]interface{}{
					"response": string(jsonString),
				})

				return fmt.Errorf("security policy project still exists")
			}

			return nil
		}
	}
	return nil
}
