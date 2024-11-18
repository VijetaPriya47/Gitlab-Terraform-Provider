data "gitlab_group_billable_member_memberships" "test_user_membership" {
  user_id  = 21
  group_id = 42
}
