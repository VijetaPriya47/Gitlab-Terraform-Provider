resource "gitlab_project_push_rules" "sample" {
  project                       = 42
  author_email_regex            = "@gitlab.com$"
  branch_name_regex             = "(feat|fix)\\/*"
  commit_committer_check        = true
  commit_committer_name_check   = true
  commit_message_negative_regex = "ssh\\:\\/\\/"
  commit_message_regex          = "(feat|fix):.*"
  deny_delete_tag               = false
  file_name_regex               = "(jar|exe)$"
  max_file_size                 = 4
  member_check                  = true
  prevent_secrets               = true
  reject_unsigned_commits       = false
}
