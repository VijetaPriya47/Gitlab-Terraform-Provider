//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabProjectLabel_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    gitlabProjectLabelResourceModel
		expectedV1State gitlabProjectLabelResourceModel
	}{
		{
			name: "Project With ID",
			givenV0State: gitlabProjectLabelResourceModel{
				Project: types.StringValue("99"),
				ID:      types.StringValue("some-label"),
			},
			expectedV1State: gitlabProjectLabelResourceModel{
				Project: types.StringValue("99"),
				ID:      types.StringValue("99:some-label"),
			},
		},
		{
			name: "Project With Namespace",
			givenV0State: gitlabProjectLabelResourceModel{
				Project: types.StringValue("foo/bar"),
				ID:      types.StringValue("some-label"),
			},
			expectedV1State: gitlabProjectLabelResourceModel{
				Project: types.StringValue("foo/bar"),
				ID:      types.StringValue("foo/bar:some-label"),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State := resourceGitlabProjectLabelStateUpgradeV0(context.Background(), &tc.givenV0State)

			if !reflect.DeepEqual(tc.expectedV1State, *actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, actualV1State)
			}
		})
	}
}

func TestAcc_GitlabProjectLabel_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectLabelDestroy,
		Steps: []resource.TestStep{
			// Create a project and label with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "fixme" {
						project     = "%d"
						name        = "FIXME-%d"
						color       = "#ffcc00"
						description = "fix this test"
					}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "name", fmt.Sprintf("FIXME-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "color", "#ffcc00"),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "description", "fix this test"),
				),
			},
			// Update the label to change the color and description
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "fixme" {
						project     = "%d"
						name        = "FIXME-%d"
						color       = "#ff0000"
						description = "red label"
					}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "name", fmt.Sprintf("FIXME-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "color", "#ff0000"),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "description", "red label"),
				),
			},
			// Update the label name (this should NOT force replacement)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "fixme" {
						project     = "%d"
						name        = "RENAMED-%d"
						color       = "#ff0000"
						description = "red label"
					}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "name", fmt.Sprintf("RENAMED-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "color", "#ff0000"),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "description", "red label"),
				),
			},
			// Update the label to get back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "fixme" {
						project     = "%d"
						name        = "FIXME-%d"
						color       = "#ffcc00"
						description = "fix this test"
					}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "name", fmt.Sprintf("FIXME-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "color", "#ffcc00"),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "description", "fix this test"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_label.fixme",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the label to use a named color
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "fixme" {
						project     = "%d"
						name        = "FIXME-%d"
						color       = "forestgreen"
						description = "fix this test"
					}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "name", fmt.Sprintf("FIXME-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "color", "forestgreen"),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "color_hex", "#228B22"),
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "description", "fix this test"),
				),
			},
		},
	})
}

// Test that the old "gitlab_label" name for the resource works in addition
// to the new "gitlab_project_label" name. This test should be removed in
// %19.0 when we remove the old name.
func TestAcc_GitlabProjectLabel_deprecatedResourceName(t *testing.T) {
	project := testutil.CreateProject(t)
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectLabelDestroy,
		Steps: []resource.TestStep{
			// Create a project and label with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_label" "fixme" {
						project     = "%d"
						name        = "FIXME-%d"
						color       = "#ffcc00"
						description = "fix this test"
					}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_label.fixme", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_label.fixme", "name", fmt.Sprintf("FIXME-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_label.fixme", "color", "#ffcc00"),
					resource.TestCheckResourceAttr("gitlab_label.fixme", "description", "fix this test"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_label.fixme",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectLabel_schemaMigrationV1toV2(t *testing.T) {
	project := testutil.CreateProject(t)
	rInt := acctest.RandInt()
	labelName := fmt.Sprintf("test-label-%d", rInt)

	legacyConfig := fmt.Sprintf(`
	resource "gitlab_label" "foo" {
		project     = "%d"
		name        = "%s"
		color       = "#FF0000"
		description = "Project label description"
	}
	`, project.ID, labelName)

	newConfig := fmt.Sprintf(`
	resource "gitlab_label" "foo" {
		project     = "%d"
		name        = "%s"
		color       = "#FF0000"
		description = "Project label description"
	}
	`, project.ID, labelName)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectLabelDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.3.0", // Before framework migration, produces V1 state.
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: legacyConfig,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   newConfig,
			},
		},
	})
}

