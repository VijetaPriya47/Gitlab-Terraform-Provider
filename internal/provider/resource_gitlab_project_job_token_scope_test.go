//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Based on the pattern found in @resource_gitlab_project_hook_test.go
func TestAcc_GitlabProjectJobTokenScope_SchemaMigration0_1(t *testing.T) {
	project := testutil.CreateProject(t)
	targetProject := testutil.CreateProject(t)

	config := fmt.Sprintf(`
	resource "gitlab_project_job_token_scope" "foo" {
	  project = "%d"
	  target_project_id = %d
	}
	`, project.ID, targetProject.ID)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAcc_GitlabProjectJobTokenScope_CheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "= 17.8.0", // From 17.8.0 deployment
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
			},
		},
	})
}

func TestAcc_GitlabProjectJobTokenScope_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    gitlabProjectJobTokenScopeResourceModelSchema0
		expectedV1State gitlabProjectJobTokenScopeResourceModel
	}{
		{
			name: "Project With ID",
			givenV0State: gitlabProjectJobTokenScopeResourceModelSchema0{
				Project:         types.StringValue("99"),
				TargetProjectID: types.Int64Value(42),
				Id:              types.StringValue("99:42"),
			},
			expectedV1State: gitlabProjectJobTokenScopeResourceModel{
				Project:         types.StringValue("99"),
				TargetProjectID: types.Int64Value(42),
				Id:              types.StringValue("99:project:42"),
			},
		},
		{
			name: "Project With Namespace",
			givenV0State: gitlabProjectJobTokenScopeResourceModelSchema0{
				Project:         types.StringValue("foo/bar"),
				TargetProjectID: types.Int64Value(42),
				Id:              types.StringValue("foo/bar:42"),
			},
			expectedV1State: gitlabProjectJobTokenScopeResourceModel{
				Project:         types.StringValue("foo/bar"),
				TargetProjectID: types.Int64Value(42),
				Id:              types.StringValue("foo/bar:project:42"),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabProjectJobTokenScopeStateUpgradeV0ToV1(context.Background(), &tc.givenV0State)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, *actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, *actualV1State)
			}
		})
	}

}

func TestAcc_GitlabProjectJobTokenScope_basic(t *testing.T) {
	testutil.RunIfAtLeast(t, "16.1")

	// Set up project environment.
	project := testutil.CreateProject(t)
	projectIDStr := strconv.Itoa(project.ID)
	// Target Types
	targetTypeProject := "project"
	targetTypeGroup := "group"
	// Target Projects
	targetProject1 := testutil.CreateProject(t)
	targetProject1IDStr := strconv.Itoa(targetProject1.ID)
	targetProject2 := testutil.CreateProject(t)
	targetProject2IDStr := strconv.Itoa(targetProject2.ID)
	// Target Groups
	targetGroup1 := testutil.CreateGroups(t, 1)[0]
	targetGroup1IDStr := strconv.Itoa(targetGroup1.ID)
	targetGroup2 := testutil.CreateGroups(t, 1)[0]
	targetGroup2IDStr := strconv.Itoa(targetGroup2.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScope_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a basic CI/CD job token scope.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_project_id = %d
				}`, project.ID, targetProject1.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "id", utils.BuildThreePartID(&projectIDStr, &targetTypeProject, &targetProject1IDStr)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "project", projectIDStr),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "target_project_id", targetProject1IDStr),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scope.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the CI/CD job token scope.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_project_id = %d
				}`, project.ID, targetProject2.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "id", utils.BuildThreePartID(&projectIDStr, &targetTypeProject, &targetProject2IDStr)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "project", projectIDStr),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "target_project_id", targetProject2IDStr),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scope.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the CI/CD job token scope to target a group.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_group_id = %d
				}`, project.ID, targetGroup1.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "id", utils.BuildThreePartID(&projectIDStr, &targetTypeGroup, &targetGroup1IDStr)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "project", projectIDStr),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "target_group_id", targetGroup1IDStr),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scope.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the CI/CD job token scope to target a group.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_group_id = %d
				}`, project.ID, targetGroup2.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "id", utils.BuildThreePartID(&projectIDStr, &targetTypeGroup, &targetGroup2IDStr)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "project", projectIDStr),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "target_group_id", targetGroup2IDStr),
				),
			},
		},
	})
}

// lintignore: AT002 // specialized import test
func TestAcc_GitlabProjectJobTokenScope_ImportState(t *testing.T) {
	testutil.RunIfAtLeast(t, "16.1")

	project := testutil.CreateProject(t)
	targetProject := testutil.CreateProject(t)
	targetGroup := testutil.CreateGroups(t, 1)[0]
	targetTypeProject := "project"
	targetTypeGroup := "group"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScope_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a project target first
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_project_id = %d
				}`, project.ID, targetProject.ID),
			},

			// Test importing with the legacy ID format. 'project:target_project_id'
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_project_id = %d
				}`, project.ID, targetProject.ID),
				ResourceName:      "gitlab_project_job_token_scope.this",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%d:%d", project.ID, targetProject.ID),
				ImportStateVerify: true,
			},
			// Test importing with the new ID format. 'project:target_type:target_id' project target
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_project_id = %d
				}`, project.ID, targetProject.ID),
				ResourceName:      "gitlab_project_job_token_scope.this",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%d:%s:%d", project.ID, targetTypeProject, targetProject.ID),
				ImportStateVerify: true,
			},
			// Create a group target
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_group_id = %d
				}`, project.ID, targetGroup.ID),
			},
			// Test importing with the new ID format. 'project:target_type:target_id' group target
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_group_id = %d
				}`, project.ID, targetGroup.ID),
				ResourceName:      "gitlab_project_job_token_scope.this",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%d:%s:%d", project.ID, targetTypeGroup, targetGroup.ID),
				ImportStateVerify: true,
			},
		},
	})
}

func testAcc_GitlabProjectJobTokenScope_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_job_token_scope" {
			projectID := rs.Primary.Attributes["project"]

			// Only try to parse IDs if they exist
			var targetProjectID, targetGroupID int
			if targetProjectIDStr := rs.Primary.Attributes["target_project_id"]; targetProjectIDStr != "" {
				var err error
				targetProjectID, err = strconv.Atoi(targetProjectIDStr)
				if err != nil {
					return fmt.Errorf("Failed to parse target project ID: %w", err)
				}
			}

			if targetGroupIDStr := rs.Primary.Attributes["target_group_id"]; targetGroupIDStr != "" {
				var err error
				targetGroupID, err = strconv.Atoi(targetGroupIDStr)
				if err != nil {
					return fmt.Errorf("Failed to parse target group ID: %w", err)
				}
			}

			projects, _, err := testutil.TestGitlabClient.JobTokenScope.GetProjectJobTokenInboundAllowList(projectID, nil, nil)
			if err != nil {
				return fmt.Errorf("Failed to fetch CI/CD Job Token Scope: %w", err)
			}

			for i := range projects {
				if (targetProjectID != 0 && projects[i].ID == targetProjectID) ||
					(targetGroupID != 0 && projects[i].ID == targetGroupID) {
					return fmt.Errorf("The target project ID (%d) or group ID (%d) still exists in the source project (%s)",
						targetProjectID, targetGroupID, projectID)
				}
			}

			return nil
		}
	}
	return nil
}
