resource "gitlab_runner_controller" "example" {
  description = "My runner controller"
  state       = "enabled"
}
