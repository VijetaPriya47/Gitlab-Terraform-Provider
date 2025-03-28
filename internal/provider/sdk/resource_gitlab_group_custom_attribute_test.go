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

func TestAccGitlabGroupCustomAttribute_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupCustomAttributesDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "gitlab_group_custom_attribute" "attr" {
	group = %d
	key   = "foo"
	value = "bar"
}`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_custom_attribute.attr", "group", strconv.Itoa(group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_custom_attribute.attr", "key", "foo"),
					resource.TestCheckResourceAttr("gitlab_group_custom_attribute.attr", "value", "bar"),
				),
			},
			// Update the custom attribute
			{
				Config: fmt.Sprintf(`
resource "gitlab_group_custom_attribute" "attr" {
	group = %d
	key   = "foo"
	value = "updated"
}`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_custom_attribute.attr", "group", strconv.Itoa(group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_custom_attribute.attr", "key", "foo"),
					resource.TestCheckResourceAttr("gitlab_group_custom_attribute.attr", "value", "updated"),
				),
			},
			{
				ResourceName:      "gitlab_group_custom_attribute.attr",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabGroupCustomAttributesDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_custom_attribute" {
			continue
		}

		parts := strings.SplitN(rs.Primary.ID, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("unexpected ID format (%q). Expected group-id:key", rs.Primary.ID)
		}

		groupID, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("Unable to parse group id (%q) into an integer", rs.Primary.ID)
		}

		attribute, _, err := testutil.TestGitlabClient.CustomAttribute.GetCustomGroupAttribute(groupID, parts[1])
		if err == nil && attribute != nil {
			return fmt.Errorf("Group custom attribute still exists")
		}

		if !api.Is404(err) {
			return err
		}

		return nil
	}
	return nil
}
