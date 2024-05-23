#!/usr/bin/env bash

set -e

VERSION="$1"
if [ -z "$VERSION" ]; then
  echo "Error: Please provide the new version to generate the changelog for as first argument." >&2
  exit 1
fi

# Regex from https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
SEMANTIC_VERSION_REGEX='^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$'

match=$(echo "$VERSION" | perl -ne "print if /$SEMANTIC_VERSION_REGEX/")
if [ -z "$match" ]; then
  echo "Error: The given version doesn't match the semantic versioning format." >&2
  exit 1
fi

default_branch="main"

head="$(git rev-parse --abbrev-ref HEAD)"
if [ "$head" != "$default_branch" ]; then
  echo "Error: Please checkout the $default_branch branch first: git switch $default_branch" >&2
  exit 1
fi

if [ -z "$GITLAB_TOKEN" ]; then
  echo "Error: Please set the GITLAB_TOKEN environment variable" >&2
  exit 1
fi

release_branch="releases/$VERSION"
echo -n "Creating release branch $release_branch ..."
curl \
  --fail-with-body \
  --silent \
  --request POST \
  --header "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  'https://gitlab.com/api/v4/projects/gitlab-org%2Fterraform-provider-gitlab/repository/branches' \
  --data "branch=$release_branch&ref=$default_branch" | true
echo " OK"

echo -n "Creating changelog in $release_branch ... "
curl \
  --fail-with-body \
  --silent \
  --request POST \
  --header "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  'https://gitlab.com/api/v4/projects/gitlab-org%2Fterraform-provider-gitlab/repository/changelog' \
  --data "version=$VERSION&branch=$release_branch&message=Add changelog for $VERSION" > /dev/null
echo " OK"

echo -n "Creating merge request ... "
output=$(curl \
  --silent \
  --request POST \
  --header "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  --header "Content-Type: application/json" \
  'https://gitlab.com/api/v4/projects/gitlab-org%2Fterraform-provider-gitlab/merge_requests' \
  --data '{
    "source_branch": "'$release_branch'",
    "target_branch": "'$default_branch'",
    "title": "Release '$VERSION'",
    "description": "Please review the automatically generated changelog.\n/assign @timofurrer @patrickricee\n/assign_reviewer @timofurrer @patrickrice",
    "labels": "group::environments, maintenance::release"
  }'
)

mr_id=$(echo "$output" | jq -r ".id")
if [ "$mr_id" != "null" ]; then
  echo " OK"

  echo "Success: merge request created at: $(echo "$output" | jq -r ".web_url")"
  exit 0
fi

echo "$output" | grep -q "Another open merge request already exists"
if [ "$?" -eq 0 ]; then
  echo " OK"

  echo "Success: merge request already exists"
  exit 0
fi

echo "Error: something went wrong when trying to create the merge request:" >&2
echo "$output" | jq >&2
exit 1
