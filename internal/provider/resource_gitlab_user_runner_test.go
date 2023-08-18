//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabUserRunner_basicInstanceRunner(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             userRunnerCheckDestroy,
		Steps: []resource.TestStep{
			{
				// test that validation of instance + group_id doesn't work
				Config: `
					 resource "gitlab_user_runner" "this" {
						runner_type = "instance_type"
						group_id    = 12345
					 }
					`,
				ExpectError: regexp.MustCompile(`Runner Type was set to "instance_type", but a project or group ID was provided`),
			},
			{
				// Create a valid runner with the base set of attributes
				Config: `
					 resource "gitlab_user_runner" "this" {
						runner_type = "instance_type"
					 }
					`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_user_runner.this", "token"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
			{
				// Update the runner with a bunch of optional attributes
				Config: `
					 resource "gitlab_user_runner" "this" {
						runner_type = "instance_type"

						description = "Eat fish on floor meowwww yet climb leg"
						paused = true
						locked = true
						untagged = false
						tag_list = ["stuff", "things"]
						access_level = "not_protected"
						maximum_timeout = 600
					 }
					`,
				Check: resource.TestCheckResourceAttrSet("gitlab_user_runner.this", "token"), // rest of attributes checked by import
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
		},
	})
}

func TestAcc_GitlabUserRunner_basicProjectRunner(t *testing.T) {

	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             userRunnerCheckDestroy,
		Steps: []resource.TestStep{
			{
				// test that validation of project and no ID works
				Config: `
					 resource "gitlab_user_runner" "this" {
						runner_type = "project_type"
					 }
					`,
				ExpectError: regexp.MustCompile(`Project ID not provided when Runner Type is set to "project_id".`),
			},
			{
				// test that validation of project and no ID works
				Config: `
					 resource "gitlab_user_runner" "this" {
						runner_type = "project_type"
						project_id = 1234
						group_id = 1234
					 }
					`,
				ExpectError: regexp.MustCompile(`When creating a Project Runner, "group_id" must not be provided.`),
			},
			{
				// Create a valid runner with the base set of attributes
				Config: fmt.Sprintf(`
					 resource "gitlab_user_runner" "this" {
						runner_type = "project_type"
						project_id  = "%d"
					 }
					`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_user_runner.this", "token"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
			{
				// Update the runner with a bunch of optional attributes
				Config: fmt.Sprintf(`
					 resource "gitlab_user_runner" "this" {
						runner_type = "project_type"
						project_id = "%d"

						description = "Eat fish on floor meowwww yet climb leg"
						paused = true
						locked = true
						untagged = false
						tag_list = ["stuff", "things"]
						access_level = "not_protected"
						maximum_timeout = 600
					 }
					`, project.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_user_runner.this", "token"), // rest of attributes checked by import
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
		},
	})
}

func TestAcc_GitlabUserRunner_basicGroupRunner(t *testing.T) {

	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             userRunnerCheckDestroy,
		Steps: []resource.TestStep{
			{
				// test that validation of group and no ID works
				Config: `
					 resource "gitlab_user_runner" "this" {
						runner_type = "group_type"
					 }
					`,
				ExpectError: regexp.MustCompile(`Group ID not provided when Runner Type is set to "group_type".`),
			},
			{
				// test that validation of group and no ID works
				Config: `
					 resource "gitlab_user_runner" "this" {
						runner_type = "group_type"
						group_id = 1234
						project_id = 1234
					 }
					`,
				ExpectError: regexp.MustCompile(`When creating a Group Runner, "project_id" must not be provided.`),
			},
			{
				// Create a valid runner with the base set of attributes
				Config: fmt.Sprintf(`
					 resource "gitlab_user_runner" "this" {
						runner_type = "group_type"
						group_id  = "%d"
					 }
					`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_user_runner.this", "token"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
			{
				// Update the runner with a bunch of optional attributes
				Config: fmt.Sprintf(`
					 resource "gitlab_user_runner" "this" {
						runner_type = "group_type"
						group_id = "%d"

						description = "Eat fish on floor meowwww yet climb leg"
						paused = true
						locked = true
						untagged = false
						tag_list = ["stuff", "things"]
						access_level = "not_protected"
						maximum_timeout = 600
					 }
					`, group.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_user_runner.this", "token"), // rest of attributes checked by import
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
		},
	})
}

func TestAcc_GitlabUserRunner_createWithOptions(t *testing.T) {

	group := testutil.CreateGroups(t, 1)[0]
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             userRunnerCheckDestroy,
		Steps: []resource.TestStep{
			{
				// test that validation of group and no ID works
				Config: `
					 resource "gitlab_user_runner" "instance_runner" {
						runner_type = "instance_type"

						description = "Eat fish on floor meowwww yet climb leg"
						paused = true
						locked = true
						untagged = false
						tag_list = ["stuff", "things"]
						access_level = "not_protected"
						maximum_timeout = 600
					 }
					`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_user_runner.instance_runner", "token"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.instance_runner",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
			{
				// test that validation of group and no ID works
				Config: fmt.Sprintf(`
					 resource "gitlab_user_runner" "group_runner" {
						runner_type = "group_type"
						group_id = "%d"

						description = "Eat fish on floor meowwww yet climb leg"
						paused = true
						locked = true
						untagged = false
						tag_list = ["stuff", "things"]
						access_level = "not_protected"
						maximum_timeout = 600
					 }
					`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_user_runner.group_runner", "token"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.group_runner",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
			{
				// test that validation of group and no ID works
				Config: fmt.Sprintf(`
					 resource "gitlab_user_runner" "project_runner" {
						runner_type = "project_type"
						project_id = "%d"

						description = "Eat fish on floor meowwww yet climb leg"
						paused = true
						locked = true
						untagged = false
						tag_list = ["stuff", "things"]
						access_level = "not_protected"
						maximum_timeout = 600
					 }
					`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_user_runner.project_runner", "token"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_user_runner.project_runner",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"}, // doesn't import
			},
		},
	})
}

func userRunnerCheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_user_runner" {

			// Attempt to retrieve the runner by its ID
			runnerId := rs.Primary.Attributes["id"]
			details, _, err := testutil.TestGitlabClient.Runners.GetRunnerDetails(runnerId)

			// An error shouold be received. If we don't get one, it found a runner
			// and we need to error.
			if err == nil {
				return fmt.Errorf("Runner still found with details %v , not properly destroyed", details)
			}

			// If the runner is destroyed, this should receive a 404. If it's not 404, we throw an error
			if !api.Is404(err) {
				return fmt.Errorf("Unknown server error when attempting to retrieve the runner: %v", err)
			}

			// Stop looping because we found the resource.
			break
		}
	}
	return nil
}
