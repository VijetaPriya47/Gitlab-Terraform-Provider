//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectIssueLabelEvents_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	label := testutil.CreateProjectLabels(t, project.ID, 1)[0]
	// Create an issue in the project
	issue := testutil.CreateProjectIssues(t, project.ID, 1)[0]
	testutil.AddLabelToIssue(t, project.ID, issue.IID, label)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project_issue_label_events" "test" {
				  project = "%s"
				  issue_iid = %d
				}
				`, project.PathWithNamespace, issue.IID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "id", fmt.Sprintf("%s:%d", project.PathWithNamespace, issue.IID)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "issue_iid", strconv.FormatInt(issue.IID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.0.action", "add"),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.0.label.name", label.Name),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.0.label.color", label.Color),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.0.user.state", "active"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_issue_label_events.test", "events.0.created_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_issue_label_events.test", "events.0.user.username"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectIssueLabelEvents_noLabels(t *testing.T) {
	project := testutil.CreateProject(t)
	issue := testutil.CreateProjectIssues(t, project.ID, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project_issue_label_events" "test" {
				  project    = "%s"
				  issue_iid  = %d
				}
				`, project.PathWithNamespace, issue.IID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "issue_iid", strconv.FormatInt(issue.IID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.#", "0"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectIssueLabelEvents_fiveLabels(t *testing.T) {
	project := testutil.CreateProject(t)
	labels := testutil.CreateProjectLabels(t, project.ID, 5)
	issue := testutil.CreateProjectIssues(t, project.ID, 1)[0]

	for _, label := range labels {
		testutil.AddLabelToIssue(t, project.ID, issue.IID, label)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project_issue_label_events" "test" {
				  project    = "%s"
				  issue_iid  = %d
				}
				`, project.PathWithNamespace, issue.IID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "id", fmt.Sprintf("%s:%d", project.PathWithNamespace, issue.IID)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "issue_iid", strconv.FormatInt(issue.IID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.#", "5"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectIssueLabelEvents_pagination(t *testing.T) {
	// Skip this test in short mode due to resource intensity
	if testing.Short() {
		t.Skip("Skipping pagination test in short mode")
	}

	project := testutil.CreateProject(t)
	issue := testutil.CreateProjectIssues(t, project.ID, 1)[0]

	// Create 21 distinct labels to generate 21 label events
	// (21 exceeds default page size of 20 to trigger pagination)
	labels := testutil.CreateProjectLabels(t, project.ID, 21)
	for _, label := range labels {
		testutil.AddLabelToIssue(t, project.ID, issue.IID, label)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project_issue_label_events" "test" {
				  project    = "%s"
				  issue_iid  = %d
				}
				`, project.PathWithNamespace, issue.IID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "id", fmt.Sprintf("%s:%d", project.PathWithNamespace, issue.IID)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "issue_iid", strconv.FormatInt(issue.IID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.#", "20"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectIssueLabelEvents_multi_page(t *testing.T) {
	// Skip this test in short mode due to resource intensity
	if testing.Short() {
		t.Skip("Skipping pagination test in short mode")
	}

	project := testutil.CreateProject(t)
	issue := testutil.CreateProjectIssues(t, project.ID, 1)[0]

	labels := testutil.CreateProjectLabels(t, project.ID, 40)
	for _, label := range labels {
		testutil.AddLabelToIssue(t, project.ID, issue.IID, label)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project_issue_label_events" "test" {
				  project    = "%s"
				  issue_iid  = %d
				  pages_returned = 2
				}
				`, project.PathWithNamespace, issue.IID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "id", fmt.Sprintf("%s:%d", project.PathWithNamespace, issue.IID)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "project", project.PathWithNamespace),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "issue_iid", strconv.FormatInt(issue.IID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue_label_events.test", "events.#", "40"),
				),
			},
		},
	})
}
