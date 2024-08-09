//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectIDs_basic(t *testing.T) {

	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`				
				data "gitlab_project_ids" "foo" {
				  project = "%s"
				}
				`, project.PathWithNamespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_ids.foo", "project_id", strconv.Itoa(project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_ids.foo", "project_full_path", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_project_ids.foo", "project_graphql_id", fmt.Sprintf("gid://gitlab/Project/%d", project.ID)),
				),
			},
		},
	})
}
