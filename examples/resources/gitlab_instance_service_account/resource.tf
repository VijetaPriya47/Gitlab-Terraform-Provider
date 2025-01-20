# create a service account
resource "gitlab_instance_service_account" "example_sa" {
  name     = "example-name"
  username = "example-username"
  timeouts = {
    # service accounts do not delete instantly, but you can set up a timeout on deletion
    delete = "3m"
  }
}

resources "gitlab_personal_access_token" "example_token" {
  user_id    = gitlab_instance_service_account.example_sa.service_account_id
  name       = "Example personal access token for a service account"
  expires_at = "2026-01-01"

  scopes = ["api"]
}
