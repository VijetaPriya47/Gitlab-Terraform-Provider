resource "gitlab_project_job_token_scope" "allowed_single_project" {
  project           = "gitlab-org/gitlab"
  target_project_id = 123
}

# Allow multiple projects
locals {
  allowed_project_ids = [123, 456, 789]
}

data "gitlab_project" "deployment_project" {
  name = "example-project"
}

resource "gitlab_project_job_token_scope" "allowed_project" {
  for_each = toset(local.allowed_project_ids)

  project           = data.gitlab_project.deployment_project.id
  target_project_id = each.key
}
