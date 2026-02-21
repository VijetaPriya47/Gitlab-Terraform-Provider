//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
)

func TestAcc_GitlabProjectIntegrationTelegram_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationTelegramCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Telegram integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_telegram" "this" {
					project = "%s"
					token   = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
					room    = "-1000000000000000"

					notify_only_broken_pipelines = true
					push_events                  = false
					issues_events                = false
					confidential_issues_events   = false
					merge_requests_events        = false
					tag_push_events              = false
					note_events                  = false
					confidential_note_events     = false
					pipeline_events              = false
					wiki_page_events             = false
				}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_telegram.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "token", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "room", "-1000000000000000"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "branches_to_be_notified", ""),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "confidential_note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "pipeline_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "wiki_page_events", "false"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_project_integration_telegram.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update the Telegram integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_telegram" "this" {
					project = %d
					token   = "923456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
					room    = "-2000000000000000"

					notify_only_broken_pipelines = false
					branches_to_be_notified      = "all"
					push_events                  = true
					issues_events                = true
					confidential_issues_events   = true
					merge_requests_events        = true
					tag_push_events              = true
					note_events                  = true
					confidential_note_events     = true
					pipeline_events              = true
					wiki_page_events             = true
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_telegram.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "project", strconv.FormatInt(testProject.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "token", "923456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "room", "-2000000000000000"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "confidential_issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "merge_requests_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "tag_push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "confidential_note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "pipeline_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_telegram.this", "wiki_page_events", "true"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_project_integration_telegram.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationTelegram_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationTelegramCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Telegram integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_telegram" "this" {
					project = "%s"
					token   = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
					room    = "-1000000000000000"

					notify_only_broken_pipelines = true
					push_events                  = false
					issues_events                = false
					confidential_issues_events   = false
					merge_requests_events        = false
					tag_push_events              = false
					note_events                  = false
					confidential_note_events     = false
					pipeline_events              = false
					wiki_page_events             = false
				}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_telegram.this", "id"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "token", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "room", "-1000000000000000"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "branches_to_be_notified", ""),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "confidential_note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "pipeline_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "wiki_page_events", "false"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_integration_telegram.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update the Telegram integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_telegram" "this" {
					project = %d
					token   = "923456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
					room    = "-2000000000000000"

					notify_only_broken_pipelines = false
					branches_to_be_notified      = "all"
					push_events                  = true
					issues_events                = true
					confidential_issues_events   = true
					merge_requests_events        = true
					tag_push_events              = true
					note_events                  = true
					confidential_note_events     = true
					pipeline_events              = true
					wiki_page_events             = true
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_telegram.this", "id"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "project", strconv.FormatInt(testProject.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "token", "923456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "room", "-2000000000000000"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "confidential_issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "merge_requests_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "tag_push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "confidential_note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "pipeline_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_telegram.this", "wiki_page_events", "true"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_integration_telegram.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationTelegram_missingRequired(t *testing.T) {
	testProject := testutil.CreateProject(t)

	requiredAttrs := map[string]string{
		"project":                    strconv.FormatInt(testProject.ID, 10),
		"token":                      `"123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"`,
		"room":                       `"-1000000000000000"`,
		"push_events":                "false",
		"issues_events":              "false",
		"confidential_issues_events": "false",
		"merge_requests_events":      "false",
		"tag_push_events":            "false",
		"note_events":                "false",
		"confidential_note_events":   "false",
		"pipeline_events":            "false",
		"wiki_page_events":           "false",
	}

	resourceWithout := func(missingAttr string) string {
		b := strings.Builder{}
		b.WriteString("resource \"gitlab_project_integration_telegram\" \"this\" {\n")
		for attr, value := range requiredAttrs {
			if attr != missingAttr {
				b.WriteString(attr)
				b.WriteString(" = ")
				b.WriteString(value)
				b.WriteString("\n")
			}
		}
		b.WriteString("}\n")

		return b.String()
	}

	steps := make([]resource.TestStep, 0, len(requiredAttrs))
	for attr := range requiredAttrs {
		steps = append(steps, resource.TestStep{
			Config:      resourceWithout(attr),
			ExpectError: regexp.MustCompile(`The argument "` + attr + `" is required`),
		})
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationTelegramCheckDestroy(testProject.ID),
		Steps:                    steps,
	})
}

func TestAcc_GitlabProjectIntegrationTelegram_invalidValues(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationTelegramCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Fail on invalid value of branches_to_be_notified
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_telegram" "this" {
					project                    = %d
					token                      = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
					room                       = "-1000000000000000"
					push_events                = true
					issues_events              = true
					confidential_issues_events = true
					merge_requests_events      = true
					tag_push_events            = true
					note_events                = true
					confidential_note_events   = true
					pipeline_events            = true
					wiki_page_events           = true

					branches_to_be_notified = "invalid"
				}`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute branches_to_be_notified value must be one of`),
			},
		},
	})
}

// TestAcc_GitlabProjectIntegrationTelegram_stateMove verifies that the moved block works
// when migrating from gitlab_integration_telegram to gitlab_project_integration_telegram.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAcc_GitlabProjectIntegrationTelegram_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccGitlabProjectIntegrationTelegramCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Telegram integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_telegram" "old" {
					project = "%s"
					token   = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
					room    = "-1000000000000000"

					notify_only_broken_pipelines = true
					push_events                  = false
					issues_events                = false
					confidential_issues_events   = false
					merge_requests_events        = false
					tag_push_events              = false
					note_events                  = false
					confidential_note_events     = false
					pipeline_events              = false
					wiki_page_events             = false
				}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_telegram.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_telegram" "new" {
					project = "%s"
					token   = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
					room    = "-1000000000000000"

					notify_only_broken_pipelines = true
					push_events                  = false
					issues_events                = false
					confidential_issues_events   = false
					merge_requests_events        = false
					tag_push_events              = false
					note_events                  = false
					confidential_note_events     = false
					pipeline_events              = false
					wiki_page_events             = false
				}

				moved {
					from = gitlab_integration_telegram.old
					to   = gitlab_project_integration_telegram.new
				}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_telegram.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:            "gitlab_project_integration_telegram.new",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func testAccGitlabProjectIntegrationTelegramCheckDestroy(projectId int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		service, _, err := testutil.TestGitlabClient.Services.GetTelegramService(projectId)
		if err != nil {
			return fmt.Errorf("Error calling API to get the Telegram integration: %w", err)
		}
		if service != nil && service.Active != false {
			return errors.New("Telegram integration still exists")
		}
		return nil
	}
}
