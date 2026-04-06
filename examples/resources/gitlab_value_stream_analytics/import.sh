# Gitlab value stream analytics can be imported with a key composed of `<full_path_type>:<full_path>:<value_stream_id>`, for example:
terraform import gitlab_value_stream_analytics.group "group:people/engineers:42"

terraform import gitlab_value_stream_analytics.project "project:projects/sample:43"
