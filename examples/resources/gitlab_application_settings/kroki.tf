resource "gitlab_application_settings" "this" {
  kroki_enabled = true
  kroki_url     = "https://kroki.io"

  kroki_formats {
    bpmn       = true
    blockdiag  = false
    excalidraw = true
  }
}