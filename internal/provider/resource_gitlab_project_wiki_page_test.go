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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabWikiPage_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabWikiPageDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				  resource "gitlab_project_wiki_page" "example" {
					project     = "%d"
					title       = "Test Wiki Page"
					content     = "This is a test wiki page"
					format      = "markdown"
				  }
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					// Check that the values are set correctly for the resource attributes
					resource.TestCheckResourceAttr("gitlab_project_wiki_page.example", "content", "This is a test wiki page"),
					resource.TestCheckResourceAttr("gitlab_project_wiki_page.example", "format", "markdown"),
					resource.TestCheckResourceAttrSet("gitlab_project_wiki_page.example", "slug"),
					resource.TestCheckResourceAttr("gitlab_project_wiki_page.example", "title", "Test Wiki Page"),
					resource.TestCheckResourceAttr("gitlab_project_wiki_page.example", "encoding", "UTF-8"),
				),
			},

			// Step 2: Import the resource and verify
			{
				ResourceName:            "gitlab_project_wiki_page.example",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"slug"}, // Ignore the slug as it may change during import
			},
			// Step 3: Update one of the attributes and re-verify
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_wiki_page" "example" {
					project     = "%d"
					title       = "Updated Test Wiki Page"
					content     = "This is an updated test wiki page"
					format      = "markdown"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					// Verify the updated values
					resource.TestCheckResourceAttr("gitlab_project_wiki_page.example", "content", "This is an updated test wiki page"),
					resource.TestCheckResourceAttr("gitlab_project_wiki_page.example", "format", "markdown"),
					resource.TestCheckResourceAttr("gitlab_project_wiki_page.example", "title", "Updated Test Wiki Page"),
					resource.TestCheckResourceAttr("gitlab_project_wiki_page.example", "encoding", "UTF-8"),
				),
			},
		},
	})
}

// The unique identifier for the wiki page. Slug must not contain the `:` character!
func TestAccGitlabWikiPage_TestSlugValidation(t *testing.T) {
	//lintignore:AT001 // This test is only testing validations, no new resources are created, so we don't need to check destroy.
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "gitlab_project_wiki_page" "test" {
						project = "12345"
						title   = "Invalid:Title"
						content = "This is a test."
					}`,
				ExpectError: regexp.MustCompile(`Title must not contain ':' as it generates an invalid slug`),
			},
		},
	})
}

func testAccCheckGitlabWikiPageDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_wiki_page" {
			continue
		}

		project, slug, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		// Make API call here to try and read the wiki page.
		_, _, err = testutil.TestGitlabClient.Wikis.GetWikiPage(project, slug, &gitlab.GetWikiPageOptions{})
		if err == nil {
			return fmt.Errorf("Wiki page still exists, should have been destroyed by the tests")
		}
		if !api.Is404(err) {
			return err
		}
	}
	return nil
}
