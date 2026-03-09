package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/api/client-go/v2"
)

// IsGitLabVersionLessThan is a SkipFunc that returns true if the provided version is lower then
// the current version of GitLab. It only checks the major and minor version numbers, not the patch.
func IsGitLabVersionLessThan(ctx context.Context, client *gitlab.Client, version string) func() (bool, error) {
	return func() (bool, error) {
		isAtLeast, err := IsGitLabVersionAtLeast(ctx, client, version)()
		return !isAtLeast, err
	}
}

// IsGitLabVersionAtLeast is a SkipFunc that checks that the version of GitLab is at least the
// provided wantVersion. It only checks the major and minor version numbers, not the patch.
func IsGitLabVersionAtLeast(ctx context.Context, client *gitlab.Client, wantVersion string) func() (bool, error) {
	return func() (bool, error) {
		wantMajor, wantMinor, err := parseVersionMajorMinor(wantVersion)
		if err != nil {
			return false, fmt.Errorf("failed to parse wanted version %q: %w", wantVersion, err)
		}

		actualVersion, _, err := client.Version.GetVersion(gitlab.WithContext(ctx))
		if err != nil {
			return false, err
		}

		actualMajor, actualMinor, err := parseVersionMajorMinor(actualVersion.Version)
		if err != nil {
			return false, fmt.Errorf("failed to parse actual version %q: %w", actualVersion.Version, err)
		}

		if actualMajor == wantMajor {
			return actualMinor >= wantMinor, nil
		}

		return actualMajor > wantMajor, nil
	}
}

func parseVersionMajorMinor(version string) (int, int, error) {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("need at least 2 parts (was %d)", len(parts))
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return major, minor, nil
}
