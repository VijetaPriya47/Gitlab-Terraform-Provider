# Example with deployment access level
resource "gitlab_group_protected_environment" "example_with_access_level" {
  group                   = 12345
  required_approval_count = 1
  environment             = "production"

  deploy_access_levels = [
    {
      access_level = "developer"
    }
  ]
}

# Example with group-based deployment level
resource "gitlab_group_protected_environment" "example_with_group" {
  group       = 12345
  environment = "staging"

  deploy_access_levels = [
    {
      group_id = 456
    }
  ]
}

# Example with user-based deployment level
resource "gitlab_group_protected_environment" "example_with_user" {
  group       = 12345
  environment = "other"

  deploy_access_levels = [
    {
      user_id = 789
    }
  ]
}

# Example with multiple deployment access levels
resource "gitlab_group_protected_environment" "example_with_multiple" {
  group                   = 12345
  required_approval_count = 2
  environment             = "development"

  deploy_access_levels = [
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

# Example with access-level based approval rules
resource "gitlab_group_protected_environment" "example_with_multiple" {
  group                   = 12345
  required_approval_count = 2
  environment             = "testing"

  deploy_access_levels = [
    {
      access_level = "developer"
    }
  ]

  approval_rules = [
    {
      access_level = "developer"
    }
  ]
}

# Example with multiple approval rules, using access level, user, and group
resource "gitlab_group_protected_environment" "example_with_multiple" {
  group                   = 12345
  required_approval_count = 2
  environment             = "production"

  deploy_access_levels = [
    {
      access_level = "developer"
    }
  ]

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
