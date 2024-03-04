# By project ID and tag_name
data "gitlab_release" "example" {
  project_id = 1234
  tag_name   = "v1.0"
}
