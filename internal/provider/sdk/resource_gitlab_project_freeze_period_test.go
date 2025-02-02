//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectFreezePeriod_basic(t *testing.T) {
	var schedule gitlab.FreezePeriod
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project and freeze period with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name        = "foo-%d"
						description = "Terraform acceptance tests"
						
						# So that acceptance tests can be run in a gitlab organization
						# with no billing
						visibility_level = "public"
					}
					
					resource "gitlab_project_freeze_period" "schedule" {
						project       = gitlab_project.foo.id
						freeze_start  = "0 23 * * 5"
						freeze_end    =  "0 7 * * 1"
						cron_timezone = "UTC"
					}
				`, rInt),
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
					resource "gitlab_project" "foo" {
						name        = "foo-%d"
						description = "Terraform acceptance tests"
						
						# So that acceptance tests can be run in a gitlab organization
						# with no billing
						visibility_level = "public"
					}
					
					resource "gitlab_project_freeze_period" "schedule" {
						project       = gitlab_project.foo.id
						freeze_start  = "0 20 * * 6"
						freeze_end    =  "0 7 * * 3"
						cron_timezone = "EST"
					}
				`, rInt),
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
					resource "gitlab_project" "foo" {
						name        = "foo-%d"
						description = "Terraform acceptance tests"
						
						# So that acceptance tests can be run in a gitlab organization
						# with no billing
						visibility_level = "public"
					}
					
					resource "gitlab_project_freeze_period" "schedule" {
						project       = gitlab_project.foo.id
						freeze_start  = "0 23 * * 5"
						freeze_end    =  "0 7 * * 1"
						cron_timezone = "UTC"
					}
				`, rInt),
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
