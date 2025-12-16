//go:build acceptance

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceGitlabSecurityPolicyDocument_scanExecution_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "gitlab_security_policy_document" "test" {
  scan_execution_policy = [
    {
      name    = "Basic SAST Policy"
      enabled = true
      
      rules = [
        {
          type        = "pipeline"
          branch_type = "all"
        }
      ]
      
      actions = [
        {
          scan = "sast"
        }
      ]
      
      skip_ci = {
        allowed = true
      }
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "id"),
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "yaml"),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan_execution_policy:")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("name: Basic SAST Policy")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("enabled: true")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan: sast")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("type: pipeline")),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabSecurityPolicyDocument_scanExecution_complete(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "gitlab_security_policy_document" "test" {
  scan_execution_policy = [
    {
      name        = "Complete Security Policy"
      description = "Comprehensive security scanning"
      enabled     = true
      
      rules = [
        {
          type        = "pipeline"
          branch_type = "all"
        }
      ]
      
      actions = [
        {
          scan     = "secret_detection"
          template = "latest"
          variables = {
            SECURE_ENABLE_LOCAL_CONFIGURATION = "false"
          }
        },
        {
          scan = "sast"
        },
        {
          scan = "dependency_scanning"
        }
      ]
      
      policy_scope = {
        projects = {
          excluding = [123, 456]
        }
      }
      
      skip_ci = {
        allowed = true
      }
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "id"),
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "yaml"),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan_execution_policy:")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("name: Complete Security Policy")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("description: Comprehensive security scanning")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan: secret_detection")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("template: latest")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan: sast")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan: dependency_scanning")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("branch_type: all")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("policy_scope:")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("excluding:")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("skip_ci:")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("allowed: true")),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabSecurityPolicyDocument_scanExecution_multipleActions(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "gitlab_security_policy_document" "test" {
  scan_execution_policy = [
    {
      name    = "Multi-Scan Policy"
      enabled = true
      
      rules = [
        {
          type        = "pipeline"
          branch_type = "protected"
        }
      ]
      
      actions = [
        {
          scan     = "secret_detection"
          template = "latest"
          variables = {
            SECURE_ENABLE_LOCAL_CONFIGURATION = "false"
          }
        },
        {
          scan = "sast"
          variables = {
            SECURE_ENABLE_LOCAL_CONFIGURATION = "false"
          }
        },
        {
          scan = "dependency_scanning"
        },
        {
          scan = "sast_iac"
          variables = {
            SECURE_ENABLE_LOCAL_CONFIGURATION = "false"
          }
        }
      ]
      
      skip_ci = {
        allowed = true
      }
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "id"),
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "yaml"),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan: secret_detection")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan: sast")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan: dependency_scanning")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("scan: sast_iac")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("SECURE_ENABLE_LOCAL_CONFIGURATION")),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabSecurityPolicyDocument_scanExecution_scheduleRule(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "gitlab_security_policy_document" "test" {
  scan_execution_policy = [
    {
      name    = "Scheduled SAST Policy"
      enabled = true
      
      rules = [
        {
          type     = "schedule"
          cadence  = "0 2 * * *"
          branches = ["main"]
        }
      ]
      
      actions = [
        {
          scan = "sast"
        }
      ]
      
      skip_ci = {
        allowed = false
      }
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "id"),
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "yaml"),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("type: schedule")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("cadence:")),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabSecurityPolicyDocument_scanExecution_multiplePolicies(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "gitlab_security_policy_document" "test" {
  scan_execution_policy = [
    {
      name    = "SAST Policy"
      enabled = true
      
      rules = [
        {
          type        = "pipeline"
          branch_type = "all"
        }
      ]
      
      actions = [
        {
          scan = "sast"
        }
      ]
      
      skip_ci = {
        allowed = true
      }
    },
    {
      name    = "Secret Detection Policy"
      enabled = true
      
      rules = [
        {
          type        = "pipeline"
          branch_type = "all"
        }
      ]
      
      actions = [
        {
          scan = "secret_detection"
        }
      ]
      
      skip_ci = {
        allowed = true
      }
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "id"),
					resource.TestCheckResourceAttrSet("data.gitlab_security_policy_document.test", "yaml"),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("name: SAST Policy")),
					resource.TestMatchResourceAttr("data.gitlab_security_policy_document.test", "yaml", regexp.MustCompile("name: Secret Detection Policy")),
				),
			},
		},
	})
}
