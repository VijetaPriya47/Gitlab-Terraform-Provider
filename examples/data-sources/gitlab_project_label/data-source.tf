# Retrieve a project label by its ID
data "gitlab_project_label" "example" {
  project  = "385"
  label_id = 24
}

# Retrieve using project path
data "gitlab_project_label" "by_path" {
  project  = "group/project"
  label_id = 25
}
