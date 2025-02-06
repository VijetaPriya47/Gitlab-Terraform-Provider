# Create a project
resource "gitlab_project" "example" {
  name        = "example"
  description = "An example project"
}

# Create a release
resource "gitlab_release" "example" {
  project     = gitlab_project.example.id
  name        = "test-release"
  tag_name    = "v1.0.0"
  description = "Test release description"
  ref         = "main"
}
