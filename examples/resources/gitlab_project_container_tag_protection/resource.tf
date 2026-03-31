resource "gitlab_project_container_tag_protection" "example" {
  project        = 123
  tag_name_regex = "^v[0-9]+$"
  immutable      = true
}

resource "gitlab_project_container_tag_protection" "protected" {
  project                         = 123
  tag_name_regex                  = "^v[0-9]+\\-rc[0-9]+$"
  minimum_access_level_for_push   = "MAINTAINER"
  minimum_access_level_for_delete = "OWNER"
}
