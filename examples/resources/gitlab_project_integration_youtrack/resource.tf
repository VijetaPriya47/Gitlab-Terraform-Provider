resource "gitlab_project" "my_project" {
  name             = "my_project"
  description      = "My project."
  visibility_level = "public"
}

resource "gitlab_project_integration_youtrack" "default" {
  project     = gitlab_project.my_project.id
  issues_url  = "https://my.youtrack.com/issue/:id"
  project_url = "https://my.youtrack.com"
}
