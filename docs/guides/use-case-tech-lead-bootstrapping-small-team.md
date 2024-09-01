---
page_title: "A Tech Lead Bootstrapping a Small Team"
subcategory: "Use Cases"
---

Imagine you are a tech lead, responsible for a small team, and you want to get your team bootstrapped with their own group with a Wiki and a couple of projects. One to hold your full-stack application that you've been working on, and another for a user facing documentation website. Within this role, you want to make sure that code quality is verified by yourself and at least one additional team member. You also want to setup your own GitLab CI runner to run your automation jobs for your full-stack mono-repo as well as your documentation website.

You've done some research and have heard that IaC (Infrastructure as Code) is all the rage and have also found out that Terraform and OpenTofu are a great technology to use for cloud resources. Further to this, you have also found out that GitLab, your SDLC tool of choice, has its own Terraform provider that will enable you to realize all of your IaC dreams! You still have a problem though, how in the world can I take advantage of this?

Have no fear! This guide will walk you through the process of starting from a fresh installation of Terraform or OpenTofu and building our your infrastructure as code solution.

Let's start by creating a new directory on your computer that will store your IaC code. From that directory, let's create a file named `main.tf` using your text editor of choice. Within this file, let's insert the following lines of code:

```terraform
terraform {
  required_providers {
    gitlab = {
      source = "gitlabhq/gitlab"
    }
  }
}

variable "admin_token" {
  description = "Owner/Maintainer PAT token with the api scope applied."
  type        = string
}

provider "gitlab" {
  token    = var.admin_token
  base_url = "https://gitlab.com" # change this if you are on a self-hosted GitLab instance.
}
```

These are the building blocks for using Terraform to manage your GitLab instance. The `terraform` block lets Terraform know where to download the provider for all GitLab resources, and the `provider` block configures the provider to use an externally provided personal access token to authenticate with GitLab when performing any configuration.

The next thing that we will want to do is create a GitLab group for your team's code and for your Wiki to live. Groups are a wonderful feature in GitLab that allows you to supply a multi-level hierarchy to your code assets. A root, or top-level group, is typically something that an organization will create so they have policy level controls over all sub-groups and projects found within them, so it is considered a best practice to limit the amount of top-level groups, and to focus on sub-dividing into team or functional areas groupings underneath. To facilitate this, let's create a group for the team by adding this to our `main.tf` file, modifying it to fit your needs:

```terraform
resource "gitlab_group" "my_team" {
  parent_id         = 1337           # change to your top-level group ID number
  name              = "Awesome Tech" # friendly group name
  path              = "awesome-tech" # path that will be a part of clone URIs
  name              = "Awesome Team" # friendly group name
  path              = "awesome-team" # path that will be a part of clone URIs
  description       = "The Awesome Team provides awesome tech that makes our company shine!"
  wiki_access_level = "private" # make the Wiki only viewable by group members
}
```

Now that we have a group, it is quite important to add team members to it. GitLab provides various access levels that you can apply to team members that will give them differing levels of access to the group and any resources defined within it. As the team lead, we'll give you _maintainer_ access, and for your team members, we'll give them _developer_ access. Now it might be a pain for you to figure out the internal ID number for each team member, so we'll take advantage of a data source so we can supply their user handles instead of their ID numbers. Add this code to the bottom of your `main.tf`, modifying it to fit your needs:

```terraform
data "gitlab_user" "team_lead" {
  username = "Delaney"
}

data "gitlab_user" "team_members" {
  for_each = toset(["Sasha", "Priyanka", "Simone"])
  username = each.value
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
```

Because of how membership rights work in GitLab, any projects we create under this group will give these users the level of access defined at the group level. So, let's create one project where you will house your kicking fullstack app, and another one which will hold your user facing documentation. Add this code to the bottom of your `main.tf`, modifying it to fit your needs:

```terraform
resource "gitlab_project" "app" {
  namespace_id = gitlab_group.my_team.id
  name         = "Fullstack App"
  path         = "app"
  description  = "Our fullstack app which will deliver value fast!"
  wiki_enabled = false
}

resource "gitlab_project" "docs" {
  namespace_id = gitlab_group.my_team.id
  name         = "Documentation"
  path         = "docs"
  description  = "User facing documentation website."
  wiki_enabled = false
}
```

Now, let's add some approval rules. These will force any MRs created on these projects to be approved by you the tech lead, and at least one other team member, before they can be merged. This does require at least a GitLab premium license on your top-level group, so if you don't have that, you can skip this part. Add this code to the bottom of your `main.tf`, modifying it to fit your needs:

```terraform
resource "gitlab_project_approval_rule" "team_app_maintainers" {
  project            = gitlab_project.app.id
  name               = "maintainers"
  approvals_required = 1
  user_ids           = [data.gitlab_user.team_lead.id]
}

resource "gitlab_project_approval_rule" "team_app_members" {
  project            = gitlab_project.app.id
  name               = "members"
  approvals_required = 1
  user_ids           = [for user in data.gitlab_user.team_members : user.id]
}

resource "gitlab_project_approval_rule" "team_docs_maintainers" {
  project            = gitlab_project.docs.id
  name               = "maintainers"
  approvals_required = 1
  user_ids           = [data.gitlab_user.team_lead.id]
}

resource "gitlab_project_approval_rule" "team_docs_members" {
  project            = gitlab_project.docs.id
  name               = "members"
  approvals_required = 1
  user_ids           = [for user in data.gitlab_user.team_members : user.id]
}
```

With this in place, you have one item left. You want to be able to automatically run tests, build your product, package it and ship it to customers. For that you are going to need a GitLab runner! With the current runner registration workflow, there is a requirement to create a runner instance on your GitLab group or project in order to configure basic settings as well as to get a registration token that you can utilize with your deployed runners. We are going to create a group runner so that it can be shared with your fullstack application and user documentation projects. Add this code to the bottom of your `main.tf`, modifying it to fit your needs:

```terraform
resource "gitlab_user_runner" "linux" {
  group_id    = gitlab_group.my_team.id
  description = "Team Linux Job Runner"
  runner_type = "group_type"
  untagged    = true # you can use `tag_list` instead if you want user's to opt-in to using this runner.
}

output "registration_token" {
  description = "Registration token to to use with your runner installation."
  value       = gitlab_user_runner.linux.token
  sensitive   = true
}
```

With all of this configuration in place, you should be ready to rock. You can use the following commands to initialize your Terraform root module, review the changes, apply them, and then retrieve the registration token that you will want to use with your runner installation:

```shell
terraform init
terraform plan -out plan.out
terraform apply plan.out
terraform output registration_token
```
