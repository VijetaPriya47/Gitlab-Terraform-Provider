resource "gitlab_project_job_token_scopes" "allowed_single_project" {
  project            = "111"
  target_project_ids = [123]
}

resource "gitlab_project_job_token_scopes" "allowed_multiple_project" {
  project            = "111"
  target_project_ids = [123, 456, 789]
}

resource "gitlab_project_job_token_scopes" "allowed_multiple_groups" {
  project            = 111
  target_project_ids = []
  target_group_ids   = [321, 654]
}

# This will remove all job token scopes, even if added outside of TF.
resource "gitlab_project_job_token_scopes" "explicit_deny" {
  project            = "111"
  target_project_ids = []
}

# This shows the explicit behavior of the enabled flag with a list of projects and groups.
resource "gitlab_project_job_token_scopes" "allow_projects_and_groups" {
  project            = "111"
  enabled            = true
  target_project_ids = [123, 456, 789]
  target_group_ids   = [321, 654]
}

# This allows all projects and groups (disabling the CI Job Token scope protection)
resource "gitlab_project_job_token_scopes" "allow_all" {
  project = "111"
  enabled = false
}
