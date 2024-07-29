//go:build acceptance
// +build acceptance

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_GitLab_ProviderMux(t *testing.T) {
	//lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					// The gitlab_metadata data source is based on the terraform-plugin-framework
					data "gitlab_metadata" "test" {}
					
					// The gitlab_current_user data source is based on the terraform-plugin-sdk
					data "gitlab_current_user" "test" {}
                `,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify framework based data source attribute
					resource.TestCheckResourceAttr("data.gitlab_metadata.test", "id", "1"),
					// Verify sdk based data source attribute
					resource.TestCheckResourceAttr("data.gitlab_current_user.test", "id", "1"),
				),
			},
		},
	})
}
