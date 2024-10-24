resource "gitlab_group" "example" {
  name        = "example"
  path        = "example"
  description = "An example group"
}

resource "gitlab_group_service_account" "example-sa" {
  group    = gitlab_group.example.id
  name     = "example-name"
  username = "example-username"
}

resource "gitlab_group_service_account_access_token" "example-sa-token" {
  group      = gitlab_group.example.id
  user_id    = gitlab_group_service_account.example-sa.service_account_id
  name       = "Example personal access token"
  expires_at = "2020-03-14"

  scopes = ["api"]
}
