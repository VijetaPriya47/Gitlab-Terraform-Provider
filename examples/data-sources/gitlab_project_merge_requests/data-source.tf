data "gitlab_project_merge_requests" "example_one" {
  project       = "123"
  target_branch = "main"
  wip           = "yes"
}

data "gitlab_project_merge_requests" "example_two" {
  project       = "company/group/project1"
  author_id     = 5
  created_after = "2024-07-25T12:00:00Z"
}
