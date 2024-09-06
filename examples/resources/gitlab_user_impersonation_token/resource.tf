resource "gitlab_user_impersonation_token" "this" {
  user_id    = 12345
  name       = "token_name"
  scopes     = ["api"]
  expires_at = "2024-08-27"
}