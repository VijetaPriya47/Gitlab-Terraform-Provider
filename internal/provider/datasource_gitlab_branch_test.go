//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabBranch_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	branch, _, err := testutil.TestGitlabClient.Branches.GetBranch(project.ID, "main")
	if err != nil {
		t.Fatalf("could not get branch: %v", err)
	}

	// Sometimes the branch hasn't been protected yet, so wait a bit and get it again
	if !branch.Protected {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		done := ctx.Done()
		for {
			select {
			case <-done:
				t.Fatalf("timed out waiting for branch to be protected")
				return
			case <-ticker.C:
				branch, _, err = testutil.TestGitlabClient.Branches.GetBranch(project.ID, "main")
				if err != nil {
					t.Fatalf("could not get branch: %v", err)
				}
				if branch.Protected {
					return
				}
			}
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_branch" "foo" {
						name = "main"
						project = "%s"
					}
				`, project.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "name", branch.Name),
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "web_url", branch.WebURL),
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "default", fmt.Sprintf("%t", branch.Default)),
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "can_push", fmt.Sprintf("%t", branch.CanPush)),
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "merged", fmt.Sprintf("%t", branch.Merged)),
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "protected", fmt.Sprintf("%t", branch.Protected)),
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "developer_can_merge", fmt.Sprintf("%t", branch.DevelopersCanMerge)),
					resource.TestCheckResourceAttr("data.gitlab_branch.foo", "developer_can_push", fmt.Sprintf("%t", branch.DevelopersCanPush)),
				),
			},
		},
	})
}
