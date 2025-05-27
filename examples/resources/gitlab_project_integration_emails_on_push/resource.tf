resource "gitlab_project" "awesome_project" {
  name             = "awesome_project"
  description      = "My awesome project."
  visibility_level = "public"
}

resource "gitlab_project_integration_emails_on_push" "emails" {
  project    = gitlab_project.awesome_project.id
  recipients = "myrecipient@example.com myotherrecipient@example.com"
}
