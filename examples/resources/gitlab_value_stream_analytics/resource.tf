resource "gitlab_value_stream_analytics" "project" {
  name              = "TEST"
  project_full_path = "test/project"
  stages = [
    {
      name   = "Issue"
      custom = false
      hidden = false
    },
    {
      name                   = "Issue Labels"
      custom                 = true
      hidden                 = false
      start_event_identifier = "ISSUE_LABEL_ADDED"
      start_event_label_id   = "gid://gitlab/ProjectLabel/0"
      end_event_identifier   = "ISSUE_LABEL_REMOVED"
      end_event_label_id     = "gid://gitlab/ProjectLabel/1"
    }
  ]
}

resource "gitlab_value_stream_analytics" "group" {
  name            = "TEST"
  group_full_path = "test/group"
  stages = [
    {
      name   = "Issue"
      custom = false
      hidden = false
    },
    {
      name                   = "Issue Labels"
      custom                 = true
      hidden                 = false
      start_event_identifier = "ISSUE_LABEL_ADDED"
      start_event_label_id   = "gid://gitlab/GroupLabel/0"
      end_event_identifier   = "ISSUE_LABEL_REMOVED"
      end_event_label_id     = "gid://gitlab/GroupLabel/1"
    }
  ]
}
