//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAcc_GitlabValueStreamAnalytics_ProjectDefaultStages(t *testing.T) {
	testutil.SkipIfCE(t)

	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectValueStreamAnalytics_CheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						project_full_path = "%s"
						stages = [
							{
								name = "Issue"
								custom = false
								hidden = true
							},
							{
								name = "Plan"
								custom = false
								hidden = false
							}
						]
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "project_full_path", testProject.PathWithNamespace),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":   "Issue",
						"custom": "false",
						"hidden": "true",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":   "Plan",
						"custom": "false",
						"hidden": "false",
					}),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestProjectValueStreamAnalytics_ProjectCustom(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)
	testLabels := testutil.CreateProjectLabels(t, testProject.PathWithNamespace, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectValueStreamAnalytics_CheckDestroy,
		Steps: []resource.TestStep{
			// Create Value Stream with 2 custom stages
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						project_full_path = "%s"
						stages = [
							{
								name = "Issue Total"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								end_event_identifier = "ISSUE_CLOSED"
							},
							{
								name = "Issue Labels"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_LABEL_ADDED"
								start_event_label_id = "gid://gitlab/ProjectLabel/%d"
								end_event_identifier = "ISSUE_LABEL_REMOVED"
								end_event_label_id = "gid://gitlab/ProjectLabel/%d"
							}
						]
					}
				`, testProject.PathWithNamespace, testLabels[0].ID, testLabels[1].ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "project_full_path", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Issue Total",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "ISSUE_CREATED",
						"end_event_identifier":   "ISSUE_CLOSED",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Issue Labels",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "ISSUE_LABEL_ADDED",
						"end_event_identifier":   "ISSUE_LABEL_REMOVED",
						"start_event_label_id":   fmt.Sprintf("gid://gitlab/ProjectLabel/%d", testLabels[0].ID),
						"end_event_label_id":     fmt.Sprintf("gid://gitlab/ProjectLabel/%d", testLabels[1].ID),
					}),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update Value Stream with 1 custom stage
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						project_full_path = "%s"
						stages = [
							{
								name = "Merge Request Total"
								custom = true
								hidden = false
								start_event_identifier = "MERGE_REQUEST_CREATED"
								end_event_identifier = "MERGE_REQUEST_CLOSED"
							}
						]
					}
					`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "project_full_path", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Merge Request Total",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "MERGE_REQUEST_CREATED",
						"end_event_identifier":   "MERGE_REQUEST_CLOSED",
					}),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestProjectValueStreamAnalytics_EnsureErrorOnInvalidLabelEvent(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)
	testLabels := testutil.CreateProjectLabels(t, testProject.PathWithNamespace, 2)
	err_regex, err := regexp.Compile("Error: (Missing|Unexpected) Attribute")
	if err != nil {
		t.Errorf("Unable to format expected label error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						project_full_path = "%s"
						stages = [
							{
								name = "No start label when required"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_LABEL_ADDED"
								end_event_identifier = "ISSUE_CLOSED"
							}
						]
					}
				`, testProject.PathWithNamespace),
				ExpectError: err_regex,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						project_full_path = "%s"
						stages = [
							{
								name = "No end label when required"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								end_event_identifier = "ISSUE_LABEL_REMOVED"
							}
						]
					}
					`, testProject.PathWithNamespace),
				ExpectError: err_regex,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						project_full_path = "%s"
						stages = [
							{
								name = "Start label when not required"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								start_event_label_id = "gid://gitlab/GroupLabel/%d"
								end_event_identifier = "ISSUE_CLOSED"
							}
						]
					}
					`, testProject.PathWithNamespace, testLabels[0].ID),
				ExpectError: err_regex,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						project_full_path = "%s"
						stages = [
							{
								name = "End label when not required"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								end_event_identifier = "ISSUE_CLOSED"
								end_event_label_id = "gid://gitlab/GroupLabel/%d"
							}
						]
					}
					`, testProject.PathWithNamespace, testLabels[1].ID),
				ExpectError: err_regex,
			},
		},
	})
}

