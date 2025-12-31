#!/usr/bin/env sh

CONTAINER_ENGINE="${CONTAINER_ENGINE:-docker}"

set -e

if [ "$CONTAINER_ENGINE" != "docker" ]; then
  echo "Using container engine $CONTAINER_ENGINE"
fi

printf 'Waiting for GitLab container to become healthy'

until test -n "$($CONTAINER_ENGINE ps --quiet --filter label=terraform-provider-gitlab/owned --filter health=healthy)"; do
  printf '.'
  sleep 5
done

echo
echo "GitLab is healthy at $GITLAB_BASE_URL"

# Print the version, since it is useful debugging information.
curl --silent --show-error --header "Authorization: Bearer $GITLAB_TOKEN" "$GITLAB_BASE_URL/version"
echo

# We use git imports during integration tests, so the import sources need to have git enabled as of 16.0. Otherwise they're all disabled.
# In addition to enabling git imports, we also need to disable enforcement of CI/CD job token scopes, since the resource tests depend on this setting being disabled by default
echo "Setting import sources to 'git' for testing purposes"
curl --silent --show-error --request PUT --header "Authorization: Bearer $GITLAB_TOKEN" "$GITLAB_BASE_URL/application/settings?import_sources=git,gitlab_project&enforce_ci_inbound_job_token_scope_enabled=false"