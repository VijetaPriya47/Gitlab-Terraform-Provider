# The 2 data items below are useful to grab information about already provisioned resources.

data "gitlab_user" "team_lead" {
  username = var.team_lead_user
}

data "gitlab_user" "team_members" {
  for_each = toset(var.team_member_users)
  username = each.value
}

# With this, we are able to create a team specific group under a top-level "root" group and assign team members to various access levels.

resource "gitlab_group" "my_team" {
  parent_id   = var.root_group_id
  name        = "Awesome Tech"
  path        = "awesome-tech"
  description = "We provide awesome tech that makes our company shine!"
}

resource "gitlab_group_membership" "team_lead" {
  group_id     = gitlab_group.my_team.id
  access_level = "maintainer"
  user_id      = data.gitlab_user.team_lead.id
}

resource "gitlab_group_membership" "team_members" {
  for_each     = data.gitlab_user.team_members
  group_id     = gitlab_group.my_team.id
  access_level = "developer"
  user_id      = each.value.id
}

# Here we have the project, for the Team Wiki. As Wikis don't follow a typical MR flow, we aren't adding any approval rules here.

resource "gitlab_project" "team_wiki" {
  namespace_id = gitlab_group.my_team.id
  name         = "Team Wiki"
  path         = "wiki"
  description  = "Here is where we can knowledge share about our product."
}

# And here we have the fullstack app project. This will follow a typical GitLab flow, so we have approval rules as well as the definition for a GitLab runner that can be used for CI/CD automation jobs. FYI: the approval rules require GitLab premium or ultimate.

resource "gitlab_project" "app" {
  namespace_id = gitlab_group.my_team.id
  name         = "Fullstack App"
  path         = "app"
  description  = "Our fullstack app which will deliver value fast!"
}

resource "gitlab_project_approval_rule" "team_wiki_maintainers" {
  count              = var.license_type != "free" ? 1 : 0
  project            = gitlab_project.app.id
  name               = "maintainers"
  approvals_required = 1
  user_ids           = [data.gitlab_user.team_lead.id]
}

resource "gitlab_project_approval_rule" "team_wiki_members" {
  count              = var.license_type != "free" ? 1 : 0
  project            = gitlab_project.app.id
  name               = "members"
  approvals_required = 1
  user_ids           = [for user in data.gitlab_user.team_members : user.id]
}

resource "gitlab_user_runner" "linux" {
  project_id  = gitlab_project.app.id
  description = "Team Linux Job Runner"
  runner_type = "project_type"
  untagged    = true # you can use `tag_list` instead if you want user's to opt-in to using this runner.
}
