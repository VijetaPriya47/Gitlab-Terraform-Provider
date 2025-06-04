//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabBranch_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	branch, _, err := testutil.TestGitlabClient.Branches.GetBranch(project.ID, "main")
	if err != nil {
		t.Fatalf("could not get branch: %v", err)
	}

	// Sometimes the branch hasn't been protected yet, so wait a bit and get it again
	for range 5 {
		if branch.Protected {
			break
		}
		//nolint // R018 this is part of testing code, not the provider itself.
		time.Sleep(10 * time.Second)
		branch, _, err = testutil.TestGitlabClient.Branches.GetBranch(project.ID, "main")
		if err != nil {
			t.Fatalf("could not get branch: %v", err)
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
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
