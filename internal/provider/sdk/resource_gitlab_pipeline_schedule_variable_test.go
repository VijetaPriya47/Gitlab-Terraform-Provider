//go:build acceptance
// +build acceptance

package sdk

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabPipelineScheduleVariable_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    map[string]interface{}
		expectedV1State map[string]interface{}
	}{
		{
			name: "Project With ID",
			givenV0State: map[string]interface{}{
				"project":              "99",
				"pipeline_schedule_id": 42,
				"key":                  "some-key",
				"id":                   "42:some-key",
			},
			expectedV1State: map[string]interface{}{
				"project":              "99",
				"pipeline_schedule_id": 42,
				"key":                  "some-key",
				"id":                   "99:42:some-key",
			},
		},
		{
			name: "Project With ID and pipeline schedule id as float",
			givenV0State: map[string]interface{}{
				"project":              "99",
				"pipeline_schedule_id": 42.0,
				"key":                  "some-key",
				"id":                   "42:some-key",
			},
			expectedV1State: map[string]interface{}{
				"project":              "99",
				"pipeline_schedule_id": 42.0,
				"key":                  "some-key",
				"id":                   "99:42:some-key",
			},
		},
		{
			name: "Project With Namespace",
			givenV0State: map[string]interface{}{
				"project":              "foo/bar",
				"pipeline_schedule_id": 42,
				"key":                  "some-key",
				"id":                   "42:some-key",
			},
			expectedV1State: map[string]interface{}{
				"project":              "foo/bar",
				"pipeline_schedule_id": 42,
				"key":                  "some-key",
				"id":                   "foo/bar:42:some-key",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabProjectLabelStateUpgradeV0(context.Background(), tc.givenV0State, nil)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, actualV1State)
			}
		})
	}
}

func TestAccGitlabPipelineScheduleVariable_SchemaMigration0_1(t *testing.T) {
	project := testutil.CreateProject(t)
	schedule, err := testutil.CreateScheduledPipeline(t, project.ID, project.DefaultBranch)
	if err != nil {
		t.Fatalf("Failed to create dependent resources %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabPipelineScheduleVariableDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 15.7.0", // Earliest 15.X deployment
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule_variable" "schedule_var" {
					project = "%d"
					pipeline_schedule_id = "%d"
					key = "TERRAFORMED_TEST_VALUE"
					value = "test"
				}
				`, project.ID, schedule.ID),
			},
			{
				// The "id" attribute is updated to "pipeline_schedule_id" in 16.0, but it should still apply properly.
				ProtoV6ProviderFactories: providerFactoriesV6,
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule_variable" "schedule_var" {
					project = "%d"
					pipeline_schedule_id = "%d"
					key = "TERRAFORMED_TEST_VALUE"
					value = "test"
				}
				`, project.ID, schedule.ID),
				PlanOnly: true,
			},
		},
	})
}

