//go:build acceptance

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabClusterAgent_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testAgent := testutil.CreateClusterAgents(t, testProject.ID, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_cluster_agent" "this" {
						project           = "%d"
						agent_id          = %d
					}
					`, testProject.ID, testAgent.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_cluster_agent.this", "name", testAgent.Name),
					resource.TestCheckResourceAttr("data.gitlab_cluster_agent.this", "created_at", testAgent.CreatedAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_cluster_agent.this", "created_by_user_id", fmt.Sprintf("%d", testAgent.CreatedByUserID)),
				),
			},
		},
	})
}
