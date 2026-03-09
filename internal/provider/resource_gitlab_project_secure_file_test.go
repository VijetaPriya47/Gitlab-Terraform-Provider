//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAcc_GitlabProjectSecureFile_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAcc_GitlabSecureFile_CheckDestroy(),

		Steps: []resource.TestStep{
			// Create a basic secure file.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_secure_file" "basic" {
					name     = %q
					project = %d
					content = "supersecure"
				}`, name, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_secure_file.basic", "name", name),
					resource.TestCheckResourceAttr("gitlab_project_secure_file.basic", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_secure_file.basic", "content", "supersecure"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_project_secure_file.basic",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content"},
			},
		},
	})
}

func testAcc_GitlabSecureFile_CheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type == "gitlab_project_secure_file" {
				projectID, secureFileID, err := utils.ParseTwoPartID(rs.Primary.ID)
				if err != nil {
					return err
				}

				secureFileIID, err := strconv.Atoi(secureFileID)
				if err != nil {
					return fmt.Errorf("Error in converting string to int")
				}
				secureFile, _, err := testutil.TestGitlabClient.SecureFiles.ShowSecureFileDetails(projectID, int64(secureFileIID))
				if err == nil {
					return fmt.Errorf("Found GitLab application that should have been deleted: %s", gitlab.Stringify(secureFile))
				}
			}
		}
		return nil
	}
}
