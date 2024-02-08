package api

import (
	"github.com/xanzy/go-gitlab"
)

// NOTE:
// The access level story in the GitLab API is a bit tricky.
// There are different resources using the same access level names
// with an identical mapping to int ids. As also defined in the
// `gitlab.AccessLevelValue` types. However, different endpoints
// allow all of them or just a subset. There is also endpoints
// defining an additional `admin` access level, which is nowhere
// documented and probably not used at all - this provider ignores it.
// Point being, be careful when using them in a resource or data source
// and consult the upstream API docs to verify what's possible and keep
// your fingers crossed it's correct :)

// see the source of truth for `AccessLevelNameToValue` and `AccessLevelValueToName`
// here: https://docs.gitlab.com/ee/api/members.html#valid-access-levels
var ValidGroupAccessLevelNames = []string{
	"no one",
	"minimal",
	"guest",
	"reporter",
	"developer",
	"maintainer",
	"owner",

	// Deprecated and should be removed in v4 of this provider
	"master",
}
var ValidProjectAccessLevelNames = []string{
	"no one",
	"minimal",
	"guest",
	"reporter",
	"developer",
	"maintainer",
	"owner",

	// Deprecated and should be removed in v4 of this provider
	"master",
}

// NOTE(TF): the documentation here https://docs.gitlab.com/ee/api/protected_branches.html
//
//	mentions an `60 => Admin access` level, but it actually seems to not exist.
//	Ignoring here that I've every read about this ...
var ValidProtectedBranchTagAccessLevelNames = []string{
	"no one", "developer", "maintainer",
}

// The only access levels allowed to be configured to unprotect a protected branch
// The API states the others are either forbidden (via 403) or invalid
var ValidProtectedBranchUnprotectAccessLevelNames = []string{
	"developer", "maintainer", "admin",
}

var ValidProtectedEnvironmentDeploymentLevelNames = []string{
	"developer", "maintainer",
}

var ValidProjectEnvironmentStates = []string{
	"available", "stopped",
}

var ValidCIRestrictPipelineCancellationRoleValues = []string{
	"developer", "maintainer", "no one",
}

var AccessLevelNameToValue = map[string]gitlab.AccessLevelValue{
	"no one":     gitlab.NoPermissions,
	"minimal":    gitlab.MinimalAccessPermissions,
	"guest":      gitlab.GuestPermissions,
	"reporter":   gitlab.ReporterPermissions,
	"developer":  gitlab.DeveloperPermissions,
	"maintainer": gitlab.MaintainerPermissions,
	"owner":      gitlab.OwnerPermissions,
	"admin":      gitlab.AdminPermissions,

	// Deprecated and should be removed in v4 of this provider
	"master": gitlab.MaintainerPermissions,
}

var AccessLevelValueToName = map[gitlab.AccessLevelValue]string{
	gitlab.NoPermissions:            "no one",
	gitlab.MinimalAccessPermissions: "minimal",
	gitlab.GuestPermissions:         "guest",
	gitlab.ReporterPermissions:      "reporter",
	gitlab.DeveloperPermissions:     "developer",
	gitlab.MaintainerPermissions:    "maintainer",
	gitlab.OwnerPermissions:         "owner",
	gitlab.AdminPermissions:         "admin",
}
