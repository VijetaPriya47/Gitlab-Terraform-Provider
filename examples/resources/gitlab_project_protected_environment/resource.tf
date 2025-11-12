resource "gitlab_project_environment" "this" {
  project      = 123
  name         = "example"
  external_url = "www.example.com"
}

# Example with deployment access level
resource "gitlab_project_protected_environment" "example_with_access_level" {
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

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
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

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
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

  deploy_access_levels {
    access_level = "developer"
  }

  approval_rules = [
    {
      access_level       = "developer"
      required_approvals = 2
    }
  ]
}

# Example with multiple approval rules, using access level, user, and group
resource "gitlab_project_protected_environment" "example_with_multiple" {
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

  deploy_access_levels {
    access_level = "developer"
  }

  approval_rules = [
    {
      user_id = 789
    },
    {
      access_level       = "developer"
      required_approvals = 2
    },
    {
      group_id = 456
    }
  ]
}

# Example with deployment access level attribute
resource "gitlab_project_protected_environment" "example_access_levels_attribute" {
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

  deploy_access_levels_attribute = [
    {
      access_level = "developer"
    }
  ]
}

# Example with group-based deployment level attribute
resource "gitlab_project_protected_environment" "example_access_levels_attribute_with_group" {
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

  deploy_access_levels_attribute = [
    {
      group_id = 456
    }
  ]
}

# Example with user-based deployment level attribute
resource "gitlab_project_protected_environment" "example_access_levels_attribute_with_user" {
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

  deploy_access_levels_attribute = [
    {
      user_id = 789
    }
  ]
}

# Example with multiple deployment access levels attribute
resource "gitlab_project_protected_environment" "example_access_levels_attribute_with_multiple" {
  project     = gitlab_project_environment.this.project
  environment = gitlab_project_environment.this.name

  deploy_access_levels_attribute = [
    {
      access_level = "developer"
    },
    {
      group_id = 456
    },
    {
      user_id = 789
    }
  ]
}