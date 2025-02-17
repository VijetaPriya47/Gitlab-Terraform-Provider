# Create a project runner
resource "gitlab_user_runner" "project_runner" {
  runner_type = "project_type"
  project_id  = 123456

  description = "A runner created using a user access token instead of a registration token"
  tag_list    = ["a-tag", "other-tag"]
  untagged    = true
}

# Create a group runner
resource "gitlab_user_runner" "group_runner" {
  runner_type = "group_type"
  group_id    = 123456

  # populate other attributes...
}

# Create a instance runner
resource "gitlab_user_runner" "instance_runner" {
  runner_type = "instance_type"

  # populate other attributes...
}

# Create a configuration string you can write to a file file that can be used to start a gitlab-runner on a remote machine
# This could be used in startup scripts in major cloud providers to automatically create a runner
# See GitLab Runner Advanced Configuration Options here: https://docs.gitlab.com/runner/configuration/advanced-configuration/
locals {
  config_toml = <<-EOT
concurrent = 1
check_interval = 0

[session_server]
  session_timeout = 1800

[[runners]]
  name = "my_gitlab_runner"
  url = "https://example.gitlab.com"
  token = "${gitlab_user_runner.group_runner.token}"
  executor = "docker"

  [runners.custom_build_dir]
  [runners.cache]
    [runners.cache.s3]
    [runners.cache.gcs]
    [runners.cache.azure]
  [runners.docker]
    tls_verify = false
    image = "ubuntu"
    privileged = true
    disable_entrypoint_overwrite = false
    oom_kill_disable = false
    disable_cache = false
    volumes = ["/cache", "/certs/client"]
    shm_size = 0
  EOT
}
