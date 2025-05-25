resource "gitlab_project" "awesome_project" {
  name             = "awesome_project"
  description      = "My awesome project."
  visibility_level = "public"
}

resource "gitlab_project_integration_custom_issue_tracker" "tracker" {
  project     = gitlab_project.awesome_project.id
  project_url = "https://customtracker.com/issues"
  issues_url  = "https://customtracker.com/TEST-:id"
}
