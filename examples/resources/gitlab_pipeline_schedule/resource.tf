resource "gitlab_pipeline_schedule" "example" {
  project     = "12345"
  description = "Used to schedule builds"
  ref         = "refs/heads/main"
  cron        = "0 1 * * *"
}
