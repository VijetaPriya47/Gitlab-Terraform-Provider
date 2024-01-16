resource "gitlab_global_level_notifications" "foo" {
  level = "watch"
}

# Create Custom global level notification
resource "gitlab_global_level_notifications" "foo" {
  level             = "custom"
  new_merge_request = true
}
