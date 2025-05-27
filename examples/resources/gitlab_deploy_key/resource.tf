// No expiry
resource "gitlab_deploy_key" "example" {
  project = "example/deploying"
  title   = "Example deploy key"
  key     = "ssh-ed25519 AAAA..."
}

// With expiry
resource "gitlab_deploy_key" "example_expires" {
  project    = "example/deploying"
  title      = "Example deploy key"
  key        = "ssh-ed25519 AAAA..."
  expires_at = "2025-01-21T00:00:00Z"
}
