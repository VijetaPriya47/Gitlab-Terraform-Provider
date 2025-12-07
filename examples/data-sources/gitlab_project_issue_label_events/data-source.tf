data "gitlab_project_issue_label_events" "example" {
  project   = "my-group/my-project"
  issue_iid = 42
}
