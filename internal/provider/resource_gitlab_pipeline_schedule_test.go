//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabPipelineSchedule_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    gitlabPipelineScheduleResourceModelSchema0
		expectedV1State gitlabPipelineScheduleResourceModel
	}{
		{
			name: "Project With ID",
			givenV0State: gitlabPipelineScheduleResourceModelSchema0{
				Project: types.StringValue("99"),
				ID:      types.StringValue("42"),
			},
			expectedV1State: gitlabPipelineScheduleResourceModel{
				Project: types.StringValue("99"),
				ID:      types.StringValue("99:42"),
			},
		},
		{
			name: "Project With Namespace",
			givenV0State: gitlabPipelineScheduleResourceModelSchema0{
				Project: types.StringValue("foo/bar"),
				ID:      types.StringValue("42"),
			},
			expectedV1State: gitlabPipelineScheduleResourceModel{
				Project: types.StringValue("foo/bar"),
				ID:      types.StringValue("foo/bar:42"),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabPipelineScheduleStateUpgradeV0ToV1(context.Background(), &tc.givenV0State)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, *actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, *actualV1State)
			}
		})

	}
}

func TestAccGitlabPipelineSchedule_SchemaMigration0_1(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Even though we can usually ignore the `refs/heads`, that logic wasn't in place in the old
	// provider, so the full ref is required for backwards compatibility of the old provider.
	config := fmt.Sprintf(`
	resource "gitlab_pipeline_schedule" "schedule" {
		project = "%d"
		description = "Pipeline Schedule"
		ref = "refs/heads/%s"
		cron = "0 1 * * *"
	}
		`, testProject.ID, testProject.DefaultBranch)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabPipelineScheduleDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 15.7.0", // Earliest 15.X deployment
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
			},
		},
	})
}

func TestAccGitlabPipelineSchedule_takeOwnershipWithChanges(t *testing.T) {
	var schedule gitlab.PipelineSchedule

	// Set up project, user, role mapping and personal access token.
	project := testutil.CreateProject(t)
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembersWithAccessLevel(t, project.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)
	userPAT := testutil.CreatePersonalAccessToken(t, user)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabPipelineScheduleDestroy,
		Steps: []resource.TestStep{
			// Create a Pipeline Schedule with our custom user
			{
				// lintignore:AT004  // we need the provider configuration here to create the schedule with a different user
				Config: fmt.Sprintf(`
				provider "gitlab" {
					token = "%s"
				}

				resource "gitlab_pipeline_schedule" "schedule" {
					project = "%d"
					description = "Schedule"
					ref = "refs/heads/%s"
					cron = "0 4 * * *"
					active = false
				}
				`, userPAT.Token, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "owner", fmt.Sprintf("%d", user.ID)),
				),
			},
			// Let the provider take the ownership on the Pipeline Schedule (with changes)
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule" "schedule" {
					project = "%d"
					description = "Schedule Updated"
					ref = "refs/heads/%s"
					cron = "0 4 * * *"
					active = false
					take_ownership = true
				}
					`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "owner", "1"),
				),
			},
			// Verify upstream attributes with an import
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"take_ownership",
				},
			},
		},
	})
}

func TestAccGitlabPipelineSchedule_takeOwnershipWithoutChanges(t *testing.T) {
	var schedule gitlab.PipelineSchedule

	// Set up project, user, role mapping and personal access token.
	project := testutil.CreateProject(t)
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembersWithAccessLevel(t, project.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)
	userPAT := testutil.CreatePersonalAccessToken(t, user)

	// Wait some time to ensure that membership changes have propogated in the background processes.
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(20 * time.Second)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabPipelineScheduleDestroy,
		Steps: []resource.TestStep{
			// Create a Pipeline Schedule with our custom user
			{
				// lintignore:AT004  // we need the provider configuration here to create the schedule with a different user
				Config: fmt.Sprintf(`
				provider "gitlab" {
					token = "%s"
				}

				resource "gitlab_pipeline_schedule" "schedule" {
					project = "%d"
					description = "Schedule"
					ref = "refs/heads/%s"
					cron = "0 4 * * *"
					active = false
					take_ownership = true
				}
				`, userPAT.Token, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "owner", fmt.Sprintf("%d", user.ID)),
				),
			},
			// Let the provider take the ownership on the Pipeline Schedule (with no changes)
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule" "schedule" {
					project = "%d"
					description = "Schedule"
					ref = "refs/heads/%s"
					cron = "0 4 * * *"
					active = false
					take_ownership = true
				}
					`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "owner", "1"),
				),
			},
			// Verify upstream attributes with an import
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"take_ownership",
				},
			},
		},
	})
}

