//go:build acceptance

package sdk

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabRunner_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	//The runner token is not populated on the return from the group create, so re-retrieve it to get the token.
	group, _, err := testutil.TestGitlabClient.Groups.GetGroup(group.ID, &gitlab.GetGroupOptions{})
	if err != nil {
		t.Fail()
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckRunnerDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum"
				}
				`, group.RunnersToken),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_runner.this", "registration_token"),
					resource.TestCheckResourceAttrSet("gitlab_runner.this", "authentication_token"),
				),
			},
			{
				ResourceName:      "gitlab_runner.this",
				ImportState:       true,
				ImportStateVerify: true,
				//These need to be ignored since they don't come back in the "get" command
				ImportStateVerifyIgnore: []string{"authentication_token", "registration_token"},
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum Dolor Sit Amet"
				}
				`, group.RunnersToken),
			},
			{
				ResourceName:      "gitlab_runner.this",
				ImportState:       true,
				ImportStateVerify: true,
				//These need to be ignored since they don't come back in the "get" command
				ImportStateVerifyIgnore: []string{"authentication_token", "registration_token"},
			},
		},
	})
}

func TestAccGitlabRunner_instance(t *testing.T) {
	// This pulls from the gitlab.rb file, and is set on instance start-up
	token := "ACCTEST1234567890123_RUNNER_REG_TOKEN"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckRunnerDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum"
				}
				`, token),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_runner.this", "registration_token"),
					resource.TestCheckResourceAttrSet("gitlab_runner.this", "authentication_token"),
				),
			},
			{
				ResourceName:      "gitlab_runner.this",
				ImportState:       true,
				ImportStateVerify: true,
				//These need to be ignored since they don't come back in the "get" command
				ImportStateVerifyIgnore: []string{"authentication_token", "registration_token"},
			},
		},
	})
}

func TestAccGitlabRunner_comprehensive(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	//The runner token is not populated on the return from the group create, so re-retrieve it to get the token.
	group, _, err := testutil.TestGitlabClient.Groups.GetGroup(group.ID, &gitlab.GetGroupOptions{})
	if err != nil {
		t.Fail()
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckRunnerDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum"

					paused = false
					locked = false
					run_untagged = false
					tag_list = ["tag_one", "tag_two"]
					access_level = "ref_protected"
					maximum_timeout = 3600
					maintenance_note = "Leave me alone!"
				}
				`, group.RunnersToken),
			},
			{
				ResourceName:      "gitlab_runner.this",
				ImportState:       true,
				ImportStateVerify: true,
				//These need to be ignored since they don't come back in the "get" command
				ImportStateVerifyIgnore: []string{"authentication_token", "registration_token"},
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum Dolor Sit Amet"

					paused = true
					locked = true
					run_untagged = true
					tag_list = ["tag_one", "tag_two", "tag_three"]
					access_level = "not_protected"
					maximum_timeout = 4200
					maintenance_note = "Leave me!"
				}
				`, group.RunnersToken),
			},
			{
				ResourceName:      "gitlab_runner.this",
				ImportState:       true,
				ImportStateVerify: true,
				//These need to be ignored since they don't come back in the "get" command
				ImportStateVerifyIgnore: []string{"authentication_token", "registration_token"},
			},
		},
	})
}

func TestAccGitlabRunner_sorting_tags(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	//The runner token is not populated on the return from the group create, so re-retrieve it to get the token.
	group, _, err := testutil.TestGitlabClient.Groups.GetGroup(group.ID, &gitlab.GetGroupOptions{})
	if err != nil {
		t.Fatalf("Failed to retrieve group %d: %s", group.ID, err)
	}

	resource.ParallelTest(t, resource.TestCase{

		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckRunnerDestroy,

		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum"

					tag_list = ["a", "b", "c"]
				}
				`, group.RunnersToken),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum"

					tag_list = ["c", "b", "a"]
				}
				`, group.RunnersToken),
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum"

					tag_list = ["b", "c", "a"]
				}
				`, group.RunnersToken),
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_runner" "this" {
					registration_token = "%s"
					description = "Lorem Ipsum"

					tag_list = ["a", "c", "b"]
				}
				`, group.RunnersToken),
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
		},
	})
}

func testAccCheckRunnerDestroy(state *terraform.State) error {
	for _, rs := range state.RootModule().Resources {
		if rs.Type != "gitlab_runner" {
			continue
		}

		id, _ := strconv.Atoi(rs.Primary.ID)

		runner, _, err := testutil.TestGitlabClient.Runners.GetRunnerDetails(id)
		if err == nil {
			if runner != nil {
				return fmt.Errorf("runner still exists")
			}
		}

		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
