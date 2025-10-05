#!/usr/bin/env sh

# This script is intended to be used as a Docker HEALTHCHECK for the GitLab container.
# It prepares GitLab prior to running acceptance tests.
#
# This is a known workaround for docker-compose lacking lifecycle hooks.
# See: https://github.com/docker/compose/issues/1809#issuecomment-657815188

set -e

# Check for a successful HTTP status code from GitLab.
curl --silent --show-error --fail --output /dev/null 127.0.0.1:80

# Because this script runs on a regular health check interval,
# this file functions as a marker that tells us if initialization already finished.
done=/var/gitlab-acctest-initialized

test -f $done || {
  echo 'Initializing GitLab for acceptance tests'

  echo 'Creating access token'
  gitlab-rails console <<EOF
terraform_token = PersonalAccessToken.create(
  user_id: 1,
  scopes: [:api, :read_user],
  name: :terraform,
  expires_at: Time.now + 30.days
)
terraform_token.set_token('$GITLAB_TOKEN')
terraform_token.save!
EOF

  # 2020-09-07: Currently Gitlab (version 13.3.6 ) doesn't allow in admin API
  # ability to set a group as instance level templates.
  # To test resource_gitlab_project_test template features we add
  # group and admin settings directly in scripts/start-gitlab.sh
  # Once Gitlab add admin template in API we could manage group/settings
  # directly in tests like TestAccGitlabProject_basic.
  # Works on CE too

  echo 'Creating an instance level template group with a simple template based on rails'
  gitlab-rails console <<EOF
group_template = Group.new(
  name: :terraform,
  path: :terraform
)
group_template.save!
application_settings = ApplicationSetting.find_by ""
application_settings.custom_project_templates_group_id = group_template.id
application_settings.save!
EOF

  echo 'Enabling `retain_resource_access_token_user_after_revoke` feature flag'
  gitlab-rails console <<EOF
Feature.enable(:retain_resource_access_token_user_after_revoke)
EOF

  touch $done
}

echo 'GitLab is ready for acceptance tests'
