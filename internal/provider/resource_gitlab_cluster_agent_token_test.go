//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabClusterAgentToken_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testAgent := testutil.CreateClusterAgents(t, testProject.ID, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabClusterAgentTokenDestroy,
		Steps: []resource.TestStep{
			// Verify creation with minimal required attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_cluster_agent_token" "this" {
						project  = "%d"
						agent_id = "%d"
						name     = "agent-1-token"
					}
				`, testProject.ID, testAgent.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_cluster_agent_token.this", "token"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_cluster_agent_token.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Verify update with all attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_cluster_agent_token" "this" {
						project     = "%d"
						agent_id    = "%d"
						name        = "agent-1-token"
						description = "agent-1-description"
					}
				`, testProject.ID, testAgent.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_cluster_agent_token.this", "token"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_cluster_agent_token.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabClusterAgentToken_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testAgent := testutil.CreateClusterAgents(t, testProject.ID, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabClusterAgentTokenDestroy,
		Steps: []resource.TestStep{
			// Verify creation with minimal required attributes
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.2",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_cluster_agent_token" "this" {
						project  = "%d"
						agent_id = "%d"
						name     = "agent-1-token"
					}
				`, testProject.ID, testAgent.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_cluster_agent_token.this", "id"),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_cluster_agent_token" "this" {
						project  = "%d"
						agent_id = "%d"
						name     = "agent-1-token"
					}
				`, testProject.ID, testAgent.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_cluster_agent_token.this", "id"),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_cluster_agent_token.this",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore:  []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabClusterAgentTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_cluster_agent_token" {
			continue
		}

		project, agentID, tokenID, err := resourceGitlabClusterAgentTokenParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		subject, _, err := testutil.TestGitlabClient.ClusterAgents.GetAgentToken(project, agentID, tokenID)
		if err == nil && subject != nil && subject.Status != "revoked" {
			return fmt.Errorf("gitlab_cluster_agent_token resource '%s' not yet revoked", rs.Primary.ID)
		}

		if err != nil && !api.Is404(err) {
			return err
		}

		return nil
	}
	return nil
}
