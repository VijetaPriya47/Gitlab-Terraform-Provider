resource "gitlab_project" "awesome_project" {
  name             = "awesome_project"
  description      = "My awesome project."
  visibility_level = "public"
}

resource "gitlab_integration_harbor" "harbor" {
  project      = gitlab_project.awesome_project.id
  url          = "http://harbor.example.com"
  project_name = "my_project_name"
}
