# Example Usage
resource "gitlab_group_deploy_token" "example" {
  group      = "example/deploying"
  name       = "Example group deploy token"
  username   = "example-username"
  expires_at = "2020-03-14T00:00:00.000Z"

  scopes = ["read_repository", "read_registry"]
}

resource "gitlab_group_deploy_token" "example-two" {
  group      = "12345678"
  name       = "Example group deploy token expires in 24h"
  expires_at = timeadd(timestamp(), "24h")
  scopes     = ["read_repository", "read_registry"]
}
