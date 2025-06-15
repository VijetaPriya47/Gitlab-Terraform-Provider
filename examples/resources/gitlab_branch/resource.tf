# Create a project for the branch to use
resource "gitlab_project" "example" {
  name         = "example"
  description  = "An example project"
  namespace_id = gitlab_group.example.id
}

resource "gitlab_branch" "example" {
  name    = "example"
  ref     = "main"
  project = gitlab_project.example.id

  # Recommended for imports and divergent branches
  # to prevent resource destroy and recreate.
  lifecycle {
    ignore_changes = [ref]
  }
}

