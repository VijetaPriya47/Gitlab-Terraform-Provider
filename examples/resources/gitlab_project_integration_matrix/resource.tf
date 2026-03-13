resource "gitlab_project" "awesome_project" {
  name             = "awesome_project"
  description      = "My awesome project."
  visibility_level = "public"
}

resource "gitlab_project_integration_matrix" "matrix" {
  project  = gitlab_project.awesome_project.id
  hostname = "https://matrix.org"
  token    = "your-matrix-token"
  room     = "!abcdefg:matrix.org"
}
