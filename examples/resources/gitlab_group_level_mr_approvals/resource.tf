resource "gitlab_group" "foo" {
  name        = "test_group"
  path        = "test_group"
  description = "An example group"
}

resource "gitlab_group_level_mr_approvals" "foo" {
  group                                              = gitlab_group.foo.id
  allow_author_approval                              = true
  allow_committer_approval                           = true
  allow_overrides_to_approver_list_per_merge_request = true
  retain_approvals_on_push                           = true
  selective_code_owner_removals                      = false
  require_reauthentication_to_approve                = true
}
