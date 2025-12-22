resource "gitlab_project_cicd_catalog" "example" {
  project = "namespace/project"
  enabled = true
}
