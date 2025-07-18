data "gitlab_group" "example" {
  id = "foo/bar/baz"
}

data "gitlab_group_saml_links" "example" {
  group = data.gitlab_group.example.id
}
