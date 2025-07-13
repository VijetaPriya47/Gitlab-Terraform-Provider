resource "gitlab_group" "foo" {
  name = "foo-group"
  path = "foo-path"
}

resource "gitlab_group_badge" "example" {
  group     = gitlab_group.foo.id
  link_url  = "https://example.com/badge-123"
  image_url = "https://example.com/badge-123.svg"
}

# Pipeline status badges with placeholders will be enabled for each project
resource "gitlab_group_badge" "gitlab_pipeline" {
  group     = gitlab_group.foo.id
  link_url  = "https://gitlab.example.com/%%{project_path}/-/pipelines?ref=%%{default_branch}"
  image_url = "https://gitlab.example.com/%%{project_path}/badges/%%{default_branch}/pipeline.svg"
}

# Test coverage report badges with placeholders will be enabled for each project
resource "gitlab_group_badge" "gitlab_coverage" {
  group     = gitlab_group.foo.id
  link_url  = "https://gitlab.example.com/%%{project_path}/-/jobs"
  image_url = "https://gitlab.example.com/%%{project_path}/badges/%%{default_branch}/coverage.svg"
}

# Latest release badges with placeholders will be enabled for each project
resource "gitlab_group_badge" "gitlab_release" {
  group     = gitlab_group.foo.id
  link_url  = "https://gitlab.example.com/%%{project_path}/-/releases"
  image_url = "https://gitlab.example.com/%%{project_path}/-/badges/release.svg"
}
