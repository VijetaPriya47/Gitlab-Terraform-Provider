data "gitlab_project_approval_rules" "by_project_id" {
  project = "12345"
}

data "gitlab_project_approval_rules" "by_project_path" {
  project = "my-group/my-project"
}
