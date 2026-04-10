resource "gitlab_project_package_protection_rule" "this" {
  project              = 123
  package_name_pattern = "@scope/package-*"
  package_type         = "npm"

  minimum_access_level_for_push   = "owner"
  minimum_access_level_for_delete = "admin"
}
