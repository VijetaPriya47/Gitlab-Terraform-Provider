//go:build flakey
// +build flakey

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAcc_GitlabProjectSecurityPolicyAttachment_basic(t *testing.T) {
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

// This test validates the ownership check in the ModifyPlan function when the user
// is not a member of the project at all.
func TestAcc_GitlabProjectSecurityPolicyAttachment_NotAMember(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create projects and users for the test
	project := testutil.CreateProject(t)
	securityPolicyProject := testutil.CreateProject(t)

	// Create a user but do NOT add them to the project
	users := testutil.CreateUsers(t, 1)

	// Create a personal access token for the non-member user
	nonMemberUserPAT := testutil.CreatePersonalAccessToken(t, users[0])

	// Compile the expected error regex for access denied
	accessDeniedRegex, err := regexp.Compile("Access Denied")
	if err != nil {
		t.Errorf("Unable to format expected access denied error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectSecurityPolicyAttachment_CheckDestroy,
		Steps: []resource.TestStep{
			// Attempt to create security policy attachment with non-member user - should fail
			{
				// lintignore:AT004  // we need the provider configuration here to test with a different user token
				Config: fmt.Sprintf(`
					provider "gitlab" {
						token = "%s"
					}

					resource "gitlab_project_security_policy_attachment" "this" {
						project        = %d
						policy_project = %d
					}
				`, nonMemberUserPAT.Token, project.ID, securityPolicyProject.ID),
				ExpectError: accessDeniedRegex,
			},
		},
	})
}

// This test validates the ownership check in the ModifyPlan function when the user
// is a member but has insufficient permissions (Maintainer instead of Owner).
func TestAcc_GitlabProjectSecurityPolicyAttachment_InsufficientPermissions(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create projects and users for the test
	project := testutil.CreateProject(t)
	securityPolicyProject := testutil.CreateProject(t)

	// Create a user and add them as Maintainer to the project (not Owner)
	users := testutil.CreateUsers(t, 1)
	testutil.AddProjectMembersWithAccessLevel(t, project.ID, users, gitlab.MaintainerPermissions)

	// Create a personal access token for the maintainer user
	maintainerUserPAT := testutil.CreatePersonalAccessToken(t, users[0])

	// Compile the expected error regex for insufficient permissions
	insufficientPermissionsRegex, err := regexp.Compile("Insufficient Permissions")
	if err != nil {
		t.Errorf("Unable to format expected permission error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectSecurityPolicyAttachment_CheckDestroy,
		Steps: []resource.TestStep{
			// Attempt to create security policy attachment with maintainer user - should fail
			{
				// lintignore:AT004  // we need the provider configuration here to test with a different user token
				Config: fmt.Sprintf(`
					provider "gitlab" {
						token = "%s"
					}

					resource "gitlab_project_security_policy_attachment" "this" {
						project        = %d
						policy_project = %d
					}
				`, maintainerUserPAT.Token, project.ID, securityPolicyProject.ID),
				ExpectError: insufficientPermissionsRegex,
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
			_, err = testutil.TestGitlabClient.GraphQL.Do(gitlab.GraphQLQuery{Query: query}, &response)
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
