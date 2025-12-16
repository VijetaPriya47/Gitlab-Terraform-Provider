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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

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

func TestAcc_GitlabProjectLabel_schemaMigrationV0toV2(t *testing.T) {
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
						VersionConstraint: "~> 15.7", // Before V1 schema, produces V0 state.
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
