package api

import (
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
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
// here: https://docs.gitlab.com/user/permissions/#default-roles
var ValidGroupAccessLevelNames = []string{
	"no one",
	"minimal",
	"guest",
	"planner",
	"reporter",
	"developer",
	"maintainer",
	"owner",
}

var ValidGroupSAMLLinkAccessLevelNames = []string{
	"guest",
	"planner",
	"reporter",
	"developer",
	"maintainer",
	"owner",
}

var ValidProjectAccessLevelNames = []string{
	"no one",
	"minimal",
	"guest",
	"planner",
	"reporter",
	"developer",
	"maintainer",
	"owner",
}

var ValidProtectedBranchTagAccessLevelNames = []string{
	"no one", "developer", "maintainer", "admin",
}

// The only access levels allowed to be configured to unprotect a protected branch
// The API states the others are either forbidden (via 403) or invalid
var ValidProtectedBranchUnprotectAccessLevelNames = []string{
	"developer", "maintainer", "admin",
}

var ValidProtectedContainerRepositoryAccessLevelNames = []string{
	"maintainer", "owner", "admin",
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

var ValidCIPipelineVariablesMinimumOverrideRoleValues = []string{
	"developer", "maintainer", "owner", "no_one_allowed",
}

var AccessLevelNameToValue = map[string]gitlab.AccessLevelValue{
	"no one":     gitlab.NoPermissions,
	"minimal":    gitlab.MinimalAccessPermissions,
	"guest":      gitlab.GuestPermissions,
	"planner":    gitlab.PlannerPermissions,
	"reporter":   gitlab.ReporterPermissions,
	"developer":  gitlab.DeveloperPermissions,
	"maintainer": gitlab.MaintainerPermissions,
	"owner":      gitlab.OwnerPermissions,
	"admin":      gitlab.AdminPermissions,
}

var AccessLevelValueToName = map[gitlab.AccessLevelValue]string{
	gitlab.NoPermissions:            "no one",
	gitlab.MinimalAccessPermissions: "minimal",
	gitlab.GuestPermissions:         "guest",
	gitlab.PlannerPermissions:       "planner",
	gitlab.ReporterPermissions:      "reporter",
	gitlab.DeveloperPermissions:     "developer",
	gitlab.MaintainerPermissions:    "maintainer",
	gitlab.OwnerPermissions:         "owner",
	gitlab.AdminPermissions:         "admin",
}

// This function is required because the CIRestrict setting using an
// AccessControlLevel instead of an AccessLevelName, so it can't use the
// constants within client-go
func AccessControlLevelValueToName(input string) gitlab.AccessControlValue {
	var developer gitlab.AccessControlValue = "developer"
	var maintainer gitlab.AccessControlValue = "maintainer"
	var noOne gitlab.AccessControlValue = "no one"
	values := map[string]gitlab.AccessControlValue{
		"developer":  developer,
		"maintainer": maintainer,
		"no one":     noOne,
	}

	return values[input]
}

// Valid scopes for project access tokens
// See: https://docs.gitlab.com/user/project/settings/project_access_tokens/#scopes-for-a-project-access-token
var ValidProjectAccessTokenScopes = []string{
	"api",
	"read_api",
	"read_registry",
	"write_registry",
	"read_repository",
	"write_repository",
	"create_runner",
	"manage_runner",
	"ai_features",
	"k8s_proxy",
	"read_observability",
	"write_observability",
	"self_rotate",
}

// Valid scopes for group access tokens
// See: https://docs.gitlab.com/user/group/settings/group_access_tokens/#scopes-for-a-group-access-token
var ValidGroupAccessTokenScopes = []string{
	"api",
	"read_api",
	"read_registry",
	"write_registry",
	"read_virtual_registry",
	"write_virtual_registry",
	"read_repository",
	"write_repository",
	"create_runner",
	"manage_runner",
	"ai_features",
	"k8s_proxy",
	"read_observability",
	"write_observability",
	"self_rotate",
}

// Valid scopes for personal access tokens
// See: https://docs.gitlab.com/user/profile/personal_access_tokens/#personal-access-token-scopes
var ValidPersonalAccessTokenScopes = []string{
	"api",
	"read_user",
	"read_api",
	"read_repository",
	"write_repository",
	"read_registry",
	"write_registry",
	"read_virtual_registry",
	"write_virtual_registry",
	"sudo",
	"admin_mode",
	"create_runner",
	"manage_runner",
	"ai_features",
	"k8s_proxy",
	"self_rotate",
	"read_service_ping",
}

// Valid scopes for a project/group deploy token
// See: https://docs.gitlab.com/user/project/deploy_tokens/#scope
// There aren't separate docs for group tokens currently, these same scopes work for both project/group
var ValidDeployTokenScopes = []string{
	"read_repository",
	"read_registry",
	"write_registry",
	"read_virtual_registry",
	"write_virtual_registry",
	"read_package_registry",
	"write_package_registry",
}
