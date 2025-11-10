//go:build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabInstanceVariable_basic(t *testing.T) {
	var instanceVariable gitlab.InstanceVariable
	rString := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabInstanceVariableDestroy,
		Steps: []resource.TestStep{
			// Create a variable with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_instance_variable" "foo" {
				  		key = "key_%s"
				  		value = "value-%s"
				  		variable_type = "file"
				  		masked = false
					}
				`, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabInstanceVariableExists("gitlab_instance_variable.foo", &instanceVariable),
					testAccCheckGitlabInstanceVariableAttributes(&instanceVariable, &testAccGitlabInstanceVariableExpectedAttributes{
						Key:   fmt.Sprintf("key_%s", rString),
						Value: fmt.Sprintf("value-%s", rString),
					}),
				),
			},
			// Update the instance variable to toggle all the values to their inverse
			{
				Config: fmt.Sprintf(`
					resource "gitlab_instance_variable" "foo" {
				  		key = "key_%s"
				  		value = "value-inverse-%s"
				  		protected = true
				  		masked = false
					}
				`, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabInstanceVariableExists("gitlab_instance_variable.foo", &instanceVariable),
					testAccCheckGitlabInstanceVariableAttributes(&instanceVariable, &testAccGitlabInstanceVariableExpectedAttributes{
						Key:       fmt.Sprintf("key_%s", rString),
						Value:     fmt.Sprintf("value-inverse-%s", rString),
						Protected: true,
					}),
				),
			},
			// Update the instance variable to toggle the options back
			{
				Config: fmt.Sprintf(`
					resource "gitlab_instance_variable" "foo" {
				  		key = "key_%s"
				  		value = "value-%s"
				  		variable_type = "file"
				  		masked = false
					}
				`, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabInstanceVariableExists("gitlab_instance_variable.foo", &instanceVariable),
					testAccCheckGitlabInstanceVariableAttributes(&instanceVariable, &testAccGitlabInstanceVariableExpectedAttributes{
						Key:       fmt.Sprintf("key_%s", rString),
						Value:     fmt.Sprintf("value-%s", rString),
						Protected: false,
					}),
				),
			},
			// Update the instance variable to enable "masked" for a value that does not meet masking requirements, and expect an error with no state change.
			// ref: https://docs.gitlab.com/ci/variables/#mask-a-cicd-variable
			{
				Config: fmt.Sprintf(`
					resource "gitlab_instance_variable" "foo" {
				  		key = "key_%s"
				  		value = <<EOF
						value-%s"
						i am multiline
						EOF
				  		masked = true
					}
				`, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabInstanceVariableExists("gitlab_instance_variable.foo", &instanceVariable),
					testAccCheckGitlabInstanceVariableAttributes(&instanceVariable, &testAccGitlabInstanceVariableExpectedAttributes{
						Key:   fmt.Sprintf("key_%s", rString),
						Value: fmt.Sprintf("value-%s", rString),
					}),
				),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(
					"Invalid value for a masked variable. Check the masked variable requirements: https://docs.gitlab.com/ci/variables/#masked-variable-requirements",
				)),
			},
			// Update the instance variable to to enable "masked" and meet masking requirements
			// ref: https://docs.gitlab.com/ci/variables/#mask-a-cicd-variable
			{
				Config: fmt.Sprintf(`
					resource "gitlab_instance_variable" "foo" {
				  		key = "key_%s"
				  		value = "value-%s"
				  		masked = true
					}
				`, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabInstanceVariableExists("gitlab_instance_variable.foo", &instanceVariable),
					testAccCheckGitlabInstanceVariableAttributes(&instanceVariable, &testAccGitlabInstanceVariableExpectedAttributes{
						Key:    fmt.Sprintf("key_%s", rString),
						Value:  fmt.Sprintf("value-%s", rString),
						Masked: true,
					}),
				),
			},
			// Update the instance variable to toggle the options back
			{
				Config: fmt.Sprintf(`
					resource "gitlab_instance_variable" "foo" {
				  		key = "key_%s"
				  		value = "value-%s"
				  		variable_type = "file"
				  		masked = false
					}
				`, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabInstanceVariableExists("gitlab_instance_variable.foo", &instanceVariable),
					testAccCheckGitlabInstanceVariableAttributes(&instanceVariable, &testAccGitlabInstanceVariableExpectedAttributes{
						Key:       fmt.Sprintf("key_%s", rString),
						Value:     fmt.Sprintf("value-%s", rString),
						Protected: false,
					}),
				),
			},
			// Update the instance variable to set the description
			{
				Config: fmt.Sprintf(`
					resource "gitlab_instance_variable" "foo" {
				  		key = "key_%s"
				  		value = "value-%s"
				  		description = "description-%s"
				  		variable_type = "file"
				  		masked = false
					}
				`, rString, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabInstanceVariableExists("gitlab_instance_variable.foo", &instanceVariable),
					testAccCheckGitlabInstanceVariableAttributes(&instanceVariable, &testAccGitlabInstanceVariableExpectedAttributes{
						Key:         fmt.Sprintf("key_%s", rString),
						Value:       fmt.Sprintf("value-%s", rString),
						Description: fmt.Sprintf("description-%s", rString),
					}),
				),
			},
		},
	})
}

func testAccCheckGitlabInstanceVariableExists(n string, instanceVariable *gitlab.InstanceVariable) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		key := rs.Primary.Attributes["key"]
		if key == "" {
			return fmt.Errorf("No variable key is set")
		}

		gotVariable, _, err := testutil.TestGitlabClient.InstanceVariables.GetVariable(key)
		if err != nil {
			return err
		}
		*instanceVariable = *gotVariable
		return nil
	}
}

func testAccCheckGitlabInstanceVariableDestroy(s *terraform.State) error {
	var key string

	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_instance_variable" {
			key = rs.Primary.ID
		}
	}

	iv, _, err := testutil.TestGitlabClient.InstanceVariables.GetVariable(key)
	if err == nil {
		if iv != nil {
			return fmt.Errorf("Instance Variable %s still exists", key)
		}
	} else {
		if !api.Is404(err) {
			return err
		}
	}

	return nil
}

type testAccGitlabInstanceVariableExpectedAttributes struct {
	Key         string
	Value       string
	Description string
	Protected   bool
	Masked      bool
	Raw         bool
}

func testAccCheckGitlabInstanceVariableAttributes(variable *gitlab.InstanceVariable, want *testAccGitlabInstanceVariableExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if variable.Key != want.Key {
			return fmt.Errorf("got key %s; want %s", variable.Key, want.Key)
		}

		if variable.Value != want.Value {
			return fmt.Errorf("got value %s; value %s", variable.Value, want.Value)
		}

		if variable.Description != want.Description {
			return fmt.Errorf("got description %s; description %s", variable.Description, want.Description)
		}

		if variable.Protected != want.Protected {
			return fmt.Errorf("got protected %t; want %t", variable.Protected, want.Protected)
		}

		if variable.Masked != want.Masked {
			return fmt.Errorf("got masked %t; want %t", variable.Masked, want.Masked)
		}

		return nil
	}
}
