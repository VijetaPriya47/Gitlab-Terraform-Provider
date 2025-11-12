# Download a text artifact file from the latest successful pipeline
data "gitlab_artifact_file" "config" {
  project       = "namespace/myproject"
  job           = "build-job"
  ref           = "main"
  artifact_path = "config/settings.json"
}

# Use the artifact content
output "config_content" {
  value = data.gitlab_artifact_file.config.content
}

# Download a binary artifact file using base64 encoding
data "gitlab_artifact_file" "binary" {
  project       = "namespace/myproject"
  job           = "build-job"
  ref           = "v1.0.0"
  artifact_path = "dist/app.zip"
}

output "binary_content_base64" {
  value = data.gitlab_artifact_file.binary.content_base64
}

# Download artifact from a specific tag
data "gitlab_artifact_file" "release" {
  project       = "12345"
  job           = "release-job"
  ref           = "v2.1.0"
  artifact_path = "release-notes.txt"
}

# Download a larger artifact with custom size limit
data "gitlab_artifact_file" "large_artifact" {
  project        = "namespace/myproject"
  job            = "build-job"
  ref            = "main"
  artifact_path  = "dist/large-file.zip"
  max_size_bytes = 20971520 # 20MB
}
