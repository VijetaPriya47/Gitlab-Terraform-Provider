//go:build acceptance
// +build acceptance

package sdk

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectLabel_StateUpgradeV0(t *testing.T) {
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
				"id":      "some-label",
			},
			expectedV1State: map[string]interface{}{
				"project": "99",
				"id":      "99:some-label",
			},
		},
		{
			name: "Project With Namespace",
			givenV0State: map[string]interface{}{
				"project": "foo/bar",
				"id":      "some-label",
			},
			expectedV1State: map[string]interface{}{
				"project": "foo/bar",
				"id":      "foo/bar:some-label",
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

func TestAccGitlabProjectLabel_basic(t *testing.T) {
	var label gitlab.Label
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectLabelDestroy,
		Steps: []resource.TestStep{
			// Create a project and label with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name        = "foo-%d"
						description = "Terraform acceptance tests"
						
						# So that acceptance tests can be run in a gitlab organization
						# with no billing
						visibility_level = "public"
					}
					
					resource "gitlab_project_label" "fixme" {
						project     = "${gitlab_project.foo.id}"
						name        = "FIXME-%d"
						color       = "#ffcc00"
						description = "fix this test"
					}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectLabelExists("gitlab_project_label.fixme", &label),
					testAccCheckGitlabProjectLabelAttributes(&label, &testAccGitlabProjectLabelExpectedAttributes{
						Name:        fmt.Sprintf("FIXME-%d", rInt),
						Color:       "#ffcc00",
						Description: "fix this test",
					}),
				),
			},
			// Update the label to change the parameters
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name        = "foo-%d"
						description = "Terraform acceptance tests"
						
						# So that acceptance tests can be run in a gitlab organization
						# with no billing
						visibility_level = "public"
					}
					
					resource "gitlab_project_label" "fixme" {
						project     = "${gitlab_project.foo.id}"
						name        = "FIXME-%d"
						color       = "#ff0000"
						description = "red label"
					}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectLabelExists("gitlab_project_label.fixme", &label),
					testAccCheckGitlabProjectLabelAttributes(&label, &testAccGitlabProjectLabelExpectedAttributes{
						Name:        fmt.Sprintf("FIXME-%d", rInt),
						Color:       "#ff0000",
						Description: "red label",
					}),
				),
			},
			// Update the label to get back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name        = "foo-%d"
						description = "Terraform acceptance tests"
						
						# So that acceptance tests can be run in a gitlab organization
						# with no billing
						visibility_level = "public"
					}
					
					resource "gitlab_project_label" "fixme" {
						project     = "${gitlab_project.foo.id}"
						name        = "FIXME-%d"
						color       = "#ffcc00"
						description = "fix this test"
					}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectLabelExists("gitlab_project_label.fixme", &label),
					testAccCheckGitlabProjectLabelAttributes(&label, &testAccGitlabProjectLabelExpectedAttributes{
						Name:        fmt.Sprintf("FIXME-%d", rInt),
						Color:       "#ffcc00",
						Description: "fix this test",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_label.fixme",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectLabelExists(n string, label *gitlab.Label) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		projectName, labelName, err := resourceGitlabProjectLabelParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Failed to parse project label id %q: %w", rs.Primary.ID, err)
		}

		l, _, err := testutil.TestGitlabClient.Labels.GetLabel(projectName, labelName)
		*label = *l
		return err
	}
}

type testAccGitlabProjectLabelExpectedAttributes struct {
	Name        string
	Color       string
	Description string
}

func testAccCheckGitlabProjectLabelAttributes(label *gitlab.Label, want *testAccGitlabProjectLabelExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if label.Name != want.Name {
			return fmt.Errorf("got name %q; want %q", label.Name, want.Name)
		}

		if label.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", label.Description, want.Description)
		}

		if label.Color != want.Color {
			return fmt.Errorf("got color %q; want %q", label.Color, want.Color)
		}

		return nil
	}
}

func testAccCheckGitlabProjectLabelDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_label" {
			continue
		}

		projectName, labelName, err := resourceGitlabProjectLabelParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Failed to parse project label id %q: %w", rs.Primary.ID, err)
		}

		_, _, err = testutil.TestGitlabClient.Labels.GetLabel(projectName, labelName)
		if err == nil {
			return fmt.Errorf("Project label %s in project %s still exists", labelName, projectName)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
