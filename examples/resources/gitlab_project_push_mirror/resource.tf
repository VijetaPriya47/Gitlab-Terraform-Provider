resource "gitlab_project_push_mirror" "foo" {
  project = "1"
  url     = "https://username:password@github.com/org/repository.git"
}
