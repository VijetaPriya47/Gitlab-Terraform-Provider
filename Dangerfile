require 'gitlab-dangerfiles'

# see https://docs.gitlab.com/development/dangerbot/#enable-danger-on-a-project
# see https://gitlab.com/gitlab-org/ruby/gems/gitlab-dangerfiles
Gitlab::Dangerfiles.for_project(self) do |dangerfiles|
  # Import all plugins from the gem
  dangerfiles.import_plugins

  # Or import only a subset of rules
  dangerfiles.import_dangerfiles(except: %w[changelog])
end