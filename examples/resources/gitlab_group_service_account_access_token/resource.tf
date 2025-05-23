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

# To assign the service account to a group
resource "gitlab_group_membership" "example_membership" {
  group_id     = gitlab_group.example.id
  user_id      = gitlab_group_service_account.example_sa.service_account_id
  access_level = "developer"
  expires_at   = "2020-03-14"
}

# The service account access token with no expiry
resource "gitlab_group_service_account_access_token" "example_sa_token_no_expiry" {
  group   = gitlab_group.example.id
  user_id = gitlab_group_service_account.example_sa.service_account_id
  name    = "Example service account access token"

  scopes = ["api"]
}

# The service account access token with expires at
resource "gitlab_group_service_account_access_token" "example_sa_token_expires_at" {
  group      = gitlab_group.example.id
  user_id    = gitlab_group_service_account.example_sa.service_account_id
  name       = "Example service account access token"
  expires_at = "2020-03-14"

  scopes = ["api"]
}

# The service account access token with rotation configuration
resource "gitlab_group_service_account_access_token" "example_sa_token_rotation_configuration" {
  group   = gitlab_group.example.id
  user_id = gitlab_group_service_account.example_sa.service_account_id
  name    = "Example service account access token"

  rotation_configuration = {
    rotate_before_days = 2
    expiration_days    = 7
  }

  scopes = ["api"]
}
