//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabProjectMergeRequestNote_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testBranch := testutil.CreateBranches(t, testProject, 1)[0]
	testMergeRequest := testutil.CreateMergeRequest(t, testutil.CreateUsers(t, 1)[0], testProject, testBranch.Name, "main")
	testNoteBody := "A random note"
	testNoteUpdatedBody := "A random updated note"
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMergeRequestNoteDestroy,
		Steps: []resource.TestStep{
			// Create a basic note.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_merge_request_note" "test" {
					project           = %d
					merge_request_iid = %d
					body              = "%s"
				}`, testProject.ID, testMergeRequest.IID, testNoteBody),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "merge_request_iid", fmt.Sprintf("%d", testMergeRequest.IID)),
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "body", testNoteBody),
				),
			},
			{
				ResourceName:      "gitlab_project_merge_request_note.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Do a standard update of just body
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_merge_request_note" "test" {
					project           = %d
					merge_request_iid = %d
					body              = "%s"
				}`, testProject.ID, testMergeRequest.IID, testNoteUpdatedBody),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "body", testNoteUpdatedBody),
				),
			},
			{
				ResourceName:      "gitlab_project_merge_request_note.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Do a force recreate update using internal
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_merge_request_note" "test" {
					project           = %d
					merge_request_iid = %d
					body              = "%s"
					internal          = true
				}`, testProject.ID, testMergeRequest.IID, testNoteUpdatedBody),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "body", testNoteUpdatedBody),
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "internal", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project_merge_request_note.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Do a force recreate update using created_at in the past
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_merge_request_note" "test" {
					project           = %d
					merge_request_iid = %d
					body              = "%s"
					internal          = true
					created_at        = "2001-01-01T08:00:00Z"
				}`, testProject.ID, testMergeRequest.IID, testNoteUpdatedBody),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "body", testNoteUpdatedBody),
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "internal", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project_merge_request_note.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Do a force recreate update using merge_request_diff_head_sha
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_merge_request_note" "test" {
					project                     = %d
					merge_request_iid           = %d
					body                        = "%s"
					internal                    = true
					created_at                  = "2001-01-01T08:00:00Z"
					merge_request_diff_head_sha = "%s"
				}`, testProject.ID, testMergeRequest.IID, testNoteUpdatedBody, testBranch.Commit.ShortID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "body", testNoteUpdatedBody),
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "internal", "true"),
					resource.TestCheckResourceAttr("gitlab_project_merge_request_note.test", "merge_request_diff_head_sha", testBranch.Commit.ShortID),
				),
			},
			{
				ResourceName:      "gitlab_project_merge_request_note.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Not available on input as is config only and can't be retrieved by API
				ImportStateVerifyIgnore: []string{"merge_request_diff_head_sha"},
			},
		},
	})
}

func TestAcc_GitlabProjectMergeRequestNote_attributeValidation(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testBranch := testutil.CreateBranches(t, testProject, 1)[0]
	testMergeRequest := testutil.CreateMergeRequest(t, testutil.CreateUsers(t, 1)[0], testProject, testBranch.Name, "main")
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMergeRequestNoteDestroy,
		Steps: []resource.TestStep{
			// Create a basic note.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_merge_request_note" "test" {
					project           = %d
					merge_request_iid = %d
					body              = "A string"
					created_at        = "2099-01-01"
				}`, testProject.ID, testMergeRequest.IID),
				ExpectError: regexp.MustCompile("Invalid RFC3339 String Value"),
			},
		},
	})
}

func TestAcc_GitlabProjectMergeRequestNote_nonAdminUser(t *testing.T) {
	testUser := testutil.CreateUsers(t, 1)[0]
	testToken := testutil.CreatePersonalAccessToken(t, testUser)
	testProject := testutil.CreateProject(t)
	testBranch := testutil.CreateBranches(t, testProject, 1)[0]
	testMergeRequest := testutil.CreateMergeRequest(t, testutil.CreateUsers(t, 1)[0], testProject, testBranch.Name, "main")
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectMergeRequestNoteDestroy,
		Steps: []resource.TestStep{
			{
				// lintignore:AT004  // we need the provider configuration here
				Config: fmt.Sprintf(`
				provider "gitlab" {
				  token = "%s"
				}

				resource "gitlab_project_merge_request_note" "test" {
					project           = %d
					merge_request_iid = %d
					body              = "A string"
					created_at        = "2001-03-11T03:45:40Z"
				}`, testToken.Token, testProject.ID, testMergeRequest.IID),
				ExpectError: regexp.MustCompile("Attribute Not Permitted"),
			},
			{
				// lintignore:AT004  // we need the provider configuration here
				Config: fmt.Sprintf(`
				provider "gitlab" {
				  token = "%s"
				}

				resource "gitlab_project_merge_request_note" "test" {
					project           = %d
					merge_request_iid = %d
					body              = "A string"
				}`, testToken.Token, testProject.ID, testMergeRequest.IID),
			},
			{
				ResourceName:      "gitlab_project_merge_request_note.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectMergeRequestNoteDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_merge_request_note" {
			continue
		}
		project, mergeRequestIID, noteID, err := resourceGitlabProjectMergeRequestNoteParseID(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, _, err = testutil.TestGitlabClient.Notes.GetMergeRequestNote(project, mergeRequestIID, noteID)
		if err == nil {
			return fmt.Errorf("the project merge request note %d in project %s for merge request %d still exists", noteID, project, mergeRequestIID)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
