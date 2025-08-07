# Example Usage
resource "gitlab_project_deploy_token" "example" {
  project    = "example/deploying"
  name       = "Example project deploy token"
  username   = "example-username"
  expires_at = "2020-03-14T00:00:00.000Z"

  scopes = ["read_repository", "read_registry"]
}

resource "gitlab_project_deploy_token" "example-two" {
  project    = "12345678"
  name       = "Example project deploy token expires in 24h"
  expires_at = timeadd(timestamp(), "24h")
  scopes     = ["read_repository", "read_registry"]
}