func TestAccGitlabPipelineScheduleVariable_basic(t *testing.T) {
	var variable gitlab.PipelineVariable
	project := testutil.CreateProject(t)
	schedule, err := testutil.CreateScheduledPipeline(t, project.ID, project.DefaultBranch)
	if err != nil {
		t.Fatalf("Failed to create dependent resources %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabPipelineScheduleVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule_variable" "schedule_var" {
					project = "%d"
					pipeline_schedule_id = "%d"
					key = "TERRAFORMED_TEST_VALUE"
					value = "test"
				}
				`, project.ID, schedule.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleVariableExists("gitlab_pipeline_schedule_variable.schedule_var", &variable),
					testAccCheckGitlabPipelineScheduleVariableAttributes(&variable, &testAccGitlabPipelineScheduleVariableExpectedAttributes{
						Key:          "TERRAFORMED_TEST_VALUE",
						Value:        "test",
						VariableType: "env_var",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule_variable.schedule_var",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule_variable" "schedule_var" {
					project = "%d"
					pipeline_schedule_id = "%d"
					key = "TERRAFORMED_TEST_VALUE"
					value = "test_updated"
				}
				`, project.ID, schedule.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleVariableExists("gitlab_pipeline_schedule_variable.schedule_var", &variable),
					testAccCheckGitlabPipelineScheduleVariableAttributes(&variable, &testAccGitlabPipelineScheduleVariableExpectedAttributes{
						Key:          "TERRAFORMED_TEST_VALUE",
						Value:        "test_updated",
						VariableType: "env_var",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule_variable.schedule_var",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule_variable" "schedule_var" {
					project = "%d"
					pipeline_schedule_id = "%d"
					key = "TERRAFORMED_TEST_VALUE"
					value = "test"
				}
				`, project.ID, schedule.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleVariableExists("gitlab_pipeline_schedule_variable.schedule_var", &variable),
					testAccCheckGitlabPipelineScheduleVariableAttributes(&variable, &testAccGitlabPipelineScheduleVariableExpectedAttributes{
						Key:          "TERRAFORMED_TEST_VALUE",
						Value:        "test",
						VariableType: "env_var",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule_variable.schedule_var",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_schedule_variable" "schedule_var" {
					project = "%d"
					pipeline_schedule_id = "%d"
					key = "TERRAFORMED_TEST_VALUE"
					value = "test"
					variable_type = "file"
				}
				`, project.ID, schedule.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleVariableExists("gitlab_pipeline_schedule_variable.schedule_var", &variable),
					testAccCheckGitlabPipelineScheduleVariableAttributes(&variable, &testAccGitlabPipelineScheduleVariableExpectedAttributes{
						Key:          "TERRAFORMED_TEST_VALUE",
						Value:        "test",
						VariableType: "file",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule_variable.schedule_var",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabPipelineScheduleVariable_deletedPipeline(t *testing.T) {
	var variable gitlab.PipelineVariable

	project := testutil.CreateProject(t)
	schedule, err := testutil.CreateScheduledPipeline(t, project.ID, project.DefaultBranch)
	if err != nil {
		t.Fatalf("Failed to create dependent resources %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabPipelineScheduleVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule_variable" "schedule_var" {
						project = "%d"
						pipeline_schedule_id = "%d"
						key = "TERRAFORMED_TEST_VALUE"
						value = "test_updated"
					}
					`, project.ID, schedule.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleVariableExists("gitlab_pipeline_schedule_variable.schedule_var", &variable),
					func(s *terraform.State) error {
						// Get client and delete the pipeline we created above
						client := testutil.TestGitlabClient
						_, err := client.PipelineSchedules.DeletePipelineSchedule(project.ID, schedule.ID)
						return err
					},
				),
				// Plan will be non-empty because we've deleted the pipeline
				// However, without the 404 fix on schedule variable, this would error.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckGitlabPipelineScheduleVariableExists(n string, variable *gitlab.PipelineVariable) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project, scheduleID, variableKey, err := resourceGitlabPipelineScheduleVariableParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		pipelineSchedule, _, err := testutil.TestGitlabClient.PipelineSchedules.GetPipelineSchedule(project, scheduleID)
		if err != nil {
			return err
		}

		for _, pipelineVariable := range pipelineSchedule.Variables {
			if pipelineVariable.Key == variableKey {
				*variable = *pipelineVariable
				return nil
			}
		}
		return fmt.Errorf("PipelineScheduleVariable %s does not exist", variable.Key)
	}
}

type testAccGitlabPipelineScheduleVariableExpectedAttributes struct {
	Key          string
	Value        string
	VariableType string
}

func testAccCheckGitlabPipelineScheduleVariableAttributes(variable *gitlab.PipelineVariable, want *testAccGitlabPipelineScheduleVariableExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if variable.Key != want.Key {
			return fmt.Errorf("got key %q; want %q", variable.Key, want.Key)
		}

		if variable.Value != want.Value {
			return fmt.Errorf("got value %s; want %s", variable.Value, want.Value)
		}

		if variable.VariableType != want.VariableType {
			return fmt.Errorf("got variable_type %s; want %s", variable.VariableType, want.VariableType)
		}

		return nil
	}
}

func testAccCheckGitlabPipelineScheduleVariableDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_pipeline_schedule_variable" {
			continue
		}

		project, scheduleID, variableKey, err := resourceGitlabPipelineScheduleVariableParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		gotPS, _, err := testutil.TestGitlabClient.PipelineSchedules.GetPipelineSchedule(project, scheduleID)
		if err == nil {
			for _, v := range gotPS.Variables {
				if v.Key == variableKey {
					return fmt.Errorf("pipeline schedule variable still exists")
				}
			}
		}
	}
	return nil
}
