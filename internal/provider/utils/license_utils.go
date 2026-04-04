package utils

import (
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

// Global variable to cache the result of EE evaluation
var isEE *bool

// function calls gitlab server metadata API to determine if
// license model is enterprise or not
func IsRunningInEEContext(client *gitlab.Client) (bool, error) {
	if isEE != nil {
		return *isEE, nil
	}

	// Defensive: ensure client is not nil
	if client == nil {
		return false, nil
	}

	metadata, _, err := client.Metadata.GetMetadata()
	if err != nil {
		return false, err
	}

	// Defensive: ensure metadata is not nil
	if metadata == nil {
		return false, nil
	}

	result := IsEnterpriseInstance(metadata)

	// Correctly assign to global cache (no shadowing)
	isEE = gitlab.Ptr(result)

	return result, nil
}

// function determine license model based on gitlab
// metadata details
func IsEnterpriseInstance(metadata *gitlab.Metadata) bool {
	// Defensive: prevent nil pointer dereference
	if metadata == nil {
		return false
	}

	if metadata.Enterprise {
		return true
	}

	// This is only to support 15.5. From 15.8 on, we can remove this code
	// as we won't be supporting 15.5 anymore.
	if strings.Contains(metadata.Version, "-ee") {
		return true
	}

	return false
}
