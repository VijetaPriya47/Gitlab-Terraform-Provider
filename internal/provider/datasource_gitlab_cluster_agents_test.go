//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabClusterAgents_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testClusterAgents := testutil.CreateClusterAgents(t, testProject.ID, 25)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_cluster_agents" "this" {
						project = "%d"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_cluster_agents.this", "cluster_agents.#", fmt.Sprintf("%d", len(testClusterAgents))),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.0.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.0.created_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.0.created_by_user_id"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.1.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.1.created_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.1.created_by_user_id"),

					// Check that the agent_id is set to a non-0 value for the first agent
					resource.TestCheckResourceAttrWith("data.gitlab_cluster_agents.this", "cluster_agents.0.agent_id", func(value string) error {
						if value == "" || value == "0" {
							return fmt.Errorf("agent_id should be a non-empty and non-zero value. Actual value: %s", value)
						}
						return nil
					}),
				),
			},
		},
	})
}
