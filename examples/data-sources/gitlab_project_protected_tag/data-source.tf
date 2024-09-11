data "gitlab_project_protected_tag" "example" {
  project = 42
  tag     = "v1.0"
}

data "gitlab_project_protected_tag" "example" {
  project = "foo/bar/baz"
  tag     = "2.0"
}
