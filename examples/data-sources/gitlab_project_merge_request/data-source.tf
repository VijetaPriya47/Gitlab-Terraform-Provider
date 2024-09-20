data "gitlab_project_merge_request" "by_project_id" {
  project = "123"
  iid     = 456
}

data "gitlab_project_merge_request" "by_project_name" {
  project = "company/group/project1"
  iid     = 3
}
