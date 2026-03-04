resource "gitlab_runner_controller_token" "example" {
  runner_controller_id = gitlab_runner_controller.example.id
  description          = "My controller token"
}
