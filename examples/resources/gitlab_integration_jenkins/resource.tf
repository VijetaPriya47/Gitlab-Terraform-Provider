resource "gitlab_project" "awesome_project" {
  name             = "awesome_project"
  description      = "My awesome project."
  visibility_level = "public"
}

resource "gitlab_integration_jenkins" "jenkins" {
  project      = gitlab_project.awesome_project.id
  jenkins_url  = "http://jenkins.example.com"
  project_name = "my_project_name"
}
