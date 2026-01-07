# Create a project
resource "gitlab_project" "example" {
  name             = "example"
  description      = "An example project"
  visibility_level = "public"
}

# Create source issue
resource "gitlab_project_issue" "source" {
  project = gitlab_project.example.id
  title   = "Source Issue"
}

# Create target issue
resource "gitlab_project_issue" "target" {
  project = gitlab_project.example.id
  title   = "Target Issue"
}

# Link issues with "relates_to" link type (default)
resource "gitlab_project_issue_link" "relates_to" {
  project           = gitlab_project.example.id
  issue_iid         = gitlab_project_issue.source.iid
  target_project_id = gitlab_project.example.id
  target_issue_iid  = gitlab_project_issue.target.iid
  link_type         = "relates_to"
}


# Cross-project linking example
resource "gitlab_project" "other_project" {
  name        = "other_project"
  description = "Another project for cross-project linking"
}

resource "gitlab_project_issue" "cross_target" {
  project = gitlab_project.other_project.id
  title   = "Cross-Project Target Issue"
}

resource "gitlab_project_issue_link" "cross_project_relates_to" {
  project           = gitlab_project.example.id
  issue_iid         = gitlab_project_issue.source.iid
  target_project_id = gitlab_project.other_project.id
  target_issue_iid  = gitlab_project_issue.cross_target.iid
  link_type         = "relates_to"
}

