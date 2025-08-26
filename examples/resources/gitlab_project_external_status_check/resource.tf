resource "gitlab_project_external_status_check" "foo" {
  project_id   = 123
  name         = "foo"
  external_url = "https://example.gitlab.com"
}

resource "gitlab_project_external_status_check" "bar" {
  project_id           = 456
  name                 = "bar"
  external_url         = "https://example.gitlab.com"
  shared_secret        = "secret"
  protected_branch_ids = [6, 28]
}
