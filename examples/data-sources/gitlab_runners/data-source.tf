resource "gitlab_user_runner" "this" {
  runner_type = "instance_type"
  tag_list    = ["tag1", "tag2"]
}

data "gitlab_runners" "this" {
  paused   = false
  status   = "online"
  tag_list = ["tag1", "tag2"]
  type     = "instance_type"
}
