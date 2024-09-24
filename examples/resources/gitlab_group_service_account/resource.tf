resource "gitlab_group" "example" {
  name        = "example"
  path        = "example"
  description = "An example group"
}

resource "gitlab_group_service_account" "example-sa" {
  group    = gitlab_group.example.id
  name     = "example-name"
  username = "example-username"
}
