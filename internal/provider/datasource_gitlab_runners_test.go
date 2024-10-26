//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabRunner_basic(t *testing.T) {

	optsRunnerInstance := gitlab.CreateUserRunnerOptions{
		RunnerType: gitlab.Ptr("instance_type"),
		TagList:    gitlab.Ptr([]string{"cats", "are", "amazing"}),
	}
	runner := testutil.CreateRunnerWithOptions(t, &optsRunnerInstance)

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "gitlab_runners" "this" {
						paused   = false
						status   = "never_contacted"
						tag_list = ["cats", "are", "amazing"]
						type     = "instance_type"
					}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// check resource attributes
					resource.TestCheckResourceAttr("data.gitlab_runners.this", "paused", "false"),
					resource.TestCheckResourceAttr("data.gitlab_runners.this", "status", "never_contacted"),
					resource.TestCheckResourceAttr("data.gitlab_runners.this", "tag_list.2", "cats"), // sorted alphabetically in state
					resource.TestCheckResourceAttr("data.gitlab_runners.this", "type", "instance_type"),
					resource.TestCheckResourceAttrSet("data.gitlab_runners.this", "runners.0.%"),
					// check runner attributes
					resource.TestCheckResourceAttr("data.gitlab_runners.this", "runners.0.id", strconv.Itoa(runner.ID)),
					resource.TestCheckResourceAttr("data.gitlab_runners.this", "runners.0.description", ""),
					resource.TestCheckResourceAttrSet("data.gitlab_runners.this", "runners.0.paused"),
					resource.TestCheckResourceAttrSet("data.gitlab_runners.this", "runners.0.is_shared"),
					resource.TestCheckResourceAttrSet("data.gitlab_runners.this", "runners.0.runner_type"),
					resource.TestCheckResourceAttrSet("data.gitlab_runners.this", "runners.0.online"),
					resource.TestCheckResourceAttrSet("data.gitlab_runners.this", "runners.0.status"),
				),
			},
		},
	})
}

func TestAccDataGitlabRunner_filter(t *testing.T) {

	optsRunnerInstance := gitlab.CreateUserRunnerOptions{
		RunnerType: gitlab.Ptr("instance_type"),
	}
	runnerInstance := testutil.CreateRunnerWithOptions(t, &optsRunnerInstance)

	project := testutil.CreateProject(t)
	optsRunnerProject := gitlab.CreateUserRunnerOptions{
		RunnerType: gitlab.Ptr("project_type"),
		ProjectID:  gitlab.Ptr(project.ID),
		TagList:    gitlab.Ptr([]string{"cats", "are", "amazing"}),
	}
	runnerProject := testutil.CreateRunnerWithOptions(t, &optsRunnerProject)

	optsRunnerPaused := gitlab.CreateUserRunnerOptions{
		RunnerType: gitlab.Ptr("instance_type"),
		Paused:     gitlab.Ptr(true),
	}
	runnerPaused := testutil.CreateRunnerWithOptions(t, &optsRunnerPaused)

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_runners" "this" {
						type = "%s"
					}
					`,
					*optsRunnerInstance.RunnerType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_runners.this",
						"runners.0.runner_type",
						*optsRunnerInstance.RunnerType,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_runners.this",
						"runners.0.id",
						strconv.Itoa(runnerInstance.ID),
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_runners.this",
						"runners.2",
					),
				),
			},
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_runners" "this" {
						tag_list = ["%s"]
					}
					`,
					strings.Join(*optsRunnerProject.TagList, `", "`),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_runners.this",
						"runners.0.id",
						strconv.Itoa(runnerProject.ID),
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_runners.this",
						"runners.1",
					),
				),
			},
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_runners" "this" {
						paused = %t
					}
					`,
					*optsRunnerPaused.Paused,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_runners.this",
						"runners.0.id",
						strconv.Itoa(runnerPaused.ID),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_runners.this",
						"runners.0.paused",
						strconv.FormatBool(*optsRunnerPaused.Paused),
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_runners.this",
						"runners.1",
					),
				),
			},
			{
				Config: `
					data "gitlab_runners" "this" {
						status = "never_contacted"
					}
					`,
				Check: resource.TestCheckResourceAttrSet(
					"data.gitlab_runners.this",
					"runners.2.%",
				),
			},
			{
				Config: `
					data "gitlab_runners" "this" {
						status = "online"
					}
					`,
				Check: resource.TestCheckNoResourceAttr(
					"data.gitlab_runners.this",
					"runners.0",
				),
			},
		},
	})
}
