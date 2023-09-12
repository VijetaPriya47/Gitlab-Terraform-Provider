resource "gitlab_project_environment" "this" {
  project      = 123
  name         = "example"
  external_url = "www.example.com"
}

# Example with deployment access level
resource "gitlab_project_protected_environment" "example_with_access_level" {
  project                 = gitlab_project_environment.this.project
  required_approval_count = 1
  environment             = gitlab_project_environment.this.name

  deploy_access_levels {
    access_level = "developer"
  }
}

# Example with group-based deployment level
resource "gitlab_project_protected_environment" "example_with_group" {
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

  deploy_access_levels {
    group_id = 456
  }
}

# Example with user-based deployment level
resource "gitlab_project_protected_environment" "example_with_user" {
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

  deploy_access_levels {
    user_id = 789
  }
}

# Example with multiple deployment access levels
resource "gitlab_project_protected_environment" "example_with_multiple" {
  project                 = gitlab_project_environment.this.project
  required_approval_count = 2
  environment             = gitlab_project_environment.this.name

  deploy_access_levels {
    access_level = "developer"
  }

  deploy_access_levels {
    group_id = 456
  }

  deploy_access_levels {
    user_id = 789
  }
}

# Example with access-level based approval rules
resource "gitlab_project_protected_environment" "example_with_multiple" {
  project                 = gitlab_project_environment.this.project
  required_approval_count = 2
  environment             = gitlab_project_environment.this.name

  deploy_access_levels {
    access_level = "developer"
  }

  approval_rules = [
    {
      access_level = "developer"
    }
  ]
}

# Example with multiple approval rules, using access level, user, and group
resource "gitlab_project_protected_environment" "example_with_multiple" {
  project                 = gitlab_project_environment.this.project
  required_approval_count = 2
  environment             = gitlab_project_environment.this.name

  deploy_access_levels {
    access_level = "developer"
  }

  approval_rules = [
    {
      user_id = 789
    },
    {
      access_level = "developer"
    },
    {
      group_id = 456
    }
  ]
}



