//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabComplianceFramework_basic(t *testing.T) {

	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testComplianceFramework := testutil.CreateComplianceFramework(t, testGroup)

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_compliance_framework" "test" {
						namespace_path = "%s"
						name = "%s"
					}
						`, testGroup.FullPath, testComplianceFramework.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_compliance_framework.test", "name", testComplianceFramework.Name),
					resource.TestCheckResourceAttr("data.gitlab_compliance_framework.test", "framework_id", testComplianceFramework.ID),
					resource.TestCheckResourceAttr("data.gitlab_compliance_framework.test", "description", testComplianceFramework.Description),
					resource.TestCheckResourceAttr("data.gitlab_compliance_framework.test", "color", testComplianceFramework.Color),
				),
			},
		},
	})
}

func TestAccDataGitlabComplianceFramework_EnsureErrorOnFrameworkNotFound(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	err_regex, err := regexp.Compile("Compliance Framework not found")
	if err != nil {
		t.Errorf("Unable to format expected compliance framework error regex: %s", err)
	}

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_compliance_framework" "test" {
						namespace_path = "%s"
						name = "DNE"
					}
						`, testGroup.FullPath),
				ExpectError: err_regex,
			},
		},
	})
}

func TestAccDataGitlabComplianceFramework_EnsureErrorOnMatchingFrameworkNotFound(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	//create a compliance framework; name should start with 'Compliance Framework'
	testutil.CreateComplianceFramework(t, testGroup)

	err_regex, err := regexp.Compile("Compliance Framework match not found")
	if err != nil {
		t.Errorf("Unable to format expected compliance framework error regex: %s", err)
	}

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_compliance_framework" "test" {
						namespace_path = "%s"
						name = "Compliance Framework"
					}
						`, testGroup.FullPath),
				ExpectError: err_regex,
			},
		},
	})
}
