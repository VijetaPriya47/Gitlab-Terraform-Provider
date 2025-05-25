data "gitlab_group" "example" {
  id = "foo/bar/baz"
}

# Basic example
data "gitlab_group_service_account" "example" {
  service_account_id = 1
  group              = data.gitlab_group.example.id
}
