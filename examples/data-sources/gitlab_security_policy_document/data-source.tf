# Use this with `gitlab_repository_file` to manage your policies using native HCL
data "gitlab_security_policy_document" "scan" {
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
    }
  ]
}

# See `gitlab_project_security_policy_attachment` or `gitlab_group_security_policy_attachment`
# for how to link a security policy project to a project or group.
resource "gitlab_repository_file" "policy" {
  # The security policy project linked to the group or project
  project = 1234
  ref     = "main"

  # This name is important, don't change it!
  # see https://docs.gitlab.com/user/application_security/policies/enforcement/security_policy_projects/
  file_path = ".gitlab/security-policies/policy.yml"
  content   = data.gitlab_security_policy_document.scan.yaml
}
