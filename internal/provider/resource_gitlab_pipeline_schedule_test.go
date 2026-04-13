//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
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
				Inputs:  types.SetNull(pipelineScheduleInputSchema().NestedObject.Type()),
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
				Inputs:  types.SetNull(pipelineScheduleInputSchema().NestedObject.Type()),
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

func TestAccGitlabPipelineSchedule_takeOwnershipWithChanges(t *testing.T) {
	var schedule gitlab.PipelineSchedule

	// Set up project, user, role mapping and personal access token.
	project := testutil.CreateProject(t)
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembersWithAccessLevel(t, project.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)

	// Wait some time to ensure that membership changes have propogated in the background processes.
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(60 * time.Second)
	userPAT := testutil.CreatePersonalAccessToken(t, user)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

	// Wait some time to ensure that membership changes have propogated in the background processes.
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(60 * time.Second)
	userPAT := testutil.CreatePersonalAccessToken(t, user)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

func TestAccGitlabPipelineSchedule_basic(t *testing.T) {
	var schedule gitlab.PipelineSchedule
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

func resourceGitlabPipelineScheduleParseID(id string) (string, int64, error) {
	project, rawPipelineScheduleID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	pipelineScheduleID, err := strconv.ParseInt(rawPipelineScheduleID, 10, 64)
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

func TestAccGitlabPipelineSchedule_withInputs(t *testing.T) {
	var schedule gitlab.PipelineSchedule
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPipelineScheduleDestroy,
		Steps: []resource.TestStep{
			// Create a pipeline schedule with inputs
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule with Inputs"
						ref = "refs/heads/%s"
						cron = "0 1 * * *"

						inputs = [
							{
								name  = "deploy_strategy"
								value = "rolling"
							},
							{
								name  = "environment"
								value = "staging"
							}
						]
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "inputs.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "deploy_strategy",
						"value": "rolling",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "environment",
						"value": "staging",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update inputs - modify one and add a new one
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule with Inputs"
						ref = "refs/heads/%s"
						cron = "0 1 * * *"

						inputs = [
							{
								name  = "deploy_strategy"
								value = "blue-green"
							},
							{
								name  = "environment"
								value = "staging"
							},
							{
								name  = "feature_flag"
								value = "enabled"
							}
						]
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "inputs.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "deploy_strategy",
						"value": "blue-green",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "environment",
						"value": "staging",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "feature_flag",
						"value": "enabled",
					}),
				),
			},
			// Verify Import after update
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove one input (test deletion with destroy flag)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule with Inputs"
						ref = "refs/heads/%s"
						cron = "0 1 * * *"

						inputs = [
							{
								name  = "deploy_strategy"
								value = "blue-green"
							},
							{
								name  = "environment"
								value = "production"
							}
						]
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "inputs.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "deploy_strategy",
						"value": "blue-green",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "environment",
						"value": "production",
					}),
				),
			},
			// Verify Import after deletion
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove all inputs
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule with Inputs"
						ref = "refs/heads/%s"
						cron = "0 1 * * *"
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "inputs.#", "0"),
				),
			},
			// Verify Import after removing all inputs
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabPipelineSchedule_withNonStringInputs(t *testing.T) {
	var schedule gitlab.PipelineSchedule
	project := testutil.CreateProject(t)

	// Wait for the default branch protection to be created asynchronously
	// This ensures we have a consistent state before unprotecting
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	branchProtected := false
	for !branchProtected {
		select {
		case <-ctx.Done():
			t.Fatal("timed out waiting for default branch to be protected")
		case <-ticker.C:
			_, _, err := testutil.TestGitlabClient.ProtectedBranches.GetProtectedBranch(project.ID, project.DefaultBranch)
			if err == nil {
				branchProtected = true
			} else if !api.Is404(err) {
				t.Fatalf("unexpected error checking branch protection: %v", err)
			}
			// If 404, branch not protected yet, continue waiting
		}
	}

	// Remove the default branch protection to let us add a CI file directly to the default branch
	_, err := testutil.TestGitlabClient.ProtectedBranches.UnprotectRepositoryBranches(project.ID, project.DefaultBranch)
	if err != nil {
		t.Fatal(err)
	}

	// Create a CI/CD file with inputs that are non-string types. Otherwise the inputs
	// default to string type, which cause all other tests to pass.
	_, _, err = testutil.TestGitlabClient.RepositoryFiles.CreateFile(project.ID, ".gitlab-ci.yml", &gitlab.CreateFileOptions{
		Branch:        &project.DefaultBranch,
		Encoding:      gitlab.Ptr("text"),
		CommitMessage: gitlab.Ptr("Add CI File"),
		Content: gitlab.Ptr(`
spec:
  inputs:
    enabled:
      default: true
      type: boolean
    timeout:
      default: 1
      type: number
    threshold:
      default: 2.71828
      type: number
    tags:
      default:
        - one
        - two
      type: array
    environment:
      default: stuff
      type: string
---
stages:
  - test
test-job:
  stage: test
  script:
    - echo "hello world"
`),
	})
	if err != nil {
		t.Fatal(err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPipelineScheduleDestroy,
		Steps: []resource.TestStep{
			// Create a pipeline schedule with inputs that GitLab API may type-convert
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule with Non-String Inputs"
						ref = "refs/heads/%s"
						cron = "0 1 * * *"

						inputs = [
							{
								name  = "enabled"
								value = "true"
							},
							{
								name  = "timeout"
								value = "300"
							},
							{
								name  = "threshold"
								value = "3.14159"
							},
							{
								name  = "tags"
								value = "dev,test"
							},
							{
								name  = "environment"
								value = "staging"
							}
						]
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "inputs.#", "5"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "enabled",
						"value": "true",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "timeout",
						"value": "300",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "threshold",
						"value": "3.14159",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "tags",
						"value": "dev,test",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "environment",
						"value": "staging",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update inputs with different values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule with Non-String Inputs"
						ref = "refs/heads/%s"
						cron = "0 1 * * *"

						inputs = [
							{
								name  = "enabled"
								value = "false"
							},
							{
								name  = "timeout"
								value = "600"
							},
							{
								name  = "threshold"
								value = "2.71828"
							},
							{
								name  = "tags"
								value = "prod,staging"
							}
						]
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "inputs.#", "4"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "enabled",
						"value": "false",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "timeout",
						"value": "600",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "threshold",
						"value": "2.71828",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "tags",
						"value": "prod,staging",
					}),
				),
			},
			// Verify Import after update
			{
				ResourceName:      "gitlab_pipeline_schedule.schedule",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabPipelineSchedule_upgradeWithInputs(t *testing.T) {
	var schedule gitlab.PipelineSchedule
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create pipeline schedule with older provider version (v18.9.0)
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "18.7.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule for Upgrade Test"
						ref = "refs/heads/%s"
						cron = "0 2 * * *"
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "description", "Pipeline Schedule for Upgrade Test"),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "cron", "0 2 * * *"),
				),
			},
			// Step 2: Upgrade to current provider version with no config changes
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule for Upgrade Test"
						ref = "refs/heads/%s"
						cron = "0 2 * * *"
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "description", "Pipeline Schedule for Upgrade Test"),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "cron", "0 2 * * *"),
				),
			},
			// Step 3: Add inputs with current provider version
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_schedule" "schedule" {
						project = "%d"
						description = "Pipeline Schedule for Upgrade Test"
						ref = "refs/heads/%s"
						cron = "0 2 * * *"

						inputs = [
							{
								name  = "deploy_env"
								value = "production"
							},
							{
								name  = "auto_deploy"
								value = "true"
							},
							{
								name  = "replicas"
								value = "3"
							}
						]
					}`, project.ID, project.DefaultBranch),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabPipelineScheduleExists("gitlab_pipeline_schedule.schedule", &schedule),
					resource.TestCheckResourceAttr("gitlab_pipeline_schedule.schedule", "inputs.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "deploy_env",
						"value": "production",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "auto_deploy",
						"value": "true",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_pipeline_schedule.schedule", "inputs.*", map[string]string{
						"name":  "replicas",
						"value": "3",
					}),
				),
			},
			// Verify Import after adding inputs
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_pipeline_schedule.schedule",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
		CheckDestroy: testAccCheckGitlabPipelineScheduleDestroy,
	})
}
