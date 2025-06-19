resource "gitlab_project" "example" {
  name = "project"
}

# Example where the resource will not reset the settings on destroy
resource "gitlab_project_pages_settings" "this" {
  project                  = gitlab_project.example.id
  is_unique_domain_enabled = true
}

# Example where the resource will reset the settings to their original values on destroy
resource "gitlab_project_pages_settings" "this" {
  project                  = gitlab_project.example.id
  is_unique_domain_enabled = true
  keep_settings_on_destroy = false
}