func TestAcc_GitlabValueStreamAnalytics_GroupDefaultStages(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupValueStreamAnalytics_CheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						group_full_path = "%s"
						stages = [
							{
								name = "Issue"
								custom = false
								hidden = true
							},
							{
								name = "Plan"
								custom = false
								hidden = false
							}
						]
					}
				`, testGroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "group_full_path", testGroup.FullPath),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":   "Issue",
						"custom": "false",
						"hidden": "true",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":   "Plan",
						"custom": "false",
						"hidden": "false",
					}),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabValueStreamAnalytics_GroupCustomStages(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testLabels := testutil.CreateGroupLabels(t, testGroup.ID, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupValueStreamAnalytics_CheckDestroy,
		Steps: []resource.TestStep{
			// Create Value Stream with 2 custom stages
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						group_full_path = "%s"
						stages = [
							{
								name = "Issue Total"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								end_event_identifier = "ISSUE_CLOSED"
							},
							{
								name = "Issue Labels"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_LABEL_ADDED"
								start_event_label_id = "gid://gitlab/GroupLabel/%d"
								end_event_identifier = "ISSUE_LABEL_REMOVED"
								end_event_label_id = "gid://gitlab/GroupLabel/%d"
							}
						]
					}
				`, testGroup.FullPath, testLabels[0].ID, testLabels[1].ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "group_full_path", testGroup.FullPath),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Issue Total",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "ISSUE_CREATED",
						"end_event_identifier":   "ISSUE_CLOSED",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Issue Labels",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "ISSUE_LABEL_ADDED",
						"end_event_identifier":   "ISSUE_LABEL_REMOVED",
						"start_event_label_id":   fmt.Sprintf("gid://gitlab/GroupLabel/%d", testLabels[0].ID),
						"end_event_label_id":     fmt.Sprintf("gid://gitlab/GroupLabel/%d", testLabels[1].ID),
					}),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update Value Stream with 1 custom stage
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						group_full_path = "%s"
						stages = [
							{
								name = "Merge Request Total"
								custom = true
								hidden = false
								start_event_identifier = "MERGE_REQUEST_CREATED"
								end_event_identifier = "MERGE_REQUEST_CLOSED"
							}
						]
					}
					`, testGroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "group_full_path", testGroup.FullPath),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Merge Request Total",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "MERGE_REQUEST_CREATED",
						"end_event_identifier":   "MERGE_REQUEST_CLOSED",
					}),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabValueStreamAnalytics_StandingUpdate(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testLabels := testutil.CreateGroupLabels(t, testGroup.ID, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupValueStreamAnalytics_CheckDestroy,
		Steps: []resource.TestStep{
			// Create Value Stream with 2 custom stages
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						group_full_path = "%s"
						stages = [
							{
								name = "Issue Total"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								end_event_identifier = "ISSUE_CLOSED"
							},
							{
								name = "Issue Labels"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_LABEL_ADDED"
								start_event_label_id = "gid://gitlab/GroupLabel/%d"
								end_event_identifier = "ISSUE_LABEL_REMOVED"
								end_event_label_id = "gid://gitlab/GroupLabel/%d"
							}
						]
					}
				`, testGroup.FullPath, testLabels[0].ID, testLabels[0].ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "group_full_path", testGroup.FullPath),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Issue Total",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "ISSUE_CREATED",
						"end_event_identifier":   "ISSUE_CLOSED",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Issue Labels",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "ISSUE_LABEL_ADDED",
						"end_event_identifier":   "ISSUE_LABEL_REMOVED",
						"start_event_label_id":   fmt.Sprintf("gid://gitlab/GroupLabel/%d", testLabels[0].ID),
						"end_event_label_id":     fmt.Sprintf("gid://gitlab/GroupLabel/%d", testLabels[0].ID),
					}),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update Value Stream with 1 custom stage
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						group_full_path = "%s"
						stages = [
							{
								name = "Issue Total"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								end_event_identifier = "ISSUE_CLOSED"
							},
							{
								name = "Issue Labels"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_LABEL_ADDED"
								start_event_label_id = "gid://gitlab/GroupLabel/%d"
								end_event_identifier = "ISSUE_LABEL_REMOVED"
								end_event_label_id = "gid://gitlab/GroupLabel/%d"
							}
						]
					}
					`, testGroup.FullPath, testLabels[1].ID, testLabels[1].ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "group_full_path", testGroup.FullPath),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Issue Total",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "ISSUE_CREATED",
						"end_event_identifier":   "ISSUE_CLOSED",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("gitlab_value_stream_analytics.foo", "stages.*", map[string]string{
						"name":                   "Issue Labels",
						"custom":                 "true",
						"hidden":                 "false",
						"start_event_identifier": "ISSUE_LABEL_ADDED",
						"end_event_identifier":   "ISSUE_LABEL_REMOVED",
						"start_event_label_id":   fmt.Sprintf("gid://gitlab/GroupLabel/%d", testLabels[1].ID),
						"end_event_label_id":     fmt.Sprintf("gid://gitlab/GroupLabel/%d", testLabels[1].ID),
					}),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabValueStreamAnalytics_OrderRetention(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testLabels := testutil.CreateGroupLabels(t, testGroup.ID, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupValueStreamAnalytics_CheckDestroy,
		Steps: []resource.TestStep{
			// Create Value Stream with 2 custom stages
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						group_full_path = "%s"
						stages = [
							{
								name = "First Stage"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								end_event_identifier = "ISSUE_CLOSED"
							},
							{
								name = "Second Stage"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_LABEL_ADDED"
								start_event_label_id = "gid://gitlab/GroupLabel/%d"
								end_event_identifier = "ISSUE_LABEL_REMOVED"
								end_event_label_id = "gid://gitlab/GroupLabel/%d"
							}
						]
					}
				`, testGroup.FullPath, testLabels[0].ID, testLabels[0].ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "group_full_path", testGroup.FullPath),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "2"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.0.name", "First Stage"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.1.name", "Second Stage"),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_value_stream_analytics" "foo" {
						name = "test"
						group_full_path = "%s"
						stages = [
							{
								name = "Second Stage"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_CREATED"
								end_event_identifier = "ISSUE_CLOSED"
							},
							{
								name = "First Stage"
								custom = true
								hidden = false
								start_event_identifier = "ISSUE_LABEL_ADDED"
								start_event_label_id = "gid://gitlab/GroupLabel/%d"
								end_event_identifier = "ISSUE_LABEL_REMOVED"
								end_event_label_id = "gid://gitlab/GroupLabel/%d"
							}
						]
					}
				`, testGroup.FullPath, testLabels[0].ID, testLabels[0].ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_value_stream_analytics.foo", "id"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "name", "test"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "group_full_path", testGroup.FullPath),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.#", "2"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.0.name", "Second Stage"),
					resource.TestCheckResourceAttr("gitlab_value_stream_analytics.foo", "stages.1.name", "First Stage"),
				),
			},
			{
				ResourceName:      "gitlab_value_stream_analytics.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAcc_GitlabProjectValueStreamAnalytics_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_value_stream_analytics" {
			fullPath, valueStreamID, err := utils.ParseTwoPartID(rs.Primary.ID)
			if err != nil {
				return fmt.Errorf("Failed to parse value stream id %q: %w", rs.Primary.ID, err)
			}

			query := gitlab.GraphQLQuery{
				Query: fmt.Sprintf(`
						query {
							project(fullPath: "%s") {
								fullPath,
								valueStreams(id: "%s") {
									stages {
										name
									}
								}
							}
						}`, fullPath, valueStreamID),
			}

			var response projectValueStreamResponse
			if _, err := testutil.TestGitlabClient.GraphQL.Do(query, &response); err != nil {
				return err
			}

			// value stream analytics still exists if nodes is not empty
			if len(response.Data.Project.ValueStreams.Nodes) > 0 {
				return fmt.Errorf("Value Stream: %s in project: %s still exists", valueStreamID, fullPath)
			}

			return nil
		}
	}
	return nil
}

func testAcc_GitlabGroupValueStreamAnalytics_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_value_stream_analytics" {
			fullPath, valueStreamID, err := utils.ParseTwoPartID(rs.Primary.ID)
			if err != nil {
				return fmt.Errorf("Failed to parse value stream id %q: %w", rs.Primary.ID, err)
			}

			query := gitlab.GraphQLQuery{
				Query: fmt.Sprintf(`
						query {
							group(fullPath: "%s") {
								fullPath,
								valueStreams(id: "%s") {
									stages {
										name
									}
								}
							}
						}`, fullPath, valueStreamID),
			}

			var response groupValueStreamResponse
			if _, err := testutil.TestGitlabClient.GraphQL.Do(query, &response); err != nil {
				return err
			}

			// value stream analytics still exists if nodes is not empty
			if len(response.Data.Group.ValueStreams.Nodes) > 0 {
				return fmt.Errorf("Value Stream: %s in group: %s still exists", valueStreamID, fullPath)
			}

			return nil
		}
	}
	return nil
}
