resource "gitlab_project" "example" {
  name        = "example"
  description = "My awesome codebase"

  visibility_level = "public"
}

# Project with custom push rules
resource "gitlab_project" "example-two" {
  name = "example-two"

  push_rules {
    author_email_regex     = "@example\\.com$"
    commit_committer_check = true
    member_check           = true
    prevent_secrets        = true
  }
}

# Create a project for a given user (requires admin access)
data "gitlab_user" "peter_parker" {
  username = "peter_parker"
}

resource "gitlab_project" "peters_repo" {
  name         = "peters-repo"
  description  = "This is a description"
  namespace_id = data.gitlab_user.peter_parker.namespace_id
}

# Fork a project
resource "gitlab_project" "fork" {
  name                   = "my-fork"
  description            = "This is a fork"
  forked_from_project_id = gitlab_project.example.id
}

# Fork a project and setup a pull mirror (Deprecated, use gitlab_project_pull_mirror instead)
resource "gitlab_project" "fork" {
  name                   = "my-fork"
  description            = "This is a fork"
  forked_from_project_id = gitlab_project.example.id
  import_url             = gitlab_project.example.http_url_to_repo
  mirror                 = true
}

# Create a project by importing it from a public project
resource "gitlab_project" "import_public" {
  name       = "import-from-public-project"
  import_url = "https://gitlab.example.com/repo.git"
}

# Create a project by importing it from a public project and setup the pull mirror (Deprecated, use gitlab_project_pull_mirror instead)
resource "gitlab_project" "import_public_with_mirror" {
  name       = "import-from-public-project"
  import_url = "https://gitlab.example.com/repo.git"
  mirror     = true
}

# Create a project by importing it from a private project
resource "gitlab_project" "import_private" {
  name                = "import-from-public-project"
  import_url          = "https://gitlab.example.com/repo.git"
  import_url_username = "user"
  import_url_password = "pass"
}

# Create a project by importing it from a private project and setup the pull mirror (Deprecated, use gitlab_project_pull_mirror instead)
resource "gitlab_project" "import_private_with_mirror" {
  name                = "import-from-public-project"
  import_url          = "https://gitlab.example.com/repo.git"
  import_url_username = "user"
  import_url_password = "pass"
  mirror              = true
}

# Create a project by importing it from a private project and provide credentials in `import_url`
# NOTE: only use this if you really must, use `import_url_username` and `import_url_password` whenever possible
#       GitLab API will always return the `import_url` without credentials, therefore you must ignore the `import_url` for changes:
resource "gitlab_project" "import_private" {
  name       = "import-from-public-project"
  import_url = "https://user:pass@gitlab.example.com/repo.git"

  lifecycle {
    ignore_changes = [
      import_url
    ]
  }
}
