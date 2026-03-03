resource "gitlab_group" "example_group" {
  name        = "example_group"
  path        = "example_group"
  description = "An example group"
}

resource "gitlab_group_integration_microsoft_teams" "teams" {
  group   = gitlab_group.example_group.id
  webhook = "https://outlook.office.com/webhook/..."

  notify_only_broken_pipelines = false
  branches_to_be_notified      = "all"
  push_events                  = true
  issues_events                = true
  confidential_issues_events   = true
  merge_requests_events        = true
  tag_push_events              = false
  note_events                  = false
  confidential_note_events     = false
  pipeline_events              = true
  wiki_page_events             = false
}
