resource "gitlab_group" "example" {
  name        = "example"
  path        = "example"
  description = "An example group"
}

# Create a project in the example group
resource "gitlab_project" "example" {
  name         = "example"
  description  = "An example project"
  namespace_id = gitlab_group.example.id
}

# Group with custom push rules
resource "gitlab_group" "example-two" {
  name        = "example-two"
  path        = "example-two"
  description = "An example group with push rules"

  push_rules {
    author_email_regex     = "@example\\.com$"
    commit_committer_check = true
    member_check           = true
    prevent_secrets        = true
  }
}
