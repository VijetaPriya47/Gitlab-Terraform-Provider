# This must be a top-level group
resource "gitlab_group" "example" {
  name        = "example"
  path        = "example"
  description = "An example group"
}

# The service account against the top-level group
resource "gitlab_group_service_account" "example_sa" {
  group    = gitlab_group.example.id
  name     = "example-name"
  username = "example-username"
}

# Group to assign the service account to. Can be the same top-level group resource as above, or a subgroup of that group.
resource "gitlab_group" "example_subgroup" {
  name        = "subgroup"
  path        = "example/subgroup"
  description = "An example subgroup"
}

# To assign the service account to a group
resource "gitlab_group_membership" "example_membership" {
  group_id     = gitlab_group.example_subgroup.id
  user_id      = gitlab_group_service_account.example_sa.service_account_id
  access_level = "developer"
  expires_at   = "2020-03-14"
}