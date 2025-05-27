data "gitlab_group" "example" {
  id = "foo/bar/baz"
}

# Basic example
data "gitlab_group_provisioned_users" "example" {
  id = data.gitlab_group.example.id
}

# Complex example
data "gitlab_group_provisioned_users" "example" {
  id             = data.gitlab_group.example.id
  created_after  = "2024-09-20T06:15:29Z"
  active         = true
  blocked        = false
  created_before = "2024-09-21T06:15:29Z"
}

# Search example
data "gitlab_group_provisioned_users" "example" {
  id     = data.gitlab_group.example.id
  search = "Hoa nguyen 1"
}