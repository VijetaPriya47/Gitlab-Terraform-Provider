//go:build acceptance
// +build acceptance

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

func TestAcc_GitLabProjectMergeRequest_DataSource_InvalidMRIID(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
				    data "gitlab_project_merge_request" "this" {
				      project = "%s"
				      iid = %d
				    }
				    `, project.PathWithNamespace, 3,
				),
				ExpectError: regexp.MustCompile(
					"Unable to read project merge request details.*",
				),
			},
		},
	})
}

func TestAcc_GitLabProjectMergeRequest_DataSource_WithAssignee(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	project := testutil.CreateProject(t)
	branch := testutil.CreateBranches(t, project, 1)[0]
	mergeRequest := testutil.CreateMergeRequest(t, user, project, branch.Name, "main")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
				    data "gitlab_project_merge_request" "int_project" {
				      project = "%d"
				      iid = %d
				    }
				    `, project.ID, mergeRequest.IID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"id", strconv.Itoa(mergeRequest.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"iid", strconv.Itoa(mergeRequest.IID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"project", strconv.Itoa(project.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignee.id", strconv.Itoa(user.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignee.avatar_url", user.AvatarURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignee.name", user.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignee.state", user.State,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignee.username", user.Username,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignee.web_url", user.WebURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignees.0.id", strconv.Itoa(user.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignees.0.avatar_url", user.AvatarURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignees.0.name", user.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignees.0.state", user.State,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignees.0.username", user.Username,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"assignees.0.web_url", user.WebURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"author.id", strconv.Itoa(mergeRequest.Author.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"author.avatar_url", mergeRequest.Author.AvatarURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"author.name", mergeRequest.Author.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"author.state", mergeRequest.Author.State,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"author.username", mergeRequest.Author.Username,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"author.web_url", mergeRequest.Author.WebURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"blocking_discussions_resolved",
						strconv.FormatBool(mergeRequest.BlockingDiscussionsResolved),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"changes_count", "",
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"closed_at",
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"closed_by",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.int_project",
						"created_at", mergeRequest.CreatedAt.Format(time.RFC3339),
					),
				),
			},
		},
	})
}

func TestAcc_GitLabProjectMergeRequest_DataSource_WithoutAssignee(t *testing.T) {
	project := testutil.CreateProject(t)
	branch := testutil.CreateBranches(t, project, 1)[0]
	mergeRequest := testutil.CreateMergeRequest(t, nil, project, branch.Name, "main")
	mergeRequest = testutil.CloseMergeRequest(t, project, mergeRequest)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
				    data "gitlab_project_merge_request" "str_project" {
				      project = "%s"
				      iid = %d
				    }
				    `, project.PathWithNamespace, mergeRequest.IID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"id", strconv.Itoa(mergeRequest.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"iid", strconv.Itoa(mergeRequest.IID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"project", project.PathWithNamespace,
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"assignee",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"assignees.#", strconv.Itoa(0),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"author.id", strconv.Itoa(mergeRequest.Author.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"author.avatar_url", mergeRequest.Author.AvatarURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"author.name", mergeRequest.Author.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"author.state", mergeRequest.Author.State,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"author.username", mergeRequest.Author.Username,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"author.web_url", mergeRequest.Author.WebURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"blocking_discussions_resolved",
						strconv.FormatBool(mergeRequest.BlockingDiscussionsResolved),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"changes_count", "",
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"closed_at", mergeRequest.ClosedAt.Format(time.RFC3339),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"closed_by.id", strconv.Itoa(mergeRequest.ClosedBy.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"closed_by.avatar_url", mergeRequest.ClosedBy.AvatarURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"closed_by.name", mergeRequest.ClosedBy.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"closed_by.state", mergeRequest.ClosedBy.State,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"closed_by.username", mergeRequest.ClosedBy.Username,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"closed_by.web_url", mergeRequest.ClosedBy.WebURL,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_merge_request.str_project",
						"created_at", mergeRequest.CreatedAt.Format(time.RFC3339),
					),
				),
			},
		},
	})
}
