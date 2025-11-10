//go:build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabUserCustomAttribute_basic(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	var customAttribute gitlab.CustomAttribute

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "gitlab_user_custom_attribute" "attr" {
	user  = %d
	key   = "foo"
	value = "bar"
}`, user.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserCustomAttributeExists("gitlab_user_custom_attribute.attr", &customAttribute),
					testAccCheckGitlabUserCustomAttributes(&customAttribute, &testAccGitlabUserExpectedCustomAttributes{
						Key:   "foo",
						Value: "bar",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user_custom_attribute.attr",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the custom attribute
			{
				Config: fmt.Sprintf(`
resource "gitlab_user_custom_attribute" "attr" {
	user  = %d
	key   = "foo"
	value = "updated"
}`, user.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabUserCustomAttributeExists("gitlab_user_custom_attribute.attr", &customAttribute),
					testAccCheckGitlabUserCustomAttributes(&customAttribute, &testAccGitlabUserExpectedCustomAttributes{
						Key:   "foo",
						Value: "updated",
					}),
				),
			},
			{
				ResourceName:      "gitlab_user_custom_attribute.attr",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabUserCustomAttributeExists(n string, customAttribute *gitlab.CustomAttribute) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		id, key, err := parseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		gotCustomAttribute, _, err := testutil.TestGitlabClient.CustomAttribute.GetCustomUserAttribute(id, key)
		if err != nil {
			return err
		}
		*customAttribute = *gotCustomAttribute
		return nil
	}
}

type testAccGitlabUserExpectedCustomAttributes struct {
	Key   string
	Value string
}

func testAccCheckGitlabUserCustomAttributes(got *gitlab.CustomAttribute, want *testAccGitlabUserExpectedCustomAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if got.Key != want.Key {
			return fmt.Errorf("got key %q; want %q", got.Key, want.Key)
		}

		if got.Value != want.Value {
			return fmt.Errorf("got value %q; want %q", got.Value, want.Value)
		}

		return nil
	}
}
