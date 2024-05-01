resource "gitlab_project_job_token_scopes" "allowed_single_project" {
  project            = "gitlab-org/gitlab"
  target_project_ids = [123]
}

resource "gitlab_project_job_token_scopes" "allowed_multiple_project" {
  project            = "gitlab-org/gitlab"
  target_project_ids = [123, 456, 789]
}

# This will remove all job token scopes, even if added outside of TF.
resource "gitlab_project_job_token_scopes" "explicit_deny" {
  project            = "gitlab-org/gitlab"
  target_project_ids = []
}