func TestAcc_GitlabProjectLabel_migrateFromSDKToFramework(t *testing.T) {
	project := testutil.CreateProject(t)
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectLabelDestroy,
		Steps: []resource.TestStep{
			// Create the label in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.11",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "foo" {
						project     = "%d"
						name        = "test-label"
						color       = "#FF0000"
						description = "Project label description"
					}
				`, project.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_project_label.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "foo" {
						project = "%d"
						name = "test-label"
						color = "#FF0000"
						description = "Group label description"
					}
				`, project.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_project_label.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_label.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func TestAcc_GitlabProjectLabel_migrateFromSDKToFramework_deprecatedResourceName(t *testing.T) {
	project := testutil.CreateProject(t)
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectLabelDestroy,
		Steps: []resource.TestStep{
			// Create the label in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.11",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_label" "foo" {
						project     = "%d"
						name        = "test-label"
						color       = "#FF0000"
						description = "Project label description"
					}
				`, project.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_label.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_label" "foo" {
						project = "%d"
						name = "test-label"
						color = "#FF0000"
						description = "Group label description"
					}
				`, project.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_label.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_label.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func TestAcc_GitlabProjectLabel_regressionNullDescription(t *testing.T) {
	project := testutil.CreateProject(t)
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectLabelDestroy,
		Steps: []resource.TestStep{
			// Create a label with no description
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "fixme" {
						project     = "%d"
						name        = "FIXME-%d"
						color       = "#ffcc00"
					}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "description", ""),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_label.fixme",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the label to include a description
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_label" "fixme" {
						project     = "%d"
						name        = "FIXME-%d"
						color       = "#ffcc00"
						description = "fix this test"
					}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_label.fixme", "description", "fix this test"),
				),
			},
		},
	})
}

// TestAcc_GitlabProjectLabel_stateMove verifies that the moved block works
// when migrating from gitlab_label to gitlab_project_label.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAcc_GitlabProjectLabel_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectLabelDestroy,
		Steps: []resource.TestStep{
			// Create a label using the old gitlab_label resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_label" "old" {
					project     = %d
					name        = "moved-label"
					color       = "#FF0000"
					description = "Label to be moved"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_label.old", "name", "moved-label"),
					resource.TestCheckResourceAttr("gitlab_label.old", "color", "#FF0000"),
					resource.TestCheckResourceAttr("gitlab_label.old", "description", "Label to be moved"),
					resource.TestCheckResourceAttrSet("gitlab_label.old", "id"),
					resource.TestCheckResourceAttrSet("gitlab_label.old", "label_id"),
				),
			},
			// Move the state to the new gitlab_project_label resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_label" "new" {
					project     = %d
					name        = "moved-label"
					color       = "#FF0000"
					description = "Label to be moved"
				}

				moved {
					from = gitlab_label.old
					to   = gitlab_project_label.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_label.new", "name", "moved-label"),
					resource.TestCheckResourceAttr("gitlab_project_label.new", "color", "#FF0000"),
					resource.TestCheckResourceAttr("gitlab_project_label.new", "description", "Label to be moved"),
					resource.TestCheckResourceAttrSet("gitlab_project_label.new", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_label.new", "label_id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:      "gitlab_project_label.new",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectLabelDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_label" && rs.Type != "gitlab_label" {
			continue
		}

		projectName, labelID, err := (&gitlabProjectLabelResourceModel{}).ResourceGitlabProjectLabelParseID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Failed to parse project label id %q: %w", rs.Primary.ID, err)
		}

		_, _, err = testutil.TestGitlabClient.Labels.GetLabel(projectName, strconv.FormatInt(int64(labelID), 10))
		if err == nil {
			return fmt.Errorf("Project label %d in project %s still exists", labelID, projectName)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
