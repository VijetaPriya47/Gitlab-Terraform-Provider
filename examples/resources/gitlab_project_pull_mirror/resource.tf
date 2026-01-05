# Basic pull mirror from GitHub
# Note: Unspecified options will use GitLab's defaults
resource "gitlab_project_pull_mirror" "github" {
  project       = gitlab_project.example.id
  url           = "https://github.com/example/repo.git"
  auth_user     = "github-username"
  auth_password = var.github_token
}

# Pull mirror with explicit options
# Only specify options you want to control; omit others to use GitLab defaults
resource "gitlab_project_pull_mirror" "advanced" {
  project                             = gitlab_project.example.id
  url                                 = "https://github.com/example/repo.git"
  enabled                             = true
  auth_user                           = "github-username"
  auth_password                       = var.github_token
  mirror_trigger_builds               = true
  only_mirror_protected_branches      = true
  mirror_overwrites_diverged_branches = false
}

# Pull mirror with branch regex (Premium/Ultimate)
resource "gitlab_project_pull_mirror" "regex" {
  project             = gitlab_project.example.id
  url                 = "https://github.com/example/repo.git"
  auth_user           = "github-username"
  auth_password       = var.github_token
  mirror_branch_regex = "^(main|develop|release/.*)$"
}

# Minimal configuration - GitLab will apply all defaults
resource "gitlab_project_pull_mirror" "minimal" {
  project       = gitlab_project.example.id
  url           = "https://github.com/example/repo.git"
  auth_user     = "github-username"
  auth_password = var.github_token
  # enabled, mirror_trigger_builds, etc. will use GitLab's defaults
  # and be visible in terraform state after apply
}
