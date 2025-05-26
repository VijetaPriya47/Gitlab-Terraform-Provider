resource "gitlab_project" "example" {
  name             = "example project"
  description      = "Lorem Ipsum"
  visibility_level = "public"
}

// Basic example
resource "gitlab_project_level_notifications" "notifications" {
  project = gitlab_project.example.id
  level   = "global"
}

// Custom notification example
resource "gitlab_project_level_notifications" "custom" {
  project           = gitlab_project.example.id
  level             = "custom"
  new_merge_request = true
}