# Lookup by user ID
data "gitlab_user_sshkeys" "example" {
  user_id = 1
}

# Lookup by username
data "gitlab_user_sshkeys" "example" {
  username = "test.person"
}
