output "runner_registration_token" {
  description = "Runner registration token to register your runner installation with."
  value       = gitlab_user_runner.linux.token
  sensitive   = true
}
