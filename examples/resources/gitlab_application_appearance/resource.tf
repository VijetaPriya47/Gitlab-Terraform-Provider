# Example where the resource will not update the appearance on destroy
resource "gitlab_application_appearance" "this" {
  title              = "My GitLab"
  description        = "My GitLab instance"
  header_message     = "This is a header message"
  message_font_color = "#FFFF00"
}

# Example where the resource will revert the appearance to its original values on destroy
resource "gitlab_application_appearance" "this" {
  title                    = "My GitLab"
  keep_settings_on_destroy = false
  description              = "My GitLab instance"
  header_message           = "This is a header message"
  message_font_color       = "#FFFF00"
}
