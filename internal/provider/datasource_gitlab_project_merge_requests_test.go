//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitLabProjectMergeRequests_DataSource_Conflicts(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
				    data "gitlab_project_merge_requests" "this" {
				      project = "%s"
				      author_id = 4
					  author_username = "test"
				    }
				    `, project.PathWithNamespace,
				),
				ExpectError: regexp.MustCompile(
					"Invalid Attribute Combination",
				),
			},
		},
	})
}

func TestAcc_GitLabProjectMergeRequests_DataSource_Basic(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	project := testutil.CreateProject(t)
	branch := testutil.CreateBranches(t, project, 1)[0]
	mergeRequest := testutil.CreateMergeRequest(t, user, project, branch.Name, "main")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
				    data "gitlab_project_merge_requests" "int_project" {
				      project = "%d"
				      iids = [%d]
				    }
				    `, project.ID, mergeRequest.IID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.#", "1",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.id", strconv.Itoa(mergeRequest.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.iid", strconv.Itoa(mergeRequest.IID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignee.id", strconv.Itoa(user.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignee.avatar_url", user.AvatarURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignee.name", user.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignee.state", user.State,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignee.username", user.Username,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignee.web_url", user.WebURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignees.0.id", strconv.Itoa(user.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignees.0.avatar_url", user.AvatarURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignees.0.name", user.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignees.0.state", user.State,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignees.0.username", user.Username,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.assignees.0.web_url", user.WebURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.author.id", strconv.Itoa(mergeRequest.Author.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.author.avatar_url", mergeRequest.Author.AvatarURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.author.name", mergeRequest.Author.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.author.state", mergeRequest.Author.State,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.author.username", mergeRequest.Author.Username,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.author.web_url", mergeRequest.Author.WebURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.blocking_discussions_resolved",
						strconv.FormatBool(mergeRequest.BlockingDiscussionsResolved),
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.closed_at",
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.closed_by",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_requests.int_project",
						"merge_requests.0.created_at", mergeRequest.CreatedAt.Format(time.RFC3339),
					),
				),
			},
		},
	})
}
