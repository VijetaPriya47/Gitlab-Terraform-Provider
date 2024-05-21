# Basic example
resource "gitlab_group_saml_link" "test" {
  group           = "12345"
  access_level    = "developer"
  saml_group_name = "samlgroupname1"
}

# Example using a Custom Role (Ultimate only)
resource "gitlab_group_saml_link" "test_custom_role" {
  group           = "12345"
  access_level    = "developer"
  saml_group_name = "samlgroupname1"
  member_role_id  = 123
}