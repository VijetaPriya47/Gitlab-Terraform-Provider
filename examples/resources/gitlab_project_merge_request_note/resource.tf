resource "gitlab_project" "example" {
  name             = "example project"
  description      = "Lorem Ipsum"
  visibility_level = "public"
}

resource "gitlab_project_merge_request_note" "example" {
  project           = gitlab_project.example.id
  merge_request_iid = 456
  body              = "Example note"
}