func TestAccGitlabPipelineSchedule_migrateFromSDKToFramework(t *testing.T) {
	var schedule gitlab.PipelineSchedule

	// Set up project
	project := testutil.CreateProject(t)

	// Create common config for testing
	config := fmt.Sprintf(`
		resource "gitlab_pipeline_schedule" "schedule" {
			project = "%d"
			description = "Schedule"
			ref = "refs/heads/%s"
			cron = "0 4 * * *"
			active = false
		}
		`, project.ID, project.DefaultBranch)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabPipelineScheduleDestroy,
		Steps: []resource.TestStep{
			// Create the pipeline in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 16.6",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
				Check:  testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config:                   config,
				Check:                    testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_pipeline_schedule.schedule",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"take_ownership",
				},
			},
		},
	})
}

func TestAccGitlabPipelineSchedule_basic(t *testing.T) {
	var schedule gitlab.PipelineSchedule
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabPipelineScheduleDestroy,
		Steps: []resource.TestStep{
			// Create a project and pipeline schedule with default options
			{
				Config: fmt.Sprintf(`
					  resource "gitlab_pipeline_schedule" "schedule" {
						  project = "%d"
						  description = "Pipeline Schedule"
						  ref = "refs/heads/%s"
						  cron = "0 1 * * *"
					  }`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					testAccCheckGitlabPipelineScheduleAttributes(&schedule, &testAccGitlabPipelineScheduleExpectedAttributes{
						Description:  "Pipeline Schedule",
						Ref:          fmt.Sprintf("refs/heads/%s", project.DefaultBranch),
						Cron:         "0 1 * * *",
						CronTimezone: "UTC",
						Active:       true,
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the pipeline schedule to change the parameters
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule" "schedule" {
				  project = "%d"
				  description = "Schedule"
				  ref = "refs/heads/%s"
				  cron = "0 4 * * *"
				  active = false
				}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					testAccCheckGitlabPipelineScheduleAttributes(&schedule, &testAccGitlabPipelineScheduleExpectedAttributes{
						Description:  "Schedule",
						Ref:          fmt.Sprintf("refs/heads/%s", project.DefaultBranch),
						Cron:         "0 4 * * *",
						CronTimezone: "UTC",
						Active:       false,
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the pipeline schedule to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule" "schedule" {
					project = "%d"
					description = "Pipeline Schedule"
					ref = "refs/heads/%s"
					cron = "0 1 * * *"
				}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					testAccCheckGitlabPipelineScheduleAttributes(&schedule, &testAccGitlabPipelineScheduleExpectedAttributes{
						Description:  "Pipeline Schedule",
						Ref:          fmt.Sprintf("refs/heads/%s", project.DefaultBranch),
						Cron:         "0 1 * * *",
						CronTimezone: "UTC",
						Active:       true,
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func resourceGitlabPipelineScheduleParseID(id string) (string, int, error) {
	project, rawPipelineScheduleID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	pipelineScheduleID, err := strconv.Atoi(rawPipelineScheduleID)
	if err != nil {
		return "", 0, err
	}

	return project, pipelineScheduleID, nil
}

func testAccCheckGitlabPipelineScheduleExists(n string, schedule *gitlab.PipelineSchedule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project, pipelineScheduleID, err := resourceGitlabPipelineScheduleParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		sc, _, err := testutil.TestGitlabClient.PipelineSchedules.GetPipelineSchedule(project, pipelineScheduleID)
		if err != nil {
			if api.Is404(err) {
				return fmt.Errorf("Pipeline Schedule %q does not exist", rs.Primary.ID)
			}
			return err
		}
		*schedule = *sc
		return nil
	}
}

type testAccGitlabPipelineScheduleExpectedAttributes struct {
	Description  string
	Ref          string
	Cron         string
	CronTimezone string
	Active       bool
}

func testAccCheckGitlabPipelineScheduleAttributes(schedule *gitlab.PipelineSchedule, want *testAccGitlabPipelineScheduleExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if schedule.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", schedule.Description, want.Description)
		}
		if schedule.Ref != want.Ref {
			return fmt.Errorf("got ref %q; want %q", schedule.Ref, want.Ref)
		}

		if schedule.Cron != want.Cron {
			return fmt.Errorf("got cron %q; want %q", schedule.Cron, want.Cron)
		}

		if schedule.CronTimezone != want.CronTimezone {
			return fmt.Errorf("got cron_timezone %q; want %q", schedule.CronTimezone, want.CronTimezone)
		}

		if schedule.Active != want.Active {
			return fmt.Errorf("got active %t; want %t", schedule.Active, want.Active)
		}

		return nil
	}
}

func testAccCheckGitlabPipelineScheduleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_pipeline_schedule" {
			continue
		}

		project, pipelineScheduleID, err := resourceGitlabPipelineScheduleParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.PipelineSchedules.GetPipelineSchedule(project, pipelineScheduleID)
		if err == nil {
			return fmt.Errorf("the Pipeline Schedule %d in project %s still exists", pipelineScheduleID, project)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
