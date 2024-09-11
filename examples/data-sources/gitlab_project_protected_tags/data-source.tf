data "gitlab_project_protected_tags" "example" {
  project = 42
}

data "gitlab_project_protected_tags" "example" {
  project = "foo/bar/baz"
}
