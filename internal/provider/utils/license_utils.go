package utils

import (
	"github.com/xanzy/go-gitlab"
	"strings"
)

// Global variable to cache the result of EE evaluation
var isEE *bool

// function calls gitlab server metadata API to determine if
// license model is enterprise or not
func IsRunningInEEContext(client *gitlab.Client) (bool, error) {
	if isEE != nil {
		return *isEE, nil
	}
	metadata, _, err := client.Metadata.GetMetadata()
	if err != nil {
		return false, err
	}
	isEE := gitlab.Ptr(IsEnterpriseInstance(metadata))
	return *isEE, err
}

// function determine license model base on gitlab
// metadata details
func IsEnterpriseInstance(metadata *gitlab.Metadata) bool {
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
