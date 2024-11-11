resource "gitlab_project" "this" {
  name                   = "example"
  initialize_with_readme = true
}

data "gitlab_project_environments" "this" {
  project = gitlab_project.this.path_with_namespace # Can also use project id
}
