data "gitlab_project_mirror_public_key" "example" {
  project_id = 30
  mirror_id  = 42
}

data "gitlab_project_mirror_public_key" "example" {
  project_id = "foo/bar/baz"
  mirror_id  = 123
}
