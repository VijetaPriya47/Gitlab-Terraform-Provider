# Create a project for the milestone to use
resource "gitlab_project" "example" {
  name         = "example"
  description  = "An example project"
  namespace_id = gitlab_group.example.id
}

# Basic milestone with required fields only
resource "gitlab_project_milestone" "example" {
  project = gitlab_project.example.id
  title   = "example"
}

# Comprehensive milestone with all optional fields
resource "gitlab_project_milestone" "comprehensive" {
  project     = gitlab_project.example.id
  title       = "Q4 2024 Release"
  description = "Major release for Q4 2024"
  start_date  = "2024-01-01"
  due_date    = "2024-12-31"
  state       = "active"
}
