//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectCustomAttribute_basic(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectCustomAttributeDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "gitlab_project_custom_attribute" "attr" {
	project = "%d"
	key     = "foo"
	value   = "bar"
}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_custom_attribute.attr", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_custom_attribute.attr", "key", "foo"),
					resource.TestCheckResourceAttr("gitlab_project_custom_attribute.attr", "value", "bar"),
				),
			},
			// Update the custom attribute
			{
				Config: fmt.Sprintf(`
resource "gitlab_project_custom_attribute" "attr" {
	project = "%d"
	key     = "foo"
	value   = "updated"
}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_custom_attribute.attr", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_custom_attribute.attr", "key", "foo"),
					resource.TestCheckResourceAttr("gitlab_project_custom_attribute.attr", "value", "updated"),
				),
			},
			{
				ResourceName:      "gitlab_project_custom_attribute.attr",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectCustomAttributeDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_custom_attribute" {
			continue
		}

		parts := strings.SplitN(rs.Primary.ID, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("unexpected ID format (%q). Expected project-id:key", rs.Primary.ID)
		}

		projectID, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("Unable to parse project id (%q) into an integer", rs.Primary.ID)
		}

		attribute, _, err := testutil.TestGitlabClient.CustomAttribute.GetCustomProjectAttribute(projectID, parts[1])
		if err == nil && attribute != nil {
			return fmt.Errorf("Project custom attribute still exists")
		}

		if !api.Is404(err) {
			return err
		}

		return nil
	}
	return nil
}
