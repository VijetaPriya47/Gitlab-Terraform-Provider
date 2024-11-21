//go:build acceptance
// +build acceptance

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	framework_testutil "gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
)

func TestAcc_GitLabMetadata_DataSource_Basic(t *testing.T) {

	//lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: `data "gitlab_metadata" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify id attribute
					resource.TestCheckResourceAttr("data.gitlab_metadata.test", "id", "1"),
					// Verify attributes are set
					resource.TestCheckResourceAttrSet("data.gitlab_metadata.test", "version"),
					resource.TestCheckResourceAttrSet("data.gitlab_metadata.test", "revision"),
					resource.TestCheckResourceAttrSet("data.gitlab_metadata.test", "kas.enabled"),
					resource.TestCheckResourceAttrSet("data.gitlab_metadata.test", "kas.external_url"),
					framework_testutil.TestCheckResourceAttrSetIfGitLabAtLeast(t, "17.6", "data.gitlab_metadata.test", "kas.external_k8s_proxy_url"),
					resource.TestCheckResourceAttrSet("data.gitlab_metadata.test", "kas.version"),
					resource.TestCheckResourceAttrSet("data.gitlab_metadata.test", "enterprise"),
				),
			},
		},
	})
}
