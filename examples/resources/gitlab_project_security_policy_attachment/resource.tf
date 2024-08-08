# This resource can be used to attach a security policy to a pre-existing project
resource "gitlab_project_security_policy_attachment" "foo" {
  project        = 1234
  policy_project = 4567
}


# Or you can use Terraform to create a new project, add a policy to that project,
# then attach that policy project to other projects.
resource "gitlab_project" "my-policy-project" {
  name = "security-policy-project"
}

resource "gitlab_repository_file" "policy-yml" {
  project   = gitlab_project.my-policy-project.id
  file_path = ".gitlab/security-policies/my-policy.yml"
  branch    = "master"
  encoding  = "text"
  content   = <<-EOT
---
approval_policy:
- name: test
description: test
enabled: true
rules:
- type: any_merge_request
    branch_type: protected
    commits: any
approval_settings:
    block_branch_modification: true
    prevent_pushing_and_force_pushing: true
    prevent_approval_by_author: true
    prevent_approval_by_commit_author: true
    remove_approvals_with_new_commit: true
    require_password_to_approve: false
fallback_behavior:
    fail: closed
actions:
- type: send_bot_message
    enabled: true
EOT
}

resource "gitlab_project_security_policy_attachment" "my-policy" {
  project        = 1234
  policy_project = gitlab_project.my-policy-project.id
}
