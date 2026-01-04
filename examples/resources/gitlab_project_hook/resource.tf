resource "gitlab_project_hook" "example" {
  project               = "example/hooked"
  url                   = "https://example.com/hook/example"
  name                  = "example"
  description           = "Example hook"
  merge_requests_events = true

  # Set to false to avoid default true value
  push_events = false
}

# Using Custom Headers
# Values of headers can't be imported
resource "gitlab_project_hook" "custom_headers" {
  project               = "example/hooked"
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
