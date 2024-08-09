resource "gitlab_project_job_token_scopes" "allowed_single_project" {
  project            = "111"
  target_project_ids = [123]
}

resource "gitlab_project_job_token_scopes" "allowed_multiple_project" {
  project            = "111"
  target_project_ids = [123, 456, 789]
}

resource "gitlab_project_job_token_scopes" "allowed_multiple_groups" {
  project_id         = 111
  target_project_ids = []
  target_group_ids   = [321, 654]
}

# This will remove all job token scopes, even if added outside of TF.
resource "gitlab_project_job_token_scopes" "explicit_deny" {
  project            = "111"
  target_project_ids = []
}
