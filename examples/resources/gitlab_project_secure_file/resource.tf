resource "gitlab_group" "this" {
  name        = "example"
  path        = "example"
  description = "An example group"
}

resource "gitlab_project" "this" {
  name                   = "example"
  namespace_id           = gitlab_group.this.id
  initialize_with_readme = true
}

resource "gitlab_project_secure_file" "this" {
  name    = "my-secure-file"
  project = gitlab_project.this.id
  content = file("example.txt")
}

resource "gitlab_project_secure_file" "disable_poll_for_metadata" {
  name                                  = "my-secure-file"
  project                               = gitlab_project.this.id
  content                               = file("example.txt")
  poll_for_metadata_duration_in_seconds = 0
}
