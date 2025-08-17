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

func TestAccGitlabClusterAgent_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabClusterAgentDestroy,
		Steps: []resource.TestStep{
			// Verify creation
			{
				Config: fmt.Sprintf(`
					resource "gitlab_cluster_agent" "this" {
						project = "%d"
						name    = "agent-1"
					}
				`, testProject.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_cluster_agent.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Verify update / re-creation
			{
				Config: fmt.Sprintf(`
					resource "gitlab_cluster_agent" "this" {
						project = "%d"
						name    = "agent-2"
					}
				`, testProject.ID),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_cluster_agent.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabClusterAgent_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabClusterAgentDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.2",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_cluster_agent" "this" {
						project = "%d"
						name    = "agent-1"
					}
				`, testProject.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_cluster_agent.this", "id"),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_cluster_agent" "this" {
						project = "%d"
						name    = "agent-1"
					}
				`, testProject.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_cluster_agent.this", "id"),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_cluster_agent.this",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func testAccCheckGitlabClusterAgentDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_cluster_agent" {
			continue
		}

		project, agentID, err := resourceGitlabClusterAgentParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		subject, _, err := testutil.TestGitlabClient.ClusterAgents.GetAgent(project, agentID)
		if err == nil && subject != nil {
			return fmt.Errorf("gitlab_cluster_agent resource '%s' still exists", rs.Primary.ID)
		}

		if err != nil && !api.Is404(err) {
			return err
		}

		return nil
	}
	return nil
}
