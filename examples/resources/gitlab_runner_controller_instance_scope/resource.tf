resource "gitlab_runner_controller_instance_scope" "example" {
  runner_controller_id = gitlab_runner_controller.example.id
}
