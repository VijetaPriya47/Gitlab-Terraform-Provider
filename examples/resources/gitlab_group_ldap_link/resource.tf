resource "gitlab_group_ldap_link" "test" {
  group         = "12345"
  cn            = "testuser"
  group_access  = "developer"
  ldap_provider = "ldapmain"
}
