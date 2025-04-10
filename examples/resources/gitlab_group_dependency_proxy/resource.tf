resource "gitlab_group_dependency_proxy" "foo" {
  group = "1234"

  enabled  = true
  identity = "newidentity"
  secret   = "somesecret"
}
