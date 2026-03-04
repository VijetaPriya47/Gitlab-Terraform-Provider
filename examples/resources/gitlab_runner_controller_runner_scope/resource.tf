resource "gitlab_runner_controller_runner_scope" "example" {
  runner_controller_id = gitlab_runner_controller.example.id
  runner_id            = 42
}
