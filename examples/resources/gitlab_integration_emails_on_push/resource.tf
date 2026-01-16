# This resource is deprecated and will be removed in version 19.0. Use gitlab_project_integration_emails_on_push instead.

resource "gitlab_project" "awesome_project" {
  name             = "awesome_project"
  description      = "My awesome project."
  visibility_level = "public"
}

resource "gitlab_integration_emails_on_push" "emails" {
  project    = gitlab_project.awesome_project.id
  recipients = "myrecipient@example.com myotherrecipient@example.com"
}
