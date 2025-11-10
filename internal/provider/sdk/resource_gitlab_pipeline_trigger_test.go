//go:build acceptance

package sdk

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabPipelineTrigger_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    map[string]interface{}
		expectedV1State map[string]interface{}
	}{
		{
			name: "Project With ID",
			givenV0State: map[string]interface{}{
				"project": "99",
				"id":      "42",
			},
			expectedV1State: map[string]interface{}{
				"project": "99",
				"id":      "99:42",
			},
		},
		{
			name: "Project With Namespace",
			givenV0State: map[string]interface{}{
				"project": "foo/bar",
				"id":      "42",
			},
			expectedV1State: map[string]interface{}{
				"project": "foo/bar",
				"id":      "foo/bar:42",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabPipelineTriggerStateUpgradeV0(context.Background(), tc.givenV0State, nil)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, actualV1State)
			}
		})

	}
}

func TestAccGitlabPipelineTrigger_SchemaMigration0_1(t *testing.T) {
	testProject := testutil.CreateProject(t)

	config := fmt.Sprintf(`
	resource "gitlab_pipeline_trigger" "trigger" {
		project = "%d"
		description = "External Pipeline Trigger"
	}
		`, testProject.ID)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabPipelineTriggerDestroy,
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
				ProtoV6ProviderFactories: providerFactoriesV6,
				Config:                   config,
				PlanOnly:                 true,
			},
		},
	})
}

func TestAccGitlabPipelineTrigger_basic(t *testing.T) {
	var trigger gitlab.PipelineTrigger

	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabPipelineTriggerDestroy,
		Steps: []resource.TestStep{
			// Create a project and pipeline trigger with default options
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_pipeline_trigger" "trigger" {
					project = "%d"
					description = "External Pipeline Trigger"
				}
					`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineTriggerExists("gitlab_pipeline_trigger.trigger", &trigger),
					testAccCheckGitlabPipelineTriggerAttributes(&trigger, &testAccGitlabPipelineTriggerExpectedAttributes{
						Description: "External Pipeline Trigger",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_trigger.trigger",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Update the pipeline trigger to change the parameters
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_pipeline_trigger" "trigger" {
				  project = "%d"
				  description = "Trigger"
				}
					`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineTriggerExists("gitlab_pipeline_trigger.trigger", &trigger),
					testAccCheckGitlabPipelineTriggerAttributes(&trigger, &testAccGitlabPipelineTriggerExpectedAttributes{
						Description: "Trigger",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_trigger.trigger",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Update the pipeline trigger to get back to initial settings
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_pipeline_trigger" "trigger" {
					project = "%d"
					description = "External Pipeline Trigger"
				}
					`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineTriggerExists("gitlab_pipeline_trigger.trigger", &trigger),
					testAccCheckGitlabPipelineTriggerAttributes(&trigger, &testAccGitlabPipelineTriggerExpectedAttributes{
						Description: "External Pipeline Trigger",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_trigger.trigger",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func TestAccGitlabPipelineTrigger_readWithDifferentUser(t *testing.T) {
	var trigger gitlab.PipelineTrigger

	// Create a separate user to use with the second apply
	project := testutil.CreateProject(t)
	separateUser := testutil.CreateUsers(t, 1)[0]
	personalToken := testutil.CreatePersonalAccessTokenWithScopes(t, separateUser, []string{"api", "admin_mode"})

	// Add user to project (required for reading triggers)
	testutil.AddProjectMembersWithAccessLevel(t, project.ID, []*gitlab.User{separateUser}, gitlab.MaintainerPermissions)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabPipelineTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_pipeline_trigger" "trigger" {
					project = "%d"
					description = "External Pipeline Trigger"
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineTriggerExists("gitlab_pipeline_trigger.trigger", &trigger),
				),
			},
			{
				// lintignore:AT004  // we need the provider configuration here to read the trigger as a separate user
				Config: fmt.Sprintf(`
				# Instantiate a separate provider so we can use a different user from the first apply
				provider "gitlab" {
					token = "%s"
				}

				resource "gitlab_pipeline_trigger" "trigger" {
					project = "%d"
					description = "External Pipeline Trigger"
				}
				`, personalToken.Token, project.ID),
				Check: resource.ComposeTestCheckFunc(
					// Validate that the token we get from the second apply in state matches the token we have from the first apply..
					func(input *terraform.State) error {
						values := input.RootModule().Resources["gitlab_pipeline_trigger.trigger"].Primary.Attributes
						stateTokenValue := values["token"]

						if stateTokenValue != trigger.Token {
							return fmt.Errorf("Failed to retrieve the expected token value. Expected it to be the same on both tests. Values: %s , %s", stateTokenValue, trigger.Token)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccCheckGitlabPipelineTriggerExists(n string, trigger *gitlab.PipelineTrigger) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project, pipelineTriggerId, err := resourceGitlabPipelineTriggerParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		t, _, err := testutil.TestGitlabClient.PipelineTriggers.GetPipelineTrigger(project, pipelineTriggerId)
		if err != nil {
			if api.Is404(err) {
				return fmt.Errorf("Pipeline Trigger %q does not exist", rs.Primary.ID)
			}
			return err
		}
		*trigger = *t
		return nil
	}
}

type testAccGitlabPipelineTriggerExpectedAttributes struct {
	Description string
}

func testAccCheckGitlabPipelineTriggerAttributes(trigger *gitlab.PipelineTrigger, want *testAccGitlabPipelineTriggerExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if trigger.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", trigger.Description, want.Description)
		}

		return nil
	}
}

func testAccCheckGitlabPipelineTriggerDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_pipeline_trigger" {
			continue
		}

		project, pipelineTriggerId, err := resourceGitlabPipelineTriggerParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.PipelineTriggers.GetPipelineTrigger(project, pipelineTriggerId)
		if err == nil {
			return fmt.Errorf("the Pipeline Trigger %d in project %s still exists", pipelineTriggerId, project)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
