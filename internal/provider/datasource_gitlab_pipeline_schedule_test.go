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

func TestAccDataGitlabPipelineSchedule_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	schedule, _ := testutil.CreateScheduledPipeline(t, project.ID, "main")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_pipeline_schedule" "this" {
						project = %d
						pipeline_schedule_id = %d
					}
					`,
					project.ID,
					schedule.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "project", strconv.Itoa(project.ID)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "description", schedule.Description),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "ref", schedule.Ref),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "cron", schedule.Cron),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "cron_timezone", schedule.CronTimezone),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "next_run_at", schedule.NextRunAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "active", strconv.FormatBool(schedule.Active)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "created_at", schedule.CreatedAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "updated_at", schedule.UpdatedAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "owner.id", strconv.Itoa(schedule.Owner.ID)),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "owner.name", schedule.Owner.Name),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "owner.username", schedule.Owner.Username),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "owner.state", schedule.Owner.State),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "owner.avatar_url", schedule.Owner.AvatarURL),
					resource.TestCheckResourceAttr("data.gitlab_pipeline_schedule.this", "owner.web_url", schedule.Owner.WebURL),
				),
			},
		},
	})
}
