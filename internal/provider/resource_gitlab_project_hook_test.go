//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)


func TestAccGitlabProjectHook_basic(t *testing.T) {
	var hook gitlab.ProjectHook
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Create a project and hook with default options
			{
				Config: fmt.Sprintf(`resource "gitlab_project_hook" "foo" {
					project = "%d"
					url = "https://example.com/hook-%d"
					}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					testAccCheckGitlabProjectHookAttributes(&hook, &testAccGitlabProjectHookExpectedAttributes{
						URL:                   fmt.Sprintf("https://example.com/hook-%d", rInt),
						PushEvents:            true,
						EnableSSLVerification: true,
						BranchFilterStrategy:  "wildcard",
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_hook.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update the project hook to set Name and Description
			{
				Config: fmt.Sprintf(`resource "gitlab_project_hook" "foo" {
					project = "%d"
					url = "https://example.com/hook-%d"
					name = "Test"
					description = "Testing"
					}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					testAccCheckGitlabProjectHookAttributes(&hook, &testAccGitlabProjectHookExpectedAttributes{
						URL:                   fmt.Sprintf("https://example.com/hook-%d", rInt),
						Name:                  "Test",
						Description:           "Testing",
						PushEvents:            true,
						EnableSSLVerification: true,
						BranchFilterStrategy:  "wildcard",
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_hook.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update the project hook to toggle all the values to their inverse
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_hook" "foo" {
				  project = "%d"
				  url = "https://example.com/hook-%d"
				  enable_ssl_verification = false
				  push_events = true
				  push_events_branch_filter = "devel"
				  issues_events = false
				  confidential_issues_events = false
				  merge_requests_events = true
				  tag_push_events = true
				  note_events = true
				  confidential_note_events = true
				  job_events = true
				  emoji_events = true
				  pipeline_events = true
				  wiki_page_events = true
				  resource_access_token_events = true
				  deployment_events = true
				  releases_events = true
				  vulnerability_events = true
				  branch_filter_strategy = "wildcard"
				}
					`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					testAccCheckGitlabProjectHookAttributes(&hook, &testAccGitlabProjectHookExpectedAttributes{
						URL:                       fmt.Sprintf("https://example.com/hook-%d", rInt),
						Name:                      "",
						Description:               "",
						PushEvents:                true,
						PushEventsBranchFilter:    "devel",
						IssuesEvents:              false,
						ConfidentialIssuesEvents:  false,
						MergeRequestsEvents:       true,
						TagPushEvents:             true,
						NoteEvents:                true,
						ConfidentialNoteEvents:    true,
						JobEvents:                 true,
						EmojiEvents:               true,
						PipelineEvents:            true,
						WikiPageEvents:            true,
						DeploymentEvents:          true,
						ReleasesEvents:            true,
						VulnerabilityEvents:       true,
						ResourceAccessTokenEvents: true,
						EnableSSLVerification:     false,
						BranchFilterStrategy:      "wildcard",
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_hook.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update the project hook to toggle the options back
			{
				Config: fmt.Sprintf(`resource "gitlab_project_hook" "foo" {
					project = "%d"
					url = "https://example.com/hook-%d"
					}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					testAccCheckGitlabProjectHookAttributes(&hook, &testAccGitlabProjectHookExpectedAttributes{
						URL:                   fmt.Sprintf("https://example.com/hook-%d", rInt),
						PushEvents:            true,
						EnableSSLVerification: true,
						BranchFilterStrategy:  "wildcard",
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_hook.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

// Ensure the "custom_template" attribute works
func TestAccGitlabProjectHook_customTemplate(t *testing.T) {
	project := testutil.CreateProject(t)

	// Used for testing later
	var hook gitlab.ProjectHook
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Update the project hook to toggle all the values to their inverse
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_hook" "foo" {
					project = "%d"
					url = "https://example.com/hook-%d"
					enable_ssl_verification = false
					push_events = true
					push_events_branch_filter = "devel"
					issues_events = false
					confidential_issues_events = false
					merge_requests_events = true
					tag_push_events = true
					note_events = true
					confidential_note_events = true
					job_events = true
					pipeline_events = true
					wiki_page_events = true
					resource_access_token_events = true
					deployment_events = true
					releases_events = true
					custom_webhook_template = "{\"event\":\"{{object_kind}}\"}"
				  }`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					testAccCheckGitlabProjectHookAttributes(&hook, &testAccGitlabProjectHookExpectedAttributes{
						URL:                       fmt.Sprintf("https://example.com/hook-%d", rInt),
						PushEvents:                true,
						PushEventsBranchFilter:    "devel",
						IssuesEvents:              false,
						ConfidentialIssuesEvents:  false,
						MergeRequestsEvents:       true,
						TagPushEvents:             true,
						NoteEvents:                true,
						ConfidentialNoteEvents:    true,
						JobEvents:                 true,
						PipelineEvents:            true,
						WikiPageEvents:            true,
						ResourceAccessTokenEvents: true,
						DeploymentEvents:          true,
						ReleasesEvents:            true,
						EnableSSLVerification:     false,
						CustomWebhookTemplate:     "{\"event\":\"{{object_kind}}\"}",
						BranchFilterStrategy:      "wildcard",
					}),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_hook.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabProjectHook_customHeaders(t *testing.T) {
	testutil.SkipIfCE(t)
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Create a Project Hook with custom headers
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_hook" "this" {
						project = "%s"
						url   = "http://example.com"
						token = "supersecret"

						custom_headers = [
							{
								key = "test"
								value = "testValue"
							},
							{
								key = "test2"
								value = "testValue2"
							}
						]
						
					}
				`, project.PathWithNamespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_hook.this", "custom_headers.#", "2"),
					resource.TestCheckResourceAttr("gitlab_project_hook.this", "custom_headers.0.key", "test"),
					resource.TestCheckResourceAttr("gitlab_project_hook.this", "custom_headers.0.value", "testValue"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_hook.this",
				ImportState:       true,
				ImportStateVerify: true,
				// Values don't come back on "read", so they can't be imported
				ImportStateVerifyIgnore: []string{"token", "custom_headers.0.value", "custom_headers.1.value"},
			},
			// Create a Project Hook with custom headers
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_hook" "this" {
						project = "%s"
						url   = "http://example.com"
						token = "supersecret"

						custom_headers = [
							{
								key = "test"
								value = "newValue"
							},
							{
								key = "test2"
								value = "newValue2"
							}
						]
						
					}
				`, project.PathWithNamespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_hook.this", "custom_headers.#", "2"),
					resource.TestCheckResourceAttr("gitlab_project_hook.this", "custom_headers.0.key", "test"),
					resource.TestCheckResourceAttr("gitlab_project_hook.this", "custom_headers.0.value", "newValue"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_hook.this",
				ImportState:       true,
				ImportStateVerify: true,
				// Values don't come back on "read", so they can't be imported
				ImportStateVerifyIgnore: []string{"token", "custom_headers.0.value", "custom_headers.1.value"},
			},
		},
	})
}

// Test that when updating the `project` attribute, the
// hook is associated to the new project properly
func TestAccGitlabProjectHook_updateProject(t *testing.T) {
	var hook gitlab.ProjectHook
	projectOne := testutil.CreateProject(t)
	projectTwo := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Create a project and hook with default options
			{
				Config: fmt.Sprintf(`resource "gitlab_project_hook" "foo" {
					project = "%d"
					url = "https://example.com/hook-1234"
				  }`, projectOne.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					resource.TestCheckResourceAttr(
						"gitlab_project_hook.foo", "project_id", fmt.Sprintf("%d", projectOne.ID),
					),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_hook.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Create a project and hook using the second project, validate that the project_id changes
			{
				Config: fmt.Sprintf(`resource "gitlab_project_hook" "foo" {
					project = "%d"
					url = "https://example.com/hook-5678"
				  }`, projectTwo.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
					resource.TestCheckResourceAttr(
						"gitlab_project_hook.foo", "project_id", fmt.Sprintf("%d", projectTwo.ID),
					),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_hook.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabProjectHook_migrateFromSDKToFramework(t *testing.T) {
	var hook gitlab.ProjectHook
	projectOne := testutil.CreateProject(t)

	// Create common config for testing
	config := fmt.Sprintf(`resource "gitlab_project_hook" "foo" {
		project = "%d"
		url = "https://example.com/hook-1234"
	  }`, projectOne.ID)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Create the pipeline in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.4",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
				Check:  testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config:                   config,
				Check:                    testAccCheckGitlabProjectHookExists("gitlab_project_hook.foo", &hook),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_project_hook.foo",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore:  []string{"token"},
			},
		},
	})
}

func TestAccGitlabProjectHook_validations(t *testing.T) {
	testutil.SkipIfCE(t)
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Validate that URLs may not contain whitepaces
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`resource "gitlab_project_hook" "foo" {
					project = "%d"
					url = "https://example.com/hook-1234     " // spaces in the URL are invalid
				  }`, project.ID),
				ExpectError: regexp.MustCompile("The URL may not contain whitespace"),
			},
		},
	})
}

func testAccCheckGitlabProjectHookExists(n string, hook *gitlab.ProjectHook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project, hookID, err := (&gitlabProjectHookResourceModel{}).ResourceGitlabProjectHookParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		gotHook, _, err := testutil.TestGitlabClient.Projects.GetProjectHook(project, hookID)
		if err != nil {
			return err
		}
		*hook = *gotHook
		return nil
	}
}

func TestResourceGitlabProjectHook_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	givenV0State := &gitlabProjectHookResourceModel{
		Project: types.StringValue("foo/bar"),
		HookID:  types.Int64Value(42),
		ID:      types.StringValue("42"),
	}
	expectedV1State := &gitlabProjectHookResourceModel{
		Project: types.StringValue("foo/bar"),
		HookID:  types.Int64Value(42),
		ID:      types.StringValue("foo/bar:42"),
	}

	// Execute the migration
	givenV0State.v0StateUpgrade(context.Background())

	if !reflect.DeepEqual(expectedV1State, givenV0State) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expectedV1State, givenV0State)
	}
}

type testAccGitlabProjectHookExpectedAttributes struct {
	URL                       string
	Name                      string
	Description               string
	PushEvents                bool
	PushEventsBranchFilter    string
	IssuesEvents              bool
	ConfidentialIssuesEvents  bool
	MergeRequestsEvents       bool
	TagPushEvents             bool
	NoteEvents                bool
	ConfidentialNoteEvents    bool
	JobEvents                 bool
	EmojiEvents               bool
	PipelineEvents            bool
	WikiPageEvents            bool
	ResourceAccessTokenEvents bool
	DeploymentEvents          bool
	ReleasesEvents            bool
	VulnerabilityEvents       bool
	EnableSSLVerification     bool
	CustomWebhookTemplate     string
	BranchFilterStrategy      string
}

func testAccCheckGitlabProjectHookAttributes(hook *gitlab.ProjectHook, want *testAccGitlabProjectHookExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if hook.URL != want.URL {
			return fmt.Errorf("got url %q; want %q", hook.URL, want.URL)
		}

		if hook.Name != want.Name {
			return fmt.Errorf("got name %q; want %q", hook.Name, want.Name)
		}

		if hook.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", hook.Description, want.Description)
		}

		if hook.EnableSSLVerification != want.EnableSSLVerification {
			return fmt.Errorf("got enable_ssl_verification %t; want %t", hook.EnableSSLVerification, want.EnableSSLVerification)
		}

		if hook.PushEvents != want.PushEvents {
			return fmt.Errorf("got push_events %t; want %t", hook.PushEvents, want.PushEvents)
		}

		if hook.PushEventsBranchFilter != want.PushEventsBranchFilter {
			return fmt.Errorf("got push_events_branch_filter %q; want %q", hook.PushEventsBranchFilter, want.PushEventsBranchFilter)
		}

		if hook.IssuesEvents != want.IssuesEvents {
			return fmt.Errorf("got issues_events %t; want %t", hook.IssuesEvents, want.IssuesEvents)
		}

		if hook.ConfidentialIssuesEvents != want.ConfidentialIssuesEvents {
			return fmt.Errorf("got confidential_issues_events %t; want %t", hook.ConfidentialIssuesEvents, want.ConfidentialIssuesEvents)
		}

		if hook.MergeRequestsEvents != want.MergeRequestsEvents {
			return fmt.Errorf("got merge_requests_events %t; want %t", hook.MergeRequestsEvents, want.MergeRequestsEvents)
		}

		if hook.TagPushEvents != want.TagPushEvents {
			return fmt.Errorf("got tag_push_events %t; want %t", hook.TagPushEvents, want.TagPushEvents)
		}

		if hook.NoteEvents != want.NoteEvents {
			return fmt.Errorf("got note_events %t; want %t", hook.NoteEvents, want.NoteEvents)
		}

		if hook.ConfidentialNoteEvents != want.ConfidentialNoteEvents {
			return fmt.Errorf("got confidential_note_events %t; want %t", hook.ConfidentialNoteEvents, want.ConfidentialNoteEvents)
		}

		if hook.JobEvents != want.JobEvents {
			return fmt.Errorf("got job_events %t; want %t", hook.JobEvents, want.JobEvents)
		}

		if hook.EmojiEvents != want.EmojiEvents {
			return fmt.Errorf("got emoji_events %t; want %t", hook.EmojiEvents, want.EmojiEvents)
		}

		if hook.PipelineEvents != want.PipelineEvents {
			return fmt.Errorf("got pipeline_events %t; want %t", hook.PipelineEvents, want.PipelineEvents)
		}

		if hook.WikiPageEvents != want.WikiPageEvents {
			return fmt.Errorf("got wiki_page_events %t; want %t", hook.WikiPageEvents, want.WikiPageEvents)
		}

		if hook.ResourceAccessTokenEvents != want.ResourceAccessTokenEvents {
			return fmt.Errorf("got resource_access_token_events %t; want %t", hook.ResourceAccessTokenEvents, want.ResourceAccessTokenEvents)
		}

		if hook.DeploymentEvents != want.DeploymentEvents {
			return fmt.Errorf("got deployment_events %t; want %t", hook.DeploymentEvents, want.DeploymentEvents)
		}

		if hook.ReleasesEvents != want.ReleasesEvents {
			return fmt.Errorf("got releases_events %t; want %t", hook.ReleasesEvents, want.ReleasesEvents)
		}

		if hook.VulnerabilityEvents != want.VulnerabilityEvents {
			return fmt.Errorf("got vulnerability_events %t; want %t", hook.VulnerabilityEvents, want.VulnerabilityEvents)
		}

		if hook.CustomWebhookTemplate != want.CustomWebhookTemplate {
			return fmt.Errorf("got custom_webhook_template %q; want %q", hook.CustomWebhookTemplate, want.CustomWebhookTemplate)
		}

		if hook.BranchFilterStrategy != want.BranchFilterStrategy {
			return fmt.Errorf("got branch_filter_strategy %q; want %q", hook.BranchFilterStrategy, want.BranchFilterStrategy)
		}

		return nil
	}
}

func testAccCheckGitlabProjectHookDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_hook" {
			continue
		}

		project, hookID, err := (&gitlabProjectHookResourceModel{}).ResourceGitlabProjectHookParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.Projects.GetProjectHook(project, hookID)
		if err == nil {
			return fmt.Errorf("Project Hook %d in project %s still exists", hookID, project)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
