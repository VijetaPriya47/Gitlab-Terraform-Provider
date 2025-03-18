resource "gitlab_user" "example" {
  name             = "Example Foo"
  username         = "example"
  email            = "gitlab@user.create"
  is_admin         = true
  projects_limit   = 4
  can_create_group = false
  is_external      = true
}

resource "gitlab_user_identity" "example" {
  user_id           = gitlab_user.example.id
  external_provider = "google"
  external_uid      = "1234567890"
}