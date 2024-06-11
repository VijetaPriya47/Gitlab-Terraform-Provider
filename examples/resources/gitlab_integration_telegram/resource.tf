resource "gitlab_project" "awesome_project" {
  name             = "awesome_project"
  description      = "My awesome project."
  visibility_level = "public"
}

resource "gitlab_integration_telegram" "default" {
  token = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
  room  = "-1000000000000000"

  notify_only_broken_pipelines = false
  branches_to_be_notified      = "all"
  push_events                  = false
  issues_events                = false
  confidential_issues_events   = false
  merge_requests_events        = false
  tag_push_events              = false
  note_events                  = false
  confidential_note_events     = false
  pipeline_events              = false
  wiki_page_events             = false
}
