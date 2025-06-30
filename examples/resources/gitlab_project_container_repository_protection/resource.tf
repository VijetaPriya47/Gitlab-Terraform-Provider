resource "gitlab_project_container_repository_protection" "this" {
  project                 = 123
  repository_path_pattern = "my_namespace/project*"

  minimum_access_level_for_push   = "owner"
  minimum_access_level_for_delete = "admin"
}
