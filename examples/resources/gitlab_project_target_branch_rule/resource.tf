resource "gitlab_project" "example" {
  name             = "example project"
  description      = "Lorem Ipsum"
  visibility_level = "public"
}

// Basic example
resource "gitlab_project_target_branch_rule" "rule" {
  project               = gitlab_project.example.id
  source_branch_pattern = "develop"
  target_branch_name    = "release"
}
