# Example is creating a role for a security analyst to be able to read vulnerabilities for repos in example-org on gitlab.com.

resource "gitlab_member_role" "example" {
  name              = "Security Analyst"
  description       = "This is a read only role for security analysts"
  group_path        = "example-org"
  base_access_level = "REPORTER"
  enabled_permissions = [
    "READ_VULNERABILITY"
  ]
}

# Example is creating a role for an enhanced developer to be able to manage CI/CD variables and vulnerabilities on self managed gitlab instance.

resource "gitlab_member_role" "example" {
  name              = "Enhanced Developer"
  description       = "This role gives the developers additonal access to manage CI/CD variables and vulnerabilities"
  base_access_level = "DEVELOPER"
  enabled_permissions = [
    "ADMIN_CICD_VARIABLES",
    "ADMIN_VULNERABILITY"
  ]
}
