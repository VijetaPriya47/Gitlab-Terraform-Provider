resource "gitlab_project" "awesome_project" {
  name             = "awesome_project"
  description      = "My awesome project."
  visibility_level = "public"
}

resource "gitlab_integration_redmine" "redmine" {
  project       = gitlab_project.awesome_project.id
  new_issue_url = "https://redmine.example.com/issue"
  project_url   = "https://redmine.example.com/project"
  issues_url    = "https://redmine.example.com/issue/:id"
}
