# CE example
resource "gitlab_branch_protection" "ce_branch" {
  project            = "12345"
  branch             = "BranchProtected"
  push_access_level  = "developer"
  merge_access_level = "developer"
  allow_force_push   = true
}

# EE example
resource "gitlab_branch_protection" "ee_branch" {
  project                      = "12345"
  branch                       = "BranchProtected"
  allow_force_push             = true
  code_owner_approval_required = true

  allowed_to_push = [
    {
      user_id = 5
    },
    {
      user_id = 521
    },
    {
      access_level = "no one"
    }
  ]

  allowed_to_merge = [
    {
      user_id = 15
    },
    {
      user_id = 37
    },
    {
      access_level = "maintainer"
    }
  ]

  allowed_to_unprotect = [
    {
      user_id = 15
    },
    {
      group_id = 42
    },
    {
      access_level = "maintainer"
    }
  ]
}

# EE example with admin push access level
resource "gitlab_branch_protection" "admin_push" {
  project = "12345"
  branch  = "admin-protected"

  allowed_to_push = [
    {
      access_level = "admin"
    }
  ]

  allowed_to_merge = [
    {
      access_level = "maintainer"
    }
  ]

  allowed_to_unprotect = [
    {
      access_level = "maintainer"
    }
  ]
}
