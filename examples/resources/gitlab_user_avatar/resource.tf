resource "gitlab_personal_access_token" "example" {
  user_id    = "25"
  name       = "Example personal access token"
  expires_at = "2020-03-14"

  scopes = ["api"]
}

resource "gitlab_user_avatar" "example" {
  user_id     = gitlab_personal_access_token.example.user_id
  token       = gitlab_personal_access_token.example.token
  avatar      = "${path.module}/avatar.png"
  avatar_hash = filesha256("${path.module}/avatar.png")
}
