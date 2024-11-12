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
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupLabel_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    map[string]interface{}
		expectedV1State map[string]interface{}
	}{
		{
			name: "Group With ID",
			givenV0State: map[string]interface{}{
				"group": "99",
				"id":    "some-label",
			},
			expectedV1State: map[string]interface{}{
				"group": "99",
				"id":    "99:some-label",
			},
		},
		{
			name: "Group With Namespace",
			givenV0State: map[string]interface{}{
				"group": "foo/bar",
				"id":    "some-label",
			},
			expectedV1State: map[string]interface{}{
				"group": "foo/bar",
				"id":    "foo/bar:some-label",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabGroupLabelStateUpgradeV0(context.Background(), tc.givenV0State, nil)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, actualV1State)
			}
		})

	}
}

func TestAccGitlabGroupLabel_StateUpgradeV1(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV1State    map[string]interface{}
		expectedV2State map[string]interface{}
		errorMsg        string
	}{
		{
			name: "Group With ID",
			givenV1State: map[string]interface{}{
				"label_id": "11",
				"group":    "99",
				"id":       "some-label",
			},
			expectedV2State: map[string]interface{}{
				"label_id": "11",
				"group":    "99",
				"id":       "99:11",
			},
		},
		{
			name: "Group With Namespace",
			givenV1State: map[string]interface{}{
				"label_id": "11",
				"group":    "foo/bar",
				"id":       "some-label",
			},
			expectedV2State: map[string]interface{}{
				"label_id": "11",
				"group":    "foo/bar",
				"id":       "foo/bar:11",
			},
		},
		{
			name: "Invalid ID format",
			givenV1State: map[string]interface{}{
				"group": "foo/bar",
				"id":    "some-label",
			},
			expectedV2State: nil,
			errorMsg:        "failed to parse group label id \"some-label\": Unexpected ID format (\"some-label\"). Expected <part1>:<part2>",
		},
		{
			name: "State does not have the label_id",
			givenV1State: map[string]interface{}{
				"group": "99",
				"id":    "99:FIXME",
			},
			expectedV2State: nil,
			errorMsg:        "failed to fetch label: 404 Not Found",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV2State, err := resourceGitlabGroupLabelStateUpgradeV1(context.Background(), tc.givenV1State, testutil.TestGitlabClient)
			if err != nil && err.Error() != tc.errorMsg {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV2State, actualV2State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV2State, actualV2State)
			}
		})
	}
}

func TestAccGitlabGroupLabel_SchemaMigration0_2(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]

	config := fmt.Sprintf(`
	resource "gitlab_group_label" "fixme" {
	  group       = "%d"
	  name        = "FIXME-%d"
	  color       = "#ffcc00"
	  description = "fix this test"
	}
	`, testGroup.ID, acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupLabelDestroy,
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

func TestAccGitlabGroupLabel_basic(t *testing.T) {
	var label gitlab.GroupLabel
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupLabelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabGroupLabelConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupLabelExists("gitlab_group_label.fixme", &label),
					testAccCheckGitlabGroupLabelAttributes(&label, &testAccGitlabGroupLabelExpectedAttributes{
						Name:        fmt.Sprintf("FIXME-%d", rInt),
						Color:       "#ffcc00",
						Description: "fix this test",
					}),
				),
			},
			{
				Config: testAccGitlabGroupLabelUpdateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupLabelExists("gitlab_group_label.fixme", &label),
					testAccCheckGitlabGroupLabelAttributes(&label, &testAccGitlabGroupLabelExpectedAttributes{
						Name:        fmt.Sprintf("FIXME-AGAIN-%d", rInt),
						Color:       "#ff0000",
						Description: "red label",
					}),
				),
			},
			{
				Config: testAccGitlabGroupLabelConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupLabelExists("gitlab_group_label.fixme", &label),
					testAccCheckGitlabGroupLabelAttributes(&label, &testAccGitlabGroupLabelExpectedAttributes{
						Name:        fmt.Sprintf("FIXME-%d", rInt),
						Color:       "#ffcc00",
						Description: "fix this test",
					}),
				),
			},
			{
				ResourceName:      "gitlab_group_label.fixme",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabGroupLabelExists(n string, label *gitlab.GroupLabel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		groupName, labelId, err := resourceGitlabGroupLabelParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Failed to parse group label id %q: %w", rs.Primary.ID, err)
		}

		l, _, err := testutil.TestGitlabClient.GroupLabels.GetGroupLabel(groupName, labelId)
		*label = *l
		return err
	}
}

type testAccGitlabGroupLabelExpectedAttributes struct {
	Name        string
	Color       string
	Description string
}

func testAccCheckGitlabGroupLabelAttributes(label *gitlab.GroupLabel, want *testAccGitlabGroupLabelExpectedAttributes) resource.TestCheckFunc {
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

func testAccCheckGitlabGroupLabelDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_label" {
			continue
		}

		groupName, labelName, err := resourceGitlabGroupLabelParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Failed to parse group label id %q: %w", rs.Primary.ID, err)
		}

		_, _, err = testutil.TestGitlabClient.GroupLabels.GetGroupLabel(groupName, labelName)
		if err != nil {
			if api.Is404(err) {
				return nil
			}
			return err
		}
		return fmt.Errorf("Group Label %q stil exists", rs.Primary.ID)
	}
	return nil
}

func testAccGitlabGroupLabelConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name             = "foo-%d"
  path             = "foo-%d"
  description      = "Terraform acceptance tests"
  visibility_level = "public"
}

resource "gitlab_group_label" "fixme" {
  group       = "${gitlab_group.foo.id}"
  name        = "FIXME-%d"
  color       = "#ffcc00"
  description = "fix this test"
}
	`, rInt, rInt, rInt)
}

func testAccGitlabGroupLabelUpdateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name             = "foo-%d"
  path             = "foo-%d"
  description      = "Terraform acceptance tests"
  visibility_level = "public"
}

resource "gitlab_group_label" "fixme" {
  group       = "${gitlab_group.foo.id}"
  name        = "FIXME-AGAIN-%d"
  color       = "#ff0000"
  description = "red label"
}
	`, rInt, rInt, rInt)
}
