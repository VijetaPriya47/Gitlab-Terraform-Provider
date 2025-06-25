resource "gitlab_group_hook" "example" {
  group                 = "example/hooked"
  url                   = "https://example.com/hook/example"
  name                  = "Example"
  description           = "Example Group Webhook"
  merge_requests_events = true
}

# Setting all attributes
resource "gitlab_group_hook" "all_attributes" {
  group                      = 1
  url                        = "http://example.com"
  name                       = "Example"
  description                = "Example Group Webhook"
  token                      = "supersecret"
  enable_ssl_verification    = false
  push_events                = true
  push_events_branch_filter  = "devel"
  issues_events              = false
  confidential_issues_events = false
  merge_requests_events      = true
  tag_push_events            = true
  note_events                = true
  confidential_note_events   = true
  job_events                 = true
  pipeline_events            = true
  wiki_page_events           = true
  deployment_events          = true
  releases_events            = true
  subgroup_events            = true
  emoji_events               = true
  feature_flag_events        = true
  branch_filter_strategy     = "wildcard"
}

# Using Custom Headers
# Values of headers can't be imported
resource "gitlab_group_hook" "all_attributes" {
  group                 = "example/hooked"
  url                   = "https://example.com/hook/example"
  merge_requests_events = true

  custom_headers = [
    {
      key   = "X-Custom-Header"
      value = "example"
    },
    {
      key   = "X-Custom-Header-Second"
      value = "example-second"
    }
  ]
}
