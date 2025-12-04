//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabPipelineSchedules_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	schedule1, _ := testutil.CreateScheduledPipeline(t, project.ID, "main")
	schedule2, _ := testutil.CreateScheduledPipeline(t, project.ID, "main")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_pipeline_schedules" "this" {
						project = %d
					}
					`,
					project.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.id", strconv.FormatInt(schedule1.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.description", schedule1.Description),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.ref", schedule1.Ref),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.cron", schedule1.Cron),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.cron_timezone", schedule1.CronTimezone),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.next_run_at", schedule1.NextRunAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.active", strconv.FormatBool(schedule1.Active)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.created_at", schedule1.CreatedAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.updated_at", schedule1.UpdatedAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.owner.id", strconv.FormatInt(schedule1.Owner.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.owner.name", schedule1.Owner.Name),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.owner.username", schedule1.Owner.Username),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.owner.state", schedule1.Owner.State),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.owner.avatar_url", schedule1.Owner.AvatarURL),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.0.owner.web_url", schedule1.Owner.WebURL),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.id", strconv.FormatInt(schedule2.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.description", schedule2.Description),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.ref", schedule2.Ref),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.cron", schedule2.Cron),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.cron_timezone", schedule2.CronTimezone),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.next_run_at", schedule2.NextRunAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.active", strconv.FormatBool(schedule2.Active)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.created_at", schedule2.CreatedAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.updated_at", schedule2.UpdatedAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.owner.id", strconv.FormatInt(schedule2.Owner.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.owner.name", schedule2.Owner.Name),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.owner.username", schedule2.Owner.Username),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.owner.state", schedule2.Owner.State),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.owner.avatar_url", schedule2.Owner.AvatarURL),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedules.this", "pipeline_schedules.1.owner.web_url", schedule2.Owner.WebURL),
				),
			},
		},
	})
}
