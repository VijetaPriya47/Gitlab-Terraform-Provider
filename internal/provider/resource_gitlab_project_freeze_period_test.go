//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectFreezePeriod_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	var schedule gitlab.FreezePeriod

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectFreezePeriodDestroy,
		Steps: []resource.TestStep{
			// Create a project and freeze period with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_freeze_period" "schedule" {
						project       = "%d"
						freeze_start  = "0 23 * * 5"
						freeze_end    =  "0 7 * * 1"
						cron_timezone = "UTC"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectFreezePeriodExists("gitlab_project_freeze_period.schedule", &schedule),
					testAccCheckGitlabProjectFreezePeriodAttributes(&schedule, &testAccGitlabProjectFreezePeriodExpectedAttributes{
						FreezeStart:  "0 23 * * 5",
						FreezeEnd:    "0 7 * * 1",
						CronTimezone: "UTC",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_freeze_period.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the freeze period to change the parameters
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_freeze_period" "schedule" {
						project       = "%d"
						freeze_start  = "0 20 * * 6"
						freeze_end    =  "0 7 * * 3"
						cron_timezone = "EST"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectFreezePeriodExists("gitlab_project_freeze_period.schedule", &schedule),
					testAccCheckGitlabProjectFreezePeriodAttributes(&schedule, &testAccGitlabProjectFreezePeriodExpectedAttributes{
						FreezeStart:  "0 20 * * 6",
						FreezeEnd:    "0 7 * * 3",
						CronTimezone: "EST",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_freeze_period.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the freeze period to get back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_freeze_period" "schedule" {
						project       = "%d"
						freeze_start  = "0 23 * * 5"
						freeze_end    =  "0 7 * * 1"
						cron_timezone = "UTC"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectFreezePeriodExists("gitlab_project_freeze_period.schedule", &schedule),
					testAccCheckGitlabProjectFreezePeriodAttributes(&schedule, &testAccGitlabProjectFreezePeriodExpectedAttributes{
						FreezeStart:  "0 23 * * 5",
						FreezeEnd:    "0 7 * * 1",
						CronTimezone: "UTC",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_freeze_period.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectFreezePeriodExists(n string, freezePeriod *gitlab.FreezePeriod) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		projectID, freezePeriodID, err := projectAndFreezePeriodIDFromID(rs.Primary.ID)
		if err != nil {
			return err
		}

		gotFreezePeriod, _, err := testutil.TestGitlabClient.FreezePeriods.GetFreezePeriod(projectID, freezePeriodID)
		if err != nil {
			return err
		}

		*freezePeriod = *gotFreezePeriod

		return nil
	}
}

type testAccGitlabProjectFreezePeriodExpectedAttributes struct {
	FreezeStart  string
	FreezeEnd    string
	CronTimezone string
}

func testAccCheckGitlabProjectFreezePeriodAttributes(freezePeriod *gitlab.FreezePeriod, want *testAccGitlabProjectFreezePeriodExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if freezePeriod.FreezeStart != want.FreezeStart {
			return fmt.Errorf("got freeze_start %q; want %q", freezePeriod.FreezeStart, want.FreezeStart)
		}
		if freezePeriod.FreezeEnd != want.FreezeEnd {
			return fmt.Errorf("got freeze_end %q; want %q", freezePeriod.FreezeEnd, want.FreezeEnd)
		}

		if freezePeriod.CronTimezone != want.CronTimezone {
			return fmt.Errorf("got cron_timezone %q; want %q", freezePeriod.CronTimezone, want.CronTimezone)
		}

		return nil
	}
}

func testAccCheckGitlabProjectFreezePeriodDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_freeze_period" {
			continue
		}

		project, freezePeriodID, err := projectAndFreezePeriodIDFromID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Unable to parse resource ID: %s, %s", rs.Primary.ID, err.Error())
		}

		_, _, err = testutil.TestGitlabClient.FreezePeriods.GetFreezePeriod(project, freezePeriodID)
		if api.Is404(err) {
			return nil
		}

		if err != nil {
			return err
		}

		return nil
	}
	return nil
}
