package sdk

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	validProjectAccessLevels = []string{
		"disabled",
		"private",
		"enabled",
	}
	validProjectAutoCancelPendingPipelinesValues = []string{
		"enabled",
		"disabled",
	}
	validProjectBuildGitStrategyValues = []string{
		"clone",
		"fetch",
	}
	validProjectAutoDevOpsDeployStrategyValues = []string{
		"continuous",
		"manual",
		"timed_incremental",
	}
	validMergeMethods = []string{
		"merge",
		"rebase_merge",
		"ff",
	}
	validPagesAccessLevelValues = []string{
		"public",
		"private",
		"enabled",
		"disabled",
	}
	validVisibilityLevelValues = []string{
		"private",
		"internal",
		"public",
	}
)

var resourceGitLabProjectSchema = map[string]*schema.Schema{
	"name": {
		Description: "The name of the project.",
		Type:        schema.TypeString,
		Required:    true,
	},
	"path": {
		Description: "The path of the repository.",
		Type:        schema.TypeString,
		Optional:    true,
		DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
			if new == "" {
				return true
			}
			return old == new
		},
	},
	"path_with_namespace": {
		Description: "The path of the repository with namespace.",
		Type:        schema.TypeString,
		Computed:    true,
	},
	"namespace_id": {
		Description: "The namespace (group or user) of the project. Defaults to your user.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
	"description": {
		Description: "A description of the project.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"default_branch": {
		Description: "The default branch for the project.",
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
	},
	"import_url": {
		Description:   "Git URL to a repository to be imported. Together with `mirror = true` it will setup a Pull Mirror. This can also be used together with `forked_from_project_id` to setup a Pull Mirror for a fork. The fork takes precedence over the import. Make sure to provide the credentials in `import_url_username` and `import_url_password`. GitLab never returns the credentials, thus the provider cannot detect configuration drift in the credentials. They can also not be imported using `terraform import`. See the examples section for how to properly use it.",
		Type:          schema.TypeString,
		Optional:      true,
		ConflictsWith: []string{"initialize_with_readme"},
	},
	"import_url_username": {
		Description:  "The username for the `import_url`. The value of this field is used to construct a valid `import_url` and is only related to the provider. This field cannot be imported using `terraform import`.  See the examples section for how to properly use it.",
		Type:         schema.TypeString,
		Optional:     true,
		RequiredWith: []string{"import_url", "import_url_password"},
	},
	"import_url_password": {
		Description:  "The password for the `import_url`. The value of this field is used to construct a valid `import_url` and is only related to the provider. This field cannot be imported using `terraform import`. See the examples section for how to properly use it.",
		Type:         schema.TypeString,
		Optional:     true,
		Sensitive:    true,
		RequiredWith: []string{"import_url", "import_url_username"},
	},
	"request_access_enabled": {
		Description: "Allow users to request member access.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"issues_enabled": {
		Description: "Enable issue tracking for the project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"merge_requests_enabled": {
		Description: "Enable merge requests for the project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"pipelines_enabled": {
		Description: "Enable pipelines for the project. The `pipelines_enabled` field is being sent as `jobs_enabled` in the GitLab API calls.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
		Deprecated:  "Deprecated in favor of `builds_access_level`",
	},
	"approvals_before_merge": {
		Description: `Number of merge request approvals required for merging. Default is 0.
  This field **does not** work well in combination with the ` + "`gitlab_project_approval_rule`" + ` resource
  and is most likely gonna be deprecated in a future GitLab version (see [this upstream epic](https://gitlab.com/groups/gitlab-org/-/epics/7572)).
  In the meantime we recommend against using this attribute and use ` + "`gitlab_project_approval_rule`" + ` instead.
`,
		Type:     schema.TypeInt,
		Optional: true,
	},
	"wiki_enabled": {
		Description: "Enable wiki for the project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"snippets_enabled": {
		Description: "Enable snippets for the project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"container_registry_enabled": {
		Description: "Enable container registry for the project.",
		Deprecated:  "Use `container_registry_access_level` instead.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"lfs_enabled": {
		Description: "Enable LFS for the project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"visibility_level": {
		Description:  fmt.Sprintf("Set to `public` to create a public project. Valid values are %s.", utils.RenderValueListForDocs(validVisibilityLevelValues)),
		Type:         schema.TypeString,
		Optional:     true,
		Computed:     true,
		ValidateFunc: validation.StringInSlice(validVisibilityLevelValues, true),
	},
	"merge_method": {
		Description:  fmt.Sprintf("Set the merge method. Valid values are %s.", utils.RenderValueListForDocs(validMergeMethods)),
		Type:         schema.TypeString,
		Optional:     true,
		Computed:     true,
		ValidateFunc: validation.StringInSlice(validMergeMethods, true),
	},
	"only_allow_merge_if_pipeline_succeeds": {
		Description: "Set to true if you want allow merges only if a pipeline succeeds.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"only_allow_merge_if_all_discussions_are_resolved": {
		Description: "Set to true if you want allow merges only if all discussions are resolved.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"allow_merge_on_skipped_pipeline": {
		Description: "Set to true if you want to treat skipped pipelines as if they finished with success.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"allow_pipeline_trigger_approve_deployment": {
		Description: "Set whether or not a pipeline triggerer is allowed to approve deployments. Premium and Ultimate only.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"restrict_user_defined_variables": {
		Description: "Allow only users with the Maintainer role to pass user-defined variables when triggering a pipeline.",
		Type:        schema.TypeBool,
		Optional:    true,
	},
	"ssh_url_to_repo": {
		Description: "URL that can be provided to `git clone` to clone the",
		Type:        schema.TypeString,
		Computed:    true,
	},
	"http_url_to_repo": {
		Description: "URL that can be provided to `git clone` to clone the",
		Type:        schema.TypeString,
		Computed:    true,
	},
	"web_url": {
		Description: "URL that can be used to find the project in a browser.",
		Type:        schema.TypeString,
		Computed:    true,
	},
	"runners_token": {
		Description: "Registration token to use during runner setup.",
		Type:        schema.TypeString,
		Computed:    true,
		Sensitive:   true,
	},
	"shared_runners_enabled": {
		Description: "Enable shared runners for this project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"group_runners_enabled": {
		Description: "Enable group runners for this project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"tags": {
		Description: "The list of tags for a project; put array of tags, that should be finally assigned to a project. Use topics instead.",
		Type:        schema.TypeSet,
		Optional:    true,
		Computed:    true,
		ForceNew:    false,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Set:         schema.HashString,
	},
	"empty_repo": {
		Description: "Whether the project is empty.",
		Type:        schema.TypeBool,
		Computed:    true,
	},
	"archived": {
		Description: "Whether the project is in read-only mode (archived). Repositories can be archived/unarchived by toggling this parameter.",
		Type:        schema.TypeBool,
		Optional:    true,
	},
	"initialize_with_readme": {
		Description:   "Create main branch with first commit containing a README.md file. Must be set to `true` if importing an uninitialized project with a different `default_branch`.",
		Type:          schema.TypeBool,
		Optional:      true,
		ConflictsWith: []string{"import_url", "forked_from_project_id"},
	},
	"squash_option": {
		Description:  "Squash commits when merge request. Valid values are `never`, `always`, `default_on`, or `default_off`. The default value is `default_off`. [GitLab >= 14.1]",
		Type:         schema.TypeString,
		Optional:     true,
		Computed:     true,
		ValidateFunc: validation.StringInSlice([]string{"never", "default_on", "always", "default_off"}, true),
	},
	"remove_source_branch_after_merge": {
		Description: "Enable `Delete source branch` option by default for all new merge requests.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"printing_merge_request_link_enabled": {
		Description: "Show link to create/view merge request when pushing from the command line",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"packages_enabled": {
		Description: "Enable packages repository for the project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"push_rules": {
		Description: "Push rules for the project.",
		Type:        schema.TypeList,
		MaxItems:    1,
		Optional:    true,
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"author_email_regex": {
					Description: "All commit author emails must match this regex, e.g. `@my-company.com$`.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"branch_name_regex": {
					Description: "All branch names must match this regex, e.g. `(feature|hotfix)\\/*`.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"commit_message_regex": {
					Description: "All commit messages must match this regex, e.g. `Fixed \\d+\\..*`.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"commit_message_negative_regex": {
					Description: "No commit message is allowed to match this regex, e.g. `ssh\\:\\/\\/`.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"file_name_regex": {
					Description: "All committed filenames must not match this regex, e.g. `(jar|exe)$`.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"commit_committer_check": {
					Description: "Users can only push commits to this repository that were committed with one of their own verified emails.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"commit_committer_name_check": {
					Description: "Users can only push commits to this repository if the commit author name is consistent with their GitLab account name.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"deny_delete_tag": {
					Description: "Deny deleting a tag.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"member_check": {
					Description: "Restrict commits by author (email) to existing GitLab users.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"prevent_secrets": {
					Description: "GitLab will reject any files that are likely to contain secrets.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"reject_unsigned_commits": {
					Description: "Reject commit when it’s not signed through GPG.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"reject_non_dco_commits": {
					Description: "Reject commit when it’s not DCO certified.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"max_file_size": {
					Description:  "Maximum file size (MB).",
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntAtLeast(0),
				},
			},
		},
	},
	"template_name": {
		Description:   "When used without use_custom_template, name of a built-in project template. When used with use_custom_template, name of a custom project template. This option is mutually exclusive with `template_project_id`.",
		Type:          schema.TypeString,
		Optional:      true,
		ConflictsWith: []string{"template_project_id"},
		ForceNew:      true,
	},
	"template_project_id": {
		Description:   "When used with use_custom_template, project ID of a custom project template. This is preferable to using template_name since template_name may be ambiguous (enterprise edition). This option is mutually exclusive with `template_name`. See `gitlab_group_project_file_template` to set a project as a template project. If a project has not been set as a template, using it here will result in an error.",
		Type:          schema.TypeInt,
		Optional:      true,
		ConflictsWith: []string{"template_name"},
		ForceNew:      true,
	},
	"use_custom_template": {
		Description: `Use either custom instance or group (with group_with_project_templates_id) project template (enterprise edition).
		~> When using a custom template, [Group Tokens won't work](https://docs.gitlab.com/15.7/ee/user/project/settings/import_export_troubleshooting.html#import-using-the-rest-api-fails-when-using-a-group-access-token). You must use a real user's Personal Access Token.`,
		Type:     schema.TypeBool,
		Optional: true,
	},
	"group_with_project_templates_id": {
		Description: "For group-level custom templates, specifies ID of group from which all the custom project templates are sourced. Leave empty for instance-level templates. Requires use_custom_template to be true (enterprise edition).",
		Type:        schema.TypeInt,
		Optional:    true,
	},
	"pages_access_level": {
		Description:  fmt.Sprintf("Enable pages access control. Valid values are %s.", utils.RenderValueListForDocs(validPagesAccessLevelValues)),
		Type:         schema.TypeString,
		Optional:     true,
		Computed:     true,
		ValidateFunc: validation.StringInSlice(validPagesAccessLevelValues, true),
	},
	// The GitLab API requires that import_url is also set when mirror options are used
	// Ref: https://gitlab.com/gitlab-org/terraform-provider-gitlab/pull/449#discussion_r549729230
	"mirror": {
		Description:  "Enable project pull mirror.",
		Type:         schema.TypeBool,
		Optional:     true,
		RequiredWith: []string{"import_url"},
	},
	"mirror_trigger_builds": {
		Description:  "Enable trigger builds on pushes for a mirrored project.",
		Type:         schema.TypeBool,
		Optional:     true,
		Computed:     true,
		RequiredWith: []string{"import_url"},
	},
	"mirror_overwrites_diverged_branches": {
		Description:  "Enable overwrite diverged branches for a mirrored project.",
		Type:         schema.TypeBool,
		Optional:     true,
		Computed:     true,
		RequiredWith: []string{"import_url"},
	},
	"only_mirror_protected_branches": {
		Description:  "Enable only mirror protected branches for a mirrored project.",
		Type:         schema.TypeBool,
		Optional:     true,
		Computed:     true,
		RequiredWith: []string{"import_url"},
	},
	"build_coverage_regex": {
		Description: "Test coverage parsing for the project. This is deprecated feature in GitLab 15.0.",
		Type:        schema.TypeString,
		Optional:    true,
		Deprecated:  "build_coverage_regex is removed in GitLab 15.0.",
	},
	"issues_template": {
		Description: "Sets the template for new issues in the project.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"merge_requests_template": {
		Description: "Sets the template for new merge requests in the project.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"ci_config_path": {
		Description: "Custom Path to CI config file.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"archive_on_destroy": {
		Description: "Set to `true` to archive the project instead of deleting on destroy. If set to `true` it will entire omit the `DELETE` operation.",
		Type:        schema.TypeBool,
		Optional:    true,
	},
	"ci_forward_deployment_enabled": {
		Description: "When a new deployment job starts, skip older deployment jobs that are still pending.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"ci_separated_caches": {
		Description: "Use separate caches for protected branches.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"ci_restrict_pipeline_cancellation_role": {
		Description: fmt.Sprintf("The role required to cancel a pipeline or job. Introduced in GitLab 16.8. Premium and Ultimate only. Valid values are %s", utils.RenderValueListForDocs(api.ValidCIRestrictPipelineCancellationRoleValues)),
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
	},
	"ci_pipeline_variables_minimum_override_role": {
		Description:  fmt.Sprintf("The minimum role required to set variables when running pipelines and jobs. Introduced in GitLab 17.1. Valid values are %s", utils.RenderValueListForDocs(api.ValidCIPipelineVariablesMinimumOverrideRoleValues)),
		Type:         schema.TypeString,
		Optional:     true,
		Computed:     true,
		ValidateFunc: validation.StringInSlice(api.ValidCIPipelineVariablesMinimumOverrideRoleValues, true),
	},
	"keep_latest_artifact": {
		Description: "Disable or enable the ability to keep the latest artifact for this project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"merge_pipelines_enabled": {
		Description: "Enable or disable merge pipelines.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"merge_trains_enabled": {
		Description: "Enable or disable merge trains. Requires `merge_pipelines_enabled` to be set to `true` to take effect.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"resolve_outdated_diff_discussions": {
		Description: "Automatically resolve merge request diffs discussions on lines changed with a push.",
		Type:        schema.TypeBool,
		Optional:    true,
	},
	"analytics_access_level": {
		Description:      fmt.Sprintf("Set the analytics access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"auto_cancel_pending_pipelines": {
		Description:      "Auto-cancel pending pipelines. This isn’t a boolean, but enabled/disabled.",
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAutoCancelPendingPipelinesValues, false)),
	},
	"auto_devops_deploy_strategy": {
		Description:      fmt.Sprintf("Auto Deploy strategy. Valid values are %s.", utils.RenderValueListForDocs(validProjectAutoDevOpsDeployStrategyValues)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAutoDevOpsDeployStrategyValues, false)),
	},
	"auto_devops_enabled": {
		Description: "Enable Auto DevOps for this project.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"autoclose_referenced_issues": {
		Description: "Set whether auto-closing referenced issues on default branch.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"build_git_strategy": {
		Description:      fmt.Sprintf("The Git strategy. Defaults to fetch. Valid values are %s.", utils.RenderValueListForDocs(validProjectBuildGitStrategyValues)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectBuildGitStrategyValues, false)),
	},
	"build_timeout": {
		Description: "The maximum amount of time, in seconds, that a job can run.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
	"builds_access_level": {
		Description:      fmt.Sprintf("Set the builds access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"container_expiration_policy": {
		Description: "Set the image cleanup policy for this project. **Note**: this field is sometimes named `container_expiration_policy_attributes` in the GitLab Upstream API.",
		Type:        schema.TypeList,
		MaxItems:    1,
		Elem:        resourceContainerExpirationPolicyAttributesSchema,
		Optional:    true,
		Computed:    true,
	},
	"container_registry_access_level": {
		Description:      fmt.Sprintf("Set visibility of container registry, for this project. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"emails_enabled": {
		Description: "Enable email notifications.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"external_authorization_classification_label": {
		Description: "The classification label for the project.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"forking_access_level": {
		Description:      fmt.Sprintf("Set the forking access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"issues_access_level": {
		Description:      fmt.Sprintf("Set the issues access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"merge_requests_access_level": {
		Description:      fmt.Sprintf("Set the merge requests access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"public_jobs": {
		Description:   "If true, jobs can be viewed by non-project members.",
		Type:          schema.TypeBool,
		Optional:      true,
		Computed:      true,
		ConflictsWith: []string{"public_builds"},
	},
	"public_builds": {
		Description:   "If true, jobs can be viewed by non-project members.",
		Type:          schema.TypeBool,
		Optional:      true,
		Computed:      true,
		ConflictsWith: []string{"public_jobs"},
		Deprecated:    "The `public_builds` attribute has been deprecated in favor of `public_jobs` and will be removed in the next major version of the provider.",
	},
	"repository_access_level": {
		Description:      fmt.Sprintf("Set the repository access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"repository_storage": {
		Description: "	Which storage shard the repository is on. (administrator only)",
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
	},
	"requirements_access_level": {
		Description:      fmt.Sprintf("Set the requirements access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"security_and_compliance_access_level": {
		Description:      fmt.Sprintf("Set the security and compliance access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"snippets_access_level": {
		Description:      fmt.Sprintf("Set the snippets access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"suggestion_commit_message": {
		Description: "The commit message used to apply merge request suggestions.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"topics": {
		Description: "The list of topics for the project.",
		Type:        schema.TypeSet,
		Set:         schema.HashString,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Optional:    true,
		Computed:    true,
	},
	"wiki_access_level": {
		Description:      fmt.Sprintf("Set the wiki access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"squash_commit_template": {
		Description: "Template used to create squash commit message in merge requests. (Introduced in GitLab 14.6.)",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"merge_commit_template": {
		Description: "Template used to create merge commit message in merge requests. (Introduced in GitLab 14.5.)",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"ci_default_git_depth": {
		Description: "Default number of revisions for shallow cloning.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
	"forked_from_project_id": {
		Description:   "The id of the project to fork. During create the project is forked and during an update the fork relation is changed.",
		Type:          schema.TypeInt,
		Optional:      true,
		ConflictsWith: []string{"initialize_with_readme"},
	},
	"mr_default_target_self": {
		Description:  "For forked projects, target merge requests to this project. If false, the target will be the upstream project.",
		Type:         schema.TypeBool,
		Optional:     true,
		RequiredWith: []string{"forked_from_project_id"},
	},
	"releases_access_level": {
		Description:      fmt.Sprintf("Set the releases access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"environments_access_level": {
		Description:      fmt.Sprintf("Set the environments access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"feature_flags_access_level": {
		Description:      fmt.Sprintf("Set the feature flags access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"infrastructure_access_level": {
		Description:      fmt.Sprintf("Set the infrastructure access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"monitor_access_level": {
		Description:      fmt.Sprintf("Set the monitor access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"pre_receive_secret_detection_enabled": {
		Description: "Whether Secret Push Detection is enabled. Requires GitLab Ultimate and at least GitLab 17.3.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
	},
	"model_experiments_access_level": {
		Description:      fmt.Sprintf("Set visibility of machine learning model experiments. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
	"model_registry_access_level": {
		Description:      fmt.Sprintf("Set visibility of machine learning model registry. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validProjectAccessLevels, false)),
	},
}

var validContainerExpirationPolicyAttributesCadenceValues = []string{
	"1d", "7d", "14d", "1month", "3month",
}

var resourceContainerExpirationPolicyAttributesSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"cadence": {
			Description:      fmt.Sprintf("The cadence of the policy. Valid values are: %s.", utils.RenderValueListForDocs(validContainerExpirationPolicyAttributesCadenceValues)),
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validContainerExpirationPolicyAttributesCadenceValues, false)),
		},
		"keep_n": {
			Description:      "The number of images to keep.",
			Type:             schema.TypeInt,
			Optional:         true,
			Computed:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
		},
		"older_than": {
			Description: "The number of days to keep images.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},
		"name_regex": {
			Description: "The regular expression to match image names to delete.",
			Deprecated:  "`name_regex` has been deprecated. Use `name_regex_delete` instead.",
			Type:        schema.TypeString,
			ConflictsWith: []string{
				"container_expiration_policy.0.name_regex_delete",
			},
			Optional: true,
			Computed: true,
		},
		"name_regex_delete": {
			Description: "The regular expression to match image names to delete.",
			ConflictsWith: []string{
				"container_expiration_policy.0.name_regex",
			},
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"name_regex_keep": {
			Description: "The regular expression to match image names to keep.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},
		"enabled": {
			Description: "If true, the policy is enabled.",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},
		"next_run_at": {
			Description: "The next time the policy will run.",
			Type:        schema.TypeString,
			Computed:    true,
		},
	},
}

var _ = registerResource("gitlab_project", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_project`" + ` resource allows to manage the lifecycle of a project.

A project can either be created in a group or user namespace.

-> **Default Branch Protection Workaround** Projects are created with default branch protection.
Since this default branch protection is not currently managed via Terraform, to workaround this limitation,
you can remove the default branch protection via the API and create your desired Terraform managed branch protection.
In the ` + "`gitlab_project`" + ` resource, define a ` + "`local-exec`" + ` provisioner which invokes
the ` + "`/projects/:id/protected_branches/:name`" + ` API via curl to delete the branch protection on the default
branch using a ` + "`DELETE`" + ` request. Then define the desired branch protection using the ` + "`gitlab_branch_protection`" + ` resource.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ce/api/projects.html)`,

		CreateContext: resourceGitlabProjectCreate,
		ReadContext:   resourceGitlabProjectRead,
		UpdateContext: resourceGitlabProjectUpdate,
		DeleteContext: resourceGitlabProjectDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			// 10 minutes is longer than the previous 2, but it's set here
			// to match the existing "Import" timeout that happens during create
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: constructSchema(resourceGitLabProjectSchema, avatarableSchema(), map[string]*schema.Schema{
			"skip_wait_for_default_branch_protection": {
				Description: `If ` + "`true`" + `, the default behavior to wait for the default branch protection to be created is skipped.
This is necessary if the current user is not an admin and the default branch protection is disabled on an instance-level.
There is currently no known way to determine if the default branch protection is disabled on an instance-level for non-admin users.
This attribute is only used during resource creation, thus changes are suppressed and the attribute cannot be imported.
`,
				Type:     schema.TypeBool,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return d.Id() != ""
				},
			},
		}),
		CustomizeDiff: customdiff.All(
			customdiff.ComputedIf("path_with_namespace", namespaceOrPathChanged),
			customdiff.ComputedIf("ssh_url_to_repo", namespaceOrPathChanged),
			customdiff.ComputedIf("http_url_to_repo", namespaceOrPathChanged),
			customdiff.ComputedIf("web_url", namespaceOrPathChanged),
			avatarableDiff,
		),
	}
})

func resourceGitlabProjectSetToState(ctx context.Context, client *gitlab.Client, d *schema.ResourceData, project *gitlab.Project) error {
	d.SetId(fmt.Sprintf("%d", project.ID))
	d.Set("name", project.Name)
	d.Set("path", project.Path)
	d.Set("path_with_namespace", project.PathWithNamespace)
	d.Set("description", project.Description)
	d.Set("default_branch", project.DefaultBranch)
	d.Set("request_access_enabled", project.RequestAccessEnabled)
	d.Set("issues_enabled", project.IssuesEnabled)
	d.Set("merge_requests_enabled", project.MergeRequestsEnabled)
	d.Set("pipelines_enabled", project.JobsEnabled)
	d.Set("approvals_before_merge", project.ApprovalsBeforeMerge)
	d.Set("wiki_enabled", project.WikiEnabled)
	d.Set("snippets_enabled", project.SnippetsEnabled)
	d.Set("container_registry_enabled", project.ContainerRegistryEnabled)
	d.Set("lfs_enabled", project.LFSEnabled)
	d.Set("visibility_level", string(project.Visibility))
	d.Set("merge_method", string(project.MergeMethod))
	d.Set("only_allow_merge_if_pipeline_succeeds", project.OnlyAllowMergeIfPipelineSucceeds)
	d.Set("only_allow_merge_if_all_discussions_are_resolved", project.OnlyAllowMergeIfAllDiscussionsAreResolved)
	d.Set("allow_merge_on_skipped_pipeline", project.AllowMergeOnSkippedPipeline)
	d.Set("allow_pipeline_trigger_approve_deployment", project.AllowPipelineTriggerApproveDeployment)
	d.Set("restrict_user_defined_variables", project.RestrictUserDefinedVariables)
	d.Set("namespace_id", project.Namespace.ID)
	d.Set("ssh_url_to_repo", project.SSHURLToRepo)
	d.Set("http_url_to_repo", project.HTTPURLToRepo)
	d.Set("web_url", project.WebURL)
	d.Set("runners_token", project.RunnersToken)
	d.Set("shared_runners_enabled", project.SharedRunnersEnabled)
	d.Set("group_runners_enabled", project.GroupRunnersEnabled)
	if err := d.Set("tags", project.TagList); err != nil {
		return err
	}
	d.Set("empty_repo", project.EmptyRepo)
	d.Set("archived", project.Archived)
	if supportsSquashOption, err := api.IsGitLabVersionAtLeast(ctx, client, "14.1")(); err != nil {
		return err
	} else if supportsSquashOption {
		d.Set("squash_option", project.SquashOption)
	}
	d.Set("remove_source_branch_after_merge", project.RemoveSourceBranchAfterMerge)
	d.Set("printing_merge_request_link_enabled", project.PrintingMergeRequestLinkEnabled)
	d.Set("packages_enabled", project.PackagesEnabled)
	d.Set("pages_access_level", string(project.PagesAccessLevel))
	d.Set("mirror", project.Mirror)
	d.Set("mirror_trigger_builds", project.MirrorTriggerBuilds)
	d.Set("mirror_overwrites_diverged_branches", project.MirrorOverwritesDivergedBranches)
	d.Set("only_mirror_protected_branches", project.OnlyMirrorProtectedBranches)
	d.Set("import_url", project.ImportURL)
	d.Set("issues_template", project.IssuesTemplate)
	d.Set("merge_requests_template", project.MergeRequestsTemplate)
	d.Set("ci_config_path", project.CIConfigPath)
	d.Set("ci_forward_deployment_enabled", project.CIForwardDeploymentEnabled)
	d.Set("ci_separated_caches", project.CISeperateCache)
	d.Set("ci_restrict_pipeline_cancellation_role", project.CIRestrictPipelineCancellationRole)
	d.Set("ci_pipeline_variables_minimum_override_role", project.CIPipelineVariablesMinimumOverrideRole)
	d.Set("keep_latest_artifact", project.KeepLatestArtifact)
	d.Set("merge_pipelines_enabled", project.MergePipelinesEnabled)
	d.Set("merge_trains_enabled", project.MergeTrainsEnabled)
	d.Set("resolve_outdated_diff_discussions", project.ResolveOutdatedDiffDiscussions)
	d.Set("analytics_access_level", string(project.AnalyticsAccessLevel))
	d.Set("auto_cancel_pending_pipelines", project.AutoCancelPendingPipelines)
	d.Set("auto_devops_deploy_strategy", project.AutoDevopsDeployStrategy)
	d.Set("auto_devops_enabled", project.AutoDevopsEnabled)
	d.Set("autoclose_referenced_issues", project.AutocloseReferencedIssues)
	d.Set("build_git_strategy", project.BuildGitStrategy)
	d.Set("build_timeout", project.BuildTimeout)
	d.Set("builds_access_level", string(project.BuildsAccessLevel))
	if err := d.Set("container_expiration_policy", flattenContainerExpirationPolicy(project.ContainerExpirationPolicy)); err != nil {
		return fmt.Errorf("error setting container_expiration_policy: %v", err)
	}
	d.Set("container_registry_access_level", string(project.ContainerRegistryAccessLevel))
	d.Set("emails_enabled", project.EmailsEnabled)

	d.Set("external_authorization_classification_label", project.ExternalAuthorizationClassificationLabel)
	d.Set("forking_access_level", string(project.ForkingAccessLevel))
	d.Set("issues_access_level", string(project.IssuesAccessLevel))
	d.Set("merge_requests_access_level", string(project.MergeRequestsAccessLevel))

	// First, try to set the public_jobs. If it's not available, fall back to public_builds.
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if err := d.Set("public_jobs", project.PublicJobs); err != nil {
		if err := d.Set("public_builds", project.PublicJobs); err != nil {
			return fmt.Errorf("error setting public_jobs: %v", err)
		}
	}

	d.Set("repository_access_level", string(project.RepositoryAccessLevel))
	d.Set("repository_storage", project.RepositoryStorage)
	d.Set("requirements_access_level", string(project.RequirementsAccessLevel))
	d.Set("security_and_compliance_access_level", string(project.SecurityAndComplianceAccessLevel))
	d.Set("snippets_access_level", string(project.SnippetsAccessLevel))
	d.Set("suggestion_commit_message", project.SuggestionCommitMessage)
	if err := d.Set("topics", project.Topics); err != nil {
		return fmt.Errorf("error setting topics: %v", err)
	}
	d.Set("wiki_access_level", string(project.WikiAccessLevel))
	d.Set("squash_commit_template", project.SquashCommitTemplate)
	d.Set("merge_commit_template", project.MergeCommitTemplate)

	// Note: This field is deprecated and will always be an empty string starting in GitLab 15.0.
	d.Set("build_coverage_regex", project.BuildCoverageRegex)

	d.Set("ci_default_git_depth", project.CIDefaultGitDepth)
	d.Set("avatar_url", project.AvatarURL)

	if project.ForkedFromProject != nil {
		d.Set("forked_from_project_id", project.ForkedFromProject.ID)
	} else {
		d.Set("forked_from_project_id", nil)
	}

	d.Set("mr_default_target_self", project.MergeRequestDefaultTargetSelf)

	d.Set("releases_access_level", string(project.ReleasesAccessLevel))
	d.Set("environments_access_level", string(project.EnvironmentsAccessLevel))
	d.Set("feature_flags_access_level", string(project.FeatureFlagsAccessLevel))
	d.Set("infrastructure_access_level", string(project.InfrastructureAccessLevel))
	d.Set("monitor_access_level", string(project.MonitorAccessLevel))
	d.Set("pre_receive_secret_detection_enabled", project.PreReceiveSecretDetectionEnabled)
	d.Set("model_experiments_access_level", string(project.ModelExperimentsAccessLevel))
	d.Set("model_registry_access_level", string(project.ModelRegistryAccessLevel))

	return nil
}

func resourceGitlabProjectCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	// Project that has either been created or forked
	var project *gitlab.Project

	// The base project is created in one of two ways: We either create it from scratch,
	// or it's forked from an existing project and edited. This block checks the forked
	// status and handles the base create or forked logic
	if forkedFromProjectID, ok := d.GetOk("forked_from_project_id"); ok {
		tflog.Debug(ctx, "Creating forked project", map[string]interface{}{
			"forked_from_project_id": forkedFromProjectID,
			"data":                   d,
		})

		createdProject, diag := createForkedProject(ctx, forkedFromProjectID.(int), d, client)
		if diag != nil {
			return diag
		}
		project = createdProject

	} else {

		tflog.Debug(ctx, "Creating project", map[string]interface{}{
			"data": d,
		})

		createdProject, diag := createProject(ctx, d, client)
		if diag != nil {
			return diag
		}
		project = createdProject

	}

	// from this point onwards no matter how we return, resource creation
	// is committed to state since we set its ID
	d.SetId(fmt.Sprintf("%d", project.ID))

	// An import can be triggered by import_url or by creating the project from a template or from a fork.
	if project.ImportStatus != "none" {
		tflog.Debug(ctx, fmt.Sprintf("waiting for project %q import to finish", project.Name))

		stateConf := &retry.StateChangeConf{
			Pending: []string{"scheduled", "started"},
			Target:  []string{"finished"},
			Timeout: d.Timeout(schema.TimeoutCreate),
			Refresh: func() (interface{}, string, error) {
				status, _, err := client.ProjectImportExport.ImportStatus(d.Id(), gitlab.WithContext(ctx))
				if err != nil {
					return nil, "", err
				}

				return status, status.ImportStatus, nil
			},
		}

		if _, err := stateConf.WaitForStateContext(ctx); err != nil {
			return diag.Errorf("error while waiting for project %q import to finish: %s", project.Name, err)
		}

		// Read the project again, so that we can detect the default branch.
		var err error
		project, _, err = client.Projects.GetProject(project.ID, nil, gitlab.WithContext(ctx))
		if err != nil {
			return diag.Errorf("Failed to get project %q after completing import: %s", d.Id(), err)
		}
	}

	if d.Get("archived").(bool) {
		// strange as it may seem, this project is created in archived state...
		if _, _, err := client.Projects.ArchiveProject(d.Id(), gitlab.WithContext(ctx)); err != nil {
			return diag.Errorf("new project %q could not be archived: %s", d.Id(), err)
		}
	}

	if _, ok := d.GetOk("push_rules"); ok {
		err := editOrAddPushRules(ctx, client, d.Id(), d)
		if err != nil {
			if api.Is404(err) {
				tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Failed to edit push rules for project %q: %v", d.Id(), err))
				return diag.Errorf("Project push rules are not supported in your version of GitLab")
			}
			return diag.Errorf("Failed to edit push rules for project %q: %s", d.Id(), err)
		}
	}

	// If enabling Secret Push Detection, then update that value on the project via
	// GraphQL
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if _, ok := d.GetOkExists("pre_receive_secret_detection_enabled"); ok {
		val := d.Get("pre_receive_secret_detection_enabled").(bool)
		err := updateProjectSecretDetectionValue(ctx, client, project.PathWithNamespace, val)
		if err != nil {
			return diag.Errorf("Error updating Secret Push Detection on Project %d: %v", project.ID, err)
		}
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("skip_wait_for_default_branch_protection"); ok {
		d.Set("skip_wait_for_default_branch_protection", v.(bool))
	}
	if !d.Get("skip_wait_for_default_branch_protection").(bool) {
		// If the project is assigned to a group namespace and the group has *default branch protection*
		// disabled (`default_branch_protection = 0`) then we don't have to wait for one.
		waitForDefaultBranchProtection, err := expectDefaultBranchProtection(ctx, client, project)
		if err != nil {
			return diag.Errorf("Failed to discover if branch protection is enabled by default or not for project %d: %+v", project.ID, err)
		}

		if waitForDefaultBranchProtection {
			var branch *gitlab.Branch

			// Branch protection for a newly created branch is an async action, so use WaitForState to ensure it's protected
			// before we continue. Note this check should only be required when there is a custom default branch set
			// See issue 800: https://gitlab.com/gitlab-org/terraform-provider-gitlab/issues/800
			stateConf := &retry.StateChangeConf{
				Pending: []string{"false"},
				Target:  []string{"true"},

				// The async action usually completes very quickly, within seconds. However in
				// lower compute or disk constrained environment, it can take a while.
				// When importing a project and changing the branch protection, the "TimeoutCreate" may
				// happen twice, and that's OK.
				Timeout: d.Timeout(schema.TimeoutCreate),
				Refresh: func() (interface{}, string, error) {
					branch, _, err = client.Branches.GetBranch(project.ID, project.DefaultBranch, gitlab.WithContext(ctx))
					if err != nil {
						if api.Is404(err) {
							// When we hit a 404 here, it means the default branch wasn't created at all as part of the project
							// this will happen when "default_branch" isn't set, or "initialize_with_readme" is set to false.
							// We don't need to wait anymore, so return "true" to exist the wait loop.
							return branch, "true", nil
						}

						// This is legit error, return the error.
						tflog.Debug(ctx, "Error received when attempting to read branch protection of the default branch", map[string]interface{}{
							"error":   err,
							"project": project,
							"branch":  project.DefaultBranch,
						})
						return nil, "", err
					}

					tflog.Debug(ctx, "Project polling for default branch status", map[string]interface{}{
						"project":          project,
						"branch":           project.DefaultBranch,
						"protectionStatus": branch.Protected,
					})
					return branch, strconv.FormatBool(branch.Protected), nil
				},
			}

			if _, err := stateConf.WaitForStateContext(ctx); err != nil {
				return diag.Errorf("error while waiting for branch %s to reach 'protected' status; current status is protected %t, %s", branch.Name, branch.Protected, err)
			}
		}
	}

	// Create our "EditProjectOptions" call using state and the existing project
	var editProjectOptions gitlab.EditProjectOptions
	updatePostCreateEditOptions(ctx, &editProjectOptions, d, client, project)

	if (editProjectOptions != gitlab.EditProjectOptions{}) {
		if _, _, err := client.Projects.EditProject(d.Id(), &editProjectOptions, gitlab.WithContext(ctx)); err != nil {
			return diag.Errorf("Could not update project %q: %s", d.Id(), err)
		}
	}

	return resourceGitlabProjectRead(ctx, d, meta)
}

func resourceGitlabProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab project %s", d.Id()))

	project, _, err := client.Projects.GetProject(d.Id(), nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab project %s has already been deleted, removing from state", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if project.MarkedForDeletionAt != nil {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab project %s is marked for deletion, removing from state", d.Id()))
		d.SetId("")
		return nil
	}

	if err := resourceGitlabProjectSetToState(ctx, client, d, project); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab project %q push rules", d.Id()))

	pushRules, _, err := client.Projects.GetProjectPushRules(d.Id(), gitlab.WithContext(ctx))
	if api.Is404(err) {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Failed to get push rules for project %q: %v", d.Id(), err))
	} else if err != nil {
		return diag.Errorf("Failed to get push rules for project %q: %s", d.Id(), err)
	}

	if err := d.Set("push_rules", flattenProjectPushRules(pushRules)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceGitlabProjectUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	// Always send the name field, to satisfy the requirement of having one
	// of the project attributes listed below in the update call
	// https://gitlab.com/gitlab-org/gitlab-foss/-/blob/master/lib/api/helpers/projects_helpers.rb#L120-188
	options := &gitlab.EditProjectOptions{
		Name: gitlab.Ptr(d.Get("name").(string)),
	}

	transferOptions := &gitlab.TransferProjectOptions{}

	if d.HasChange("name") {
		options.Name = gitlab.Ptr(d.Get("name").(string))
	}

	if d.HasChange("path") && (d.Get("path").(string) != "") {
		options.Path = gitlab.Ptr(d.Get("path").(string))
	}

	if d.HasChange("namespace_id") {
		transferOptions.Namespace = gitlab.Ptr(d.Get("namespace_id").(int))
	}

	if d.HasChange("description") {
		options.Description = gitlab.Ptr(d.Get("description").(string))
	}

	if d.HasChange("default_branch") {
		options.DefaultBranch = gitlab.Ptr(d.Get("default_branch").(string))
	}

	if d.HasChange("visibility_level") {
		options.Visibility = stringToVisibilityLevel(d.Get("visibility_level").(string))
	}

	if d.HasChange("merge_method") {
		options.MergeMethod = stringToMergeMethod(d.Get("merge_method").(string))
	}

	if d.HasChange("only_allow_merge_if_pipeline_succeeds") {
		options.OnlyAllowMergeIfPipelineSucceeds = gitlab.Ptr(d.Get("only_allow_merge_if_pipeline_succeeds").(bool))
	}

	if d.HasChange("only_allow_merge_if_all_discussions_are_resolved") {
		options.OnlyAllowMergeIfAllDiscussionsAreResolved = gitlab.Ptr(d.Get("only_allow_merge_if_all_discussions_are_resolved").(bool))
	}

	if d.HasChange("allow_merge_on_skipped_pipeline") {
		options.AllowMergeOnSkippedPipeline = gitlab.Ptr(d.Get("allow_merge_on_skipped_pipeline").(bool))
	}

	if d.HasChange("allow_pipeline_trigger_approve_deployment") {
		options.AllowPipelineTriggerApproveDeployment = gitlab.Ptr(d.Get("allow_pipeline_trigger_approve_deployment").(bool))
	}

	if d.HasChange("restrict_user_defined_variables") {
		options.RestrictUserDefinedVariables = gitlab.Ptr(d.Get("restrict_user_defined_variables").(bool))
	}

	if d.HasChange("request_access_enabled") {
		options.RequestAccessEnabled = gitlab.Ptr(d.Get("request_access_enabled").(bool))
	}

	if d.HasChange("issues_enabled") {
		// TODO: Remove issuesEnabled on the next breaking update, since it will need to be replaced with a
		// issue access level integer.
		// nolint:staticcheck // SA1019
		options.IssuesEnabled = gitlab.Ptr(d.Get("issues_enabled").(bool))
	}

	if d.HasChange("merge_requests_enabled") {
		// TODO: Remove mergeRequestsEnabled on the next breaking update, since it will need to be replaced with a
		// merge request access level integer.
		// nolint:staticcheck // SA1019
		options.MergeRequestsEnabled = gitlab.Ptr(d.Get("merge_requests_enabled").(bool))
	}

	if d.HasChange("pipelines_enabled") {
		// nolint:staticcheck // SA1019
		options.JobsEnabled = gitlab.Ptr(d.Get("pipelines_enabled").(bool))
	}

	if d.HasChange("approvals_before_merge") {
		options.ApprovalsBeforeMerge = gitlab.Ptr(d.Get("approvals_before_merge").(int))
	}

	if d.HasChange("wiki_enabled") {
		// nolint:staticcheck // SA1019
		options.WikiEnabled = gitlab.Ptr(d.Get("wiki_enabled").(bool))
	}

	if d.HasChange("snippets_enabled") {
		// nolint:staticcheck // SA1019
		options.SnippetsEnabled = gitlab.Ptr(d.Get("snippets_enabled").(bool))
	}

	if d.HasChange("shared_runners_enabled") {
		options.SharedRunnersEnabled = gitlab.Ptr(d.Get("shared_runners_enabled").(bool))
	}

	if d.HasChange("group_runners_enabled") {
		options.GroupRunnersEnabled = gitlab.Ptr(d.Get("group_runners_enabled").(bool))
	}

	if d.HasChange("tags") {
		// nolint:staticcheck // SA1019
		options.TagList = stringSetToStringSlice(d.Get("tags").(*schema.Set))
	}

	if d.HasChange("container_registry_enabled") {
		// nolint:staticcheck // SA1019
		options.ContainerRegistryEnabled = gitlab.Ptr(d.Get("container_registry_enabled").(bool))
	}

	if d.HasChange("lfs_enabled") {
		options.LFSEnabled = gitlab.Ptr(d.Get("lfs_enabled").(bool))
	}

	if supportsSquashOption, err := api.IsGitLabVersionAtLeast(ctx, client, "14.1")(); err != nil {
		return diag.FromErr(err)
	} else if supportsSquashOption && d.HasChange("squash_option") {
		options.SquashOption = stringToSquashOptionValue(d.Get("squash_option").(string))
	}

	if d.HasChange("remove_source_branch_after_merge") {
		options.RemoveSourceBranchAfterMerge = gitlab.Ptr(d.Get("remove_source_branch_after_merge").(bool))
	}

	if d.HasChange("printing_merge_request_link_enabled") {
		options.PrintingMergeRequestLinkEnabled = gitlab.Ptr(d.Get("printing_merge_request_link_enabled").(bool))
	}

	if d.HasChange("packages_enabled") {
		options.PackagesEnabled = gitlab.Ptr(d.Get("packages_enabled").(bool))
	}

	if d.HasChange("pages_access_level") {
		options.PagesAccessLevel = stringToAccessControlValue(d.Get("pages_access_level").(string))
	}

	if d.HasChanges("mirror", "import_url", "import_url_username", "import_url_password") {
		options.Mirror = gitlab.Ptr(d.Get("mirror").(bool))
		importURL, err := constructImportUrl(d.Get("import_url").(string), d.Get("import_url_username").(string), d.Get("import_url_password").(string))
		if err != nil {
			return diag.Errorf("Unable to construct import URL for API: %s", err)
		}
		options.ImportURL = gitlab.Ptr(importURL)
	}

	if d.HasChange("mirror_trigger_builds") {
		options.MirrorTriggerBuilds = gitlab.Ptr(d.Get("mirror_trigger_builds").(bool))
		if options.ImportURL == nil {
			importURL, err := constructImportUrl(d.Get("import_url").(string), d.Get("import_url_username").(string), d.Get("import_url_password").(string))
			if err != nil {
				return diag.Errorf("Unable to construct import URL for API: %s", err)
			}
			options.ImportURL = gitlab.Ptr(importURL)
		}
	}

	if d.HasChange("only_mirror_protected_branches") {
		options.OnlyMirrorProtectedBranches = gitlab.Ptr(d.Get("only_mirror_protected_branches").(bool))
		if options.ImportURL == nil {
			importURL, err := constructImportUrl(d.Get("import_url").(string), d.Get("import_url_username").(string), d.Get("import_url_password").(string))
			if err != nil {
				return diag.Errorf("Unable to construct import URL for API: %s", err)
			}
			options.ImportURL = gitlab.Ptr(importURL)
		}
	}

	if d.HasChange("mirror_overwrites_diverged_branches") {
		options.MirrorOverwritesDivergedBranches = gitlab.Ptr(d.Get("mirror_overwrites_diverged_branches").(bool))
		if options.ImportURL == nil {
			importURL, err := constructImportUrl(d.Get("import_url").(string), d.Get("import_url_username").(string), d.Get("import_url_password").(string))
			if err != nil {
				return diag.Errorf("Unable to construct import URL for API: %s", err)
			}
			options.ImportURL = gitlab.Ptr(importURL)
		}
	}

	if d.HasChange("build_coverage_regex") {
		options.IssuesTemplate = gitlab.Ptr(d.Get("build_coverage_regex").(string))
	}

	if d.HasChange("issues_template") {
		options.IssuesTemplate = gitlab.Ptr(d.Get("issues_template").(string))
	}

	if d.HasChange("merge_requests_template") {
		options.MergeRequestsTemplate = gitlab.Ptr(d.Get("merge_requests_template").(string))
	}

	if d.HasChange("ci_config_path") {
		options.CIConfigPath = gitlab.Ptr(d.Get("ci_config_path").(string))
	}

	if d.HasChange("ci_forward_deployment_enabled") {
		options.CIForwardDeploymentEnabled = gitlab.Ptr(d.Get("ci_forward_deployment_enabled").(bool))
	}

	if d.HasChange("ci_restrict_pipeline_cancellation_role") {
		stringVal := d.Get("ci_restrict_pipeline_cancellation_role").(string)
		options.CIRestrictPipelineCancellationRole = gitlab.Ptr(api.AccessControlLevelValueToName(stringVal))
	}

	if d.HasChange("ci_pipeline_variables_minimum_override_role") {
		stringVal := d.Get("ci_pipeline_variables_minimum_override_role").(string)
		options.CIPipelineVariablesMinimumOverrideRole = gitlab.Ptr(stringVal)
	}

	if d.HasChange("merge_pipelines_enabled") {
		options.MergePipelinesEnabled = gitlab.Ptr(d.Get("merge_pipelines_enabled").(bool))
	}

	if d.HasChange("merge_trains_enabled") {
		options.MergeTrainsEnabled = gitlab.Ptr(d.Get("merge_trains_enabled").(bool))
	}

	if d.HasChange("resolve_outdated_diff_discussions") {
		options.ResolveOutdatedDiffDiscussions = gitlab.Ptr(d.Get("resolve_outdated_diff_discussions").(bool))
	}

	if d.HasChange("analytics_access_level") {
		options.AnalyticsAccessLevel = stringToAccessControlValue(d.Get("analytics_access_level").(string))
	}

	if d.HasChange("auto_cancel_pending_pipelines") {
		options.AutoCancelPendingPipelines = gitlab.Ptr(d.Get("auto_cancel_pending_pipelines").(string))
	}

	if d.HasChange("auto_devops_deploy_strategy") {
		options.AutoDevopsDeployStrategy = gitlab.Ptr(d.Get("auto_devops_deploy_strategy").(string))
	}

	if d.HasChange("auto_devops_enabled") {
		options.AutoDevopsEnabled = gitlab.Ptr(d.Get("auto_devops_enabled").(bool))
	}

	if d.HasChange("autoclose_referenced_issues") {
		options.AutocloseReferencedIssues = gitlab.Ptr(d.Get("autoclose_referenced_issues").(bool))
	}

	if d.HasChange("build_git_strategy") {
		options.BuildGitStrategy = gitlab.Ptr(d.Get("build_git_strategy").(string))
	}

	if d.HasChange("build_timeout") {
		options.BuildTimeout = gitlab.Ptr(d.Get("build_timeout").(int))
	}

	if d.HasChange("builds_access_level") {
		options.BuildsAccessLevel = stringToAccessControlValue(d.Get("builds_access_level").(string))
	}

	if d.HasChange("container_expiration_policy") {
		options.ContainerExpirationPolicyAttributes = expandContainerExpirationPolicyAttributes(d)
	}

	if d.HasChange("container_registry_access_level") {
		options.ContainerRegistryAccessLevel = stringToAccessControlValue(d.Get("container_registry_access_level").(string))
	}

	if d.HasChange("emails_enabled") {
		options.EmailsEnabled = gitlab.Ptr(d.Get("emails_enabled").(bool))
	}

	if d.HasChange("external_authorization_classification_label") {
		options.ExternalAuthorizationClassificationLabel = gitlab.Ptr(d.Get("external_authorization_classification_label").(string))
	}

	if d.HasChange("forking_access_level") {
		options.ForkingAccessLevel = stringToAccessControlValue(d.Get("forking_access_level").(string))
	}

	if d.HasChange("issues_access_level") {
		options.IssuesAccessLevel = stringToAccessControlValue(d.Get("issues_access_level").(string))
	}

	if d.HasChange("merge_requests_access_level") {
		options.MergeRequestsAccessLevel = stringToAccessControlValue(d.Get("merge_requests_access_level").(string))
	}

	// Ignore deprecated public_builds in favor of public_jobs.
	if d.HasChange("public_jobs") {
		options.PublicBuilds = gitlab.Ptr(d.Get("public_jobs").(bool))
	} else if d.HasChange("public_builds") {
		options.PublicBuilds = gitlab.Ptr(d.Get("public_builds").(bool))
	}

	if d.HasChange("repository_access_level") {
		options.RepositoryAccessLevel = stringToAccessControlValue(d.Get("repository_access_level").(string))
	}

	if d.HasChange("repository_storage") {
		options.RepositoryStorage = gitlab.Ptr(d.Get("repository_storage").(string))
	}

	if d.HasChange("requirements_access_level") {
		options.RequirementsAccessLevel = stringToAccessControlValue(d.Get("requirements_access_level").(string))
	}

	if d.HasChange("security_and_compliance_access_level") {
		options.SecurityAndComplianceAccessLevel = stringToAccessControlValue(d.Get("security_and_compliance_access_level").(string))
	}

	if d.HasChange("snippets_access_level") {
		options.SnippetsAccessLevel = stringToAccessControlValue(d.Get("snippets_access_level").(string))
	}

	if d.HasChange("suggestion_commit_message") {
		options.SuggestionCommitMessage = gitlab.Ptr(d.Get("suggestion_commit_message").(string))
	}

	if d.HasChange("topics") {
		options.Topics = stringSetToStringSlice(d.Get("topics").(*schema.Set))
	}

	if d.HasChange("wiki_access_level") {
		options.WikiAccessLevel = stringToAccessControlValue(d.Get("wiki_access_level").(string))
	}

	if d.HasChange("squash_commit_template") {
		options.SquashCommitTemplate = gitlab.Ptr(d.Get("squash_commit_template").(string))
	}

	if d.HasChange("merge_commit_template") {
		options.MergeCommitTemplate = gitlab.Ptr(d.Get("merge_commit_template").(string))
	}

	if d.HasChange("ci_default_git_depth") {
		options.CIDefaultGitDepth = gitlab.Ptr(d.Get("ci_default_git_depth").(int))
	}

	if d.HasChange("ci_separated_caches") {
		options.CISeperateCache = gitlab.Ptr(d.Get("ci_separated_caches").(bool))
	}

	if d.HasChange("keep_latest_artifact") {
		options.KeepLatestArtifact = gitlab.Ptr(d.Get("keep_latest_artifact").(bool))
	}

	if d.HasChange("mr_default_target_self") {
		options.MergeRequestDefaultTargetSelf = gitlab.Ptr(d.Get("mr_default_target_self").(bool))
	}

	if d.HasChange("releases_access_level") {
		options.ReleasesAccessLevel = stringToAccessControlValue(d.Get("releases_access_level").(string))
	}

	if d.HasChange("environments_access_level") {
		options.EnvironmentsAccessLevel = stringToAccessControlValue(d.Get("environments_access_level").(string))
	}

	if d.HasChange("feature_flags_access_level") {
		options.FeatureFlagsAccessLevel = stringToAccessControlValue(d.Get("feature_flags_access_level").(string))
	}

	if d.HasChange("infrastructure_access_level") {
		options.InfrastructureAccessLevel = stringToAccessControlValue(d.Get("infrastructure_access_level").(string))
	}

	if d.HasChange("monitor_access_level") {
		options.MonitorAccessLevel = stringToAccessControlValue(d.Get("monitor_access_level").(string))
	}

	if d.HasChange("model_experiments_access_level") {
		options.ModelExperimentsAccessLevel = stringToAccessControlValue(d.Get("model_experiments_access_level").(string))
	}

	if d.HasChange("model_registry_access_level") {
		options.ModelRegistryAccessLevel = stringToAccessControlValue(d.Get("model_registry_access_level").(string))
	}

	avatar, err := handleAvatarOnUpdate(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if avatar != nil {
		options.Avatar = &gitlab.ProjectAvatar{
			Filename: avatar.Filename,
			Image:    avatar.Image,
		}
	}

	var project *gitlab.Project
	if *options != (gitlab.EditProjectOptions{}) {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab project %s", d.Id()))
		project, _, err = client.Projects.EditProject(d.Id(), options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// If we don't have the project from the EditProject call, retrieve the project here so we have the full
	// path for further calls.
	if project == nil {
		project, _, err = client.Projects.GetProject(d.Id(), nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// If enabling Secret Push Detection, then update that value on the project via
	// GraphQL
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if changed := d.HasChange("pre_receive_secret_detection_enabled"); changed {
		val := d.Get("pre_receive_secret_detection_enabled").(bool)
		err := updateProjectSecretDetectionValue(ctx, client, project.PathWithNamespace, val)
		if err != nil {
			return diag.Errorf("Error updating Secret Push Detection on Project %s: %v", d.Id(), err)
		}
	}

	if d.HasChange("forked_from_project_id") {
		oldRaw, newRaw := d.GetChange("forked_from_project_id")
		oldValue, newValue := oldRaw.(int), newRaw.(int)
		var createRelation, removeRelation bool

		if newValue != 0 && oldValue != 0 {
			// change fork relation
			removeRelation = true
			createRelation = true
		} else if newValue != 0 {
			// create fork relation
			createRelation = true
		} else if newValue == 0 {
			removeRelation = true
		}

		if removeRelation {
			// Remove fork relation
			if _, err := client.Projects.DeleteProjectForkRelation(d.Id(), gitlab.WithContext(ctx)); err != nil {
				return diag.Errorf("unable to remove fork relation from project %q: %v", d.Id(), err)
			}
		}

		// Add fork relationship
		if createRelation {
			// Add fork relation
			if _, _, err := client.Projects.CreateProjectForkRelation(d.Id(), newValue, gitlab.WithContext(ctx)); err != nil {
				return diag.Errorf("unable to add fork relation to project %q (to project %d): %v", d.Id(), newValue, err)
			}
		}
	}

	if *transferOptions != (gitlab.TransferProjectOptions{}) {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] transferring project %s to namespace %d", d.Id(), transferOptions.Namespace))
		_, _, err := client.Projects.TransferProject(d.Id(), transferOptions, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("archived") {
		if d.Get("archived").(bool) {
			if _, _, err := client.Projects.ArchiveProject(d.Id(), gitlab.WithContext(ctx)); err != nil {
				return diag.Errorf("project %q could not be archived: %s", d.Id(), err)
			}
		} else {
			if _, _, err := client.Projects.UnarchiveProject(d.Id(), gitlab.WithContext(ctx)); err != nil {
				return diag.Errorf("project %q could not be unarchived: %s", d.Id(), err)
			}
		}
	}

	if d.HasChange("push_rules") {
		err := editOrAddPushRules(ctx, client, d.Id(), d)
		if err != nil {
			if api.Is404(err) {
				tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Failed to get push rules for project %q: %v", d.Id(), err))
				return diag.Errorf("Project push rules are not supported in your version of GitLab")
			}
			return diag.Errorf("Failed to edit push rules for project %q: %s", d.Id(), err)
		}
	}

	return resourceGitlabProjectRead(ctx, d, meta)
}

func resourceGitlabProjectDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	if !d.Get("archive_on_destroy").(bool) {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete gitlab project %s", d.Id()))
		_, err := client.Projects.DeleteProject(d.Id(), nil, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		// Wait for the project to be deleted.
		// Deleting a project in gitlab is async.
		stateConf := &retry.StateChangeConf{
			Pending: []string{"Deleting"},
			Target:  []string{"Deleted"},
			Refresh: func() (interface{}, string, error) {
				out, _, err := client.Projects.GetProject(d.Id(), nil, gitlab.WithContext(ctx))
				if err != nil {
					if api.Is404(err) {
						return out, "Deleted", nil
					}
					tflog.Debug(ctx, fmt.Sprintf("[ERROR] Received error: %#v", err))
					return out, "Error", err
				}
				if out.MarkedForDeletionAt != nil {
					// Represents a Gitlab EE soft-delete
					return out, "Deleted", nil
				}
				return out, "Deleting", nil
			},

			Timeout:    d.Timeout(schema.TimeoutDelete),
			MinTimeout: 3 * time.Second,
			Delay:      5 * time.Second,
		}

		_, err = stateConf.WaitForStateContext(ctx)
		if err != nil {
			return diag.Errorf("error waiting for project (%s) to become deleted: %s", d.Id(), err)
		}

	} else {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Archive gitlab project %s", d.Id()))
		_, _, err := client.Projects.ArchiveProject(d.Id(), gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func editOrAddPushRules(ctx context.Context, client *gitlab.Client, projectID string, d *schema.ResourceData) error {
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Editing push rules for project %q", projectID))

	pushRules, _, err := client.Projects.GetProjectPushRules(d.Id(), gitlab.WithContext(ctx))
	// NOTE: push rules id `0` indicates that there haven't been any push rules set.
	if err != nil || pushRules.ID == 0 {
		if addOptions := expandAddProjectPushRuleOptions(d); (gitlab.AddProjectPushRuleOptions{}) != addOptions {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Creating new push rules for project %q", projectID))
			_, _, err = client.Projects.AddProjectPushRule(projectID, &addOptions, gitlab.WithContext(ctx))
			if err != nil {
				return err
			}
		} else {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Don't create new push rules for defaults for project %q", projectID))
		}

		return nil
	}

	editOptions := expandEditProjectPushRuleOptions(d, pushRules)
	if (gitlab.EditProjectPushRuleOptions{}) != editOptions {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Editing existing push rules for project %q", projectID))
		_, _, err = client.Projects.EditProjectPushRule(projectID, &editOptions, gitlab.WithContext(ctx))
		if err != nil {
			return err
		}
	} else {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Don't edit existing push rules for defaults for project %q", projectID))
	}

	return nil
}

func expandEditProjectPushRuleOptions(d *schema.ResourceData, currentPushRules *gitlab.ProjectPushRules) gitlab.EditProjectPushRuleOptions {
	options := gitlab.EditProjectPushRuleOptions{}

	if d.Get("push_rules.0.author_email_regex") != currentPushRules.AuthorEmailRegex {
		options.AuthorEmailRegex = gitlab.Ptr(d.Get("push_rules.0.author_email_regex").(string))
	}

	if d.Get("push_rules.0.branch_name_regex") != currentPushRules.BranchNameRegex {
		options.BranchNameRegex = gitlab.Ptr(d.Get("push_rules.0.branch_name_regex").(string))
	}

	if d.Get("push_rules.0.commit_message_regex") != currentPushRules.CommitMessageRegex {
		options.CommitMessageRegex = gitlab.Ptr(d.Get("push_rules.0.commit_message_regex").(string))
	}

	if d.Get("push_rules.0.commit_message_negative_regex") != currentPushRules.CommitMessageNegativeRegex {
		options.CommitMessageNegativeRegex = gitlab.Ptr(d.Get("push_rules.0.commit_message_negative_regex").(string))
	}

	if d.Get("push_rules.0.file_name_regex") != currentPushRules.FileNameRegex {
		options.FileNameRegex = gitlab.Ptr(d.Get("push_rules.0.file_name_regex").(string))
	}

	if d.Get("push_rules.0.commit_committer_check") != currentPushRules.CommitCommitterCheck {
		options.CommitCommitterCheck = gitlab.Ptr(d.Get("push_rules.0.commit_committer_check").(bool))
	}

	if d.Get("push_rules.0.commit_committer_name_check") != currentPushRules.CommitCommitterNameCheck {
		options.CommitCommitterNameCheck = gitlab.Ptr(d.Get("push_rules.0.commit_committer_name_check").(bool))
	}

	if d.Get("push_rules.0.deny_delete_tag") != currentPushRules.DenyDeleteTag {
		options.DenyDeleteTag = gitlab.Ptr(d.Get("push_rules.0.deny_delete_tag").(bool))
	}

	if d.Get("push_rules.0.member_check") != currentPushRules.MemberCheck {
		options.MemberCheck = gitlab.Ptr(d.Get("push_rules.0.member_check").(bool))
	}

	if d.Get("push_rules.0.prevent_secrets") != currentPushRules.PreventSecrets {
		options.PreventSecrets = gitlab.Ptr(d.Get("push_rules.0.prevent_secrets").(bool))
	}

	if d.Get("push_rules.0.reject_unsigned_commits") != currentPushRules.RejectUnsignedCommits {
		options.RejectUnsignedCommits = gitlab.Ptr(d.Get("push_rules.0.reject_unsigned_commits").(bool))
	}

	if d.Get("push_rules.0.reject_non_dco_commits") != currentPushRules.RejectNonDCOCommits {
		options.RejectNonDCOCommits = gitlab.Ptr(d.Get("push_rules.0.reject_non_dco_commits").(bool))
	}

	if d.Get("push_rules.0.max_file_size") != currentPushRules.MaxFileSize {
		options.MaxFileSize = gitlab.Ptr(d.Get("push_rules.0.max_file_size").(int))
	}

	return options
}

func expandAddProjectPushRuleOptions(d *schema.ResourceData) gitlab.AddProjectPushRuleOptions {
	options := gitlab.AddProjectPushRuleOptions{}

	if v, ok := d.GetOk("push_rules.0.author_email_regex"); ok {
		options.AuthorEmailRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.branch_name_regex"); ok {
		options.BranchNameRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.commit_message_regex"); ok {
		options.CommitMessageRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.commit_message_negative_regex"); ok {
		options.CommitMessageNegativeRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.file_name_regex"); ok {
		options.FileNameRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("push_rules.0.commit_committer_check"); ok {
		options.CommitCommitterCheck = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.commit_committer_name_check"); ok {
		options.CommitCommitterNameCheck = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.deny_delete_tag"); ok {
		options.DenyDeleteTag = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.member_check"); ok {
		options.MemberCheck = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.prevent_secrets"); ok {
		options.PreventSecrets = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.reject_unsigned_commits"); ok {
		options.RejectUnsignedCommits = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.reject_non_dco_commits"); ok {
		options.RejectNonDCOCommits = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("push_rules.0.max_file_size"); ok {
		options.MaxFileSize = gitlab.Ptr(v.(int))
	}

	return options
}

func flattenProjectPushRules(pushRules *gitlab.ProjectPushRules) (values []map[string]interface{}) {
	if pushRules == nil {
		return []map[string]interface{}{}
	}

	return []map[string]interface{}{
		{
			"author_email_regex":            pushRules.AuthorEmailRegex,
			"branch_name_regex":             pushRules.BranchNameRegex,
			"commit_message_regex":          pushRules.CommitMessageRegex,
			"commit_message_negative_regex": pushRules.CommitMessageNegativeRegex,
			"file_name_regex":               pushRules.FileNameRegex,
			"commit_committer_check":        pushRules.CommitCommitterCheck,
			"commit_committer_name_check":   pushRules.CommitCommitterNameCheck,
			"deny_delete_tag":               pushRules.DenyDeleteTag,
			"member_check":                  pushRules.MemberCheck,
			"prevent_secrets":               pushRules.PreventSecrets,
			"reject_unsigned_commits":       pushRules.RejectUnsignedCommits,
			"reject_non_dco_commits":        pushRules.RejectNonDCOCommits,
			"max_file_size":                 pushRules.MaxFileSize,
		},
	}
}

func flattenContainerExpirationPolicy(policy *gitlab.ContainerExpirationPolicy) (values []map[string]interface{}) {
	if policy == nil {
		return
	}

	// We're setting both name_regex and name_regex_delete to keep backwards compatibility.
	values = []map[string]interface{}{
		{
			"cadence":           policy.Cadence,
			"keep_n":            policy.KeepN,
			"older_than":        policy.OlderThan,
			"name_regex_delete": policy.NameRegex,
			"name_regex":        policy.NameRegex,
			"name_regex_keep":   policy.NameRegexKeep,
			"enabled":           policy.Enabled,
		},
	}

	if policy.NextRunAt != nil {
		values[0]["next_run_at"] = policy.NextRunAt.Format(time.RFC3339)
	}
	return values
}

func expandContainerExpirationPolicyAttributes(d *schema.ResourceData) *gitlab.ContainerExpirationPolicyAttributes {
	policy := gitlab.ContainerExpirationPolicyAttributes{}

	if v, ok := d.GetOk("container_expiration_policy.0.cadence"); ok {
		policy.Cadence = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("container_expiration_policy.0.keep_n"); ok {
		policy.KeepN = gitlab.Ptr(v.(int))
	}

	if v, ok := d.GetOk("container_expiration_policy.0.older_than"); ok {
		policy.OlderThan = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("container_expiration_policy.0.name_regex_delete"); ok {
		policy.NameRegexDelete = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("container_expiration_policy.0.name_regex_keep"); ok {
		policy.NameRegexKeep = gitlab.Ptr(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("container_expiration_policy.0.enabled"); ok {
		policy.Enabled = gitlab.Ptr(v.(bool))
	}

	return &policy
}

func namespaceOrPathChanged(ctx context.Context, d *schema.ResourceDiff, meta interface{}) bool {
	return d.HasChange("namespace_id") || d.HasChange("path")
}

func expectDefaultBranchProtection(ctx context.Context, client *gitlab.Client, project *gitlab.Project) (bool, error) {
	// If the project is part of a group it may have default branch protection disabled for its projects
	if project.Namespace.Kind == "group" {
		group, _, err := client.Groups.GetGroup(project.Namespace.ID, nil, gitlab.WithContext(ctx))
		if err != nil {
			return false, err
		}

		// nolint:staticcheck // SA1019 ignore deprecated DefaultBranchProtection
		return group.DefaultBranchProtection != 0, nil
	}

	isAdmin, err := api.IsCurrentUserAdmin(ctx, client)
	if err != nil {
		return false, fmt.Errorf("failed to check if user is admin to verify is default branch protection is enabled on instance-level: %w", err)
	}

	if isAdmin {
		// If the project is not part of a group it may have default branch protection disabled because of the instance-wide application settings
		settings, _, err := api.GetSettings(client, gitlab.WithContext(ctx))
		if err != nil {
			return false, err
		}

		return settings.DefaultBranchProtection != 0, nil
	}

	// NOTE: for the lack of a better solution (at least for now), we assume that the default branch protection is NOT disabled on instance-level.
	//       To override this behavior it's best to set `skip_wait_for_default_branch_protection = true` in the resource config.
	return true, nil
}

func constructImportUrl(importURL string, username string, password string) (string, error) {
	if username == "" && password == "" {
		return importURL, nil
	}

	parsedURL, err := url.Parse(importURL)
	if err != nil {
		return "", fmt.Errorf("the given `import_url` is not a valid URL: %s", err)
	}

	credentials := url.UserPassword(username, password)
	if parsedURL.User != nil && parsedURL.User.String() != credentials.String() {
		return "", fmt.Errorf("the `import_url` already contains credentials which don't match the credentials from `import_url_username` and `import_url_password`")
	}

	parsedURL.User = credentials

	return parsedURL.String(), nil
}

// Create a project. Extracted from the main `create` function for readability and to differentiate from
// creating a _forked_ project.
func createProject(ctx context.Context, d *schema.ResourceData, client *gitlab.Client) (*gitlab.Project, diag.Diagnostics) {

	options := &gitlab.CreateProjectOptions{
		Name: gitlab.Ptr(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("build_coverage_regex"); ok {
		options.BuildCoverageRegex = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("path"); ok {
		options.Path = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("namespace_id"); ok {
		options.NamespaceID = gitlab.Ptr(v.(int))
	}

	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("default_branch"); ok {
		options.DefaultBranch = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("tags"); ok {
		// TODO: Remove TagList on next breaking change. TagList and Topics aren't completely synonymous.
		// nolint:staticcheck // SA1019
		options.TagList = stringSetToStringSlice(v.(*schema.Set))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("container_registry_enabled"); ok {
		// nolint:staticcheck // SA1019
		options.ContainerRegistryEnabled = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("initialize_with_readme"); ok {
		options.InitializeWithReadme = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("pipelines_enabled"); ok {
		// nolint:staticcheck // SA1019
		options.JobsEnabled = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("import_url"); ok {
		importURL, err := constructImportUrl(v.(string), d.Get("import_url_username").(string), d.Get("import_url_password").(string))
		if err != nil {
			return nil, diag.Errorf("Unable to construct import URL for API: %s", err)
		}
		options.ImportURL = gitlab.Ptr(importURL)
	}

	if v, ok := d.GetOk("template_name"); ok {
		options.TemplateName = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("template_project_id"); ok {
		options.TemplateProjectID = gitlab.Ptr(v.(int))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("use_custom_template"); ok {
		// There is currently a bug where `use_custom_template` returns a 500 if the
		// value is set to `false`, requiring it to be set to `null` instead to work.
		// As a result, only apply this value if it's "true"
		if v.(bool) {
			options.UseCustomTemplate = gitlab.Ptr(v.(bool))
		}
	}

	if v, ok := d.GetOk("group_with_project_templates_id"); ok {
		options.GroupWithProjectTemplatesID = gitlab.Ptr(v.(int))
	}

	if v, ok := d.GetOk("pages_access_level"); ok {
		options.PagesAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("ci_config_path"); ok {
		options.CIConfigPath = gitlab.Ptr(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("resolve_outdated_diff_discussions"); ok {
		options.ResolveOutdatedDiffDiscussions = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("analytics_access_level"); ok {
		options.AnalyticsAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("auto_cancel_pending_pipelines"); ok {
		options.AutoCancelPendingPipelines = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("auto_devops_deploy_strategy"); ok {
		options.AutoDevopsDeployStrategy = gitlab.Ptr(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("auto_devops_enabled"); ok {
		options.AutoDevopsEnabled = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("autoclose_referenced_issues"); ok {
		options.AutocloseReferencedIssues = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("build_git_strategy"); ok {
		options.BuildGitStrategy = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("build_timeout"); ok {
		options.BuildTimeout = gitlab.Ptr(v.(int))
	}

	if v, ok := d.GetOk("builds_access_level"); ok {
		options.BuildsAccessLevel = stringToAccessControlValue(v.(string))
	}

	if _, ok := d.GetOk("container_expiration_policy"); ok {
		options.ContainerExpirationPolicyAttributes = expandContainerExpirationPolicyAttributes(d)
	}

	if v, ok := d.GetOk("container_registry_access_level"); ok {
		options.ContainerRegistryAccessLevel = stringToAccessControlValue(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("emails_enabled"); ok {
		options.EmailsEnabled = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("external_authorization_classification_label"); ok {
		options.ExternalAuthorizationClassificationLabel = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("forking_access_level"); ok {
		options.ForkingAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("issues_access_level"); ok {
		options.IssuesAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("merge_requests_access_level"); ok {
		options.MergeRequestsAccessLevel = stringToAccessControlValue(v.(string))
	}

	// Ignore deprecated public_builds in favor of public_jobs.
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("public_jobs"); ok {
		options.PublicBuilds = gitlab.Ptr(v.(bool))
	} else if v, ok := d.GetOkExists("public_builds"); ok {
		options.PublicBuilds = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("repository_access_level"); ok {
		options.RepositoryAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("repository_storage"); ok {
		options.RepositoryStorage = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("requirements_access_level"); ok {
		options.RequirementsAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("security_and_compliance_access_level"); ok {
		options.SecurityAndComplianceAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("snippets_access_level"); ok {
		options.SnippetsAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("suggestion_commit_message"); ok {
		options.SuggestionCommitMessage = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("topics"); ok {
		options.Topics = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("wiki_access_level"); ok {
		options.WikiAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("squash_commit_template"); ok {
		options.SquashCommitTemplate = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("merge_commit_template"); ok {
		options.MergeCommitTemplate = gitlab.Ptr(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("request_access_enabled"); ok {
		options.RequestAccessEnabled = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("issues_enabled"); ok {
		options.IssuesEnabled = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("merge_requests_enabled"); ok {
		// nolint:staticcheck // SA1019
		options.MergeRequestsEnabled = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("approvals_before_merge"); ok {
		options.ApprovalsBeforeMerge = gitlab.Ptr(v.(int))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("wiki_enabled"); ok {
		// nolint:staticcheck // SA1019
		options.WikiEnabled = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("snippets_enabled"); ok {
		// nolint:staticcheck // SA1019
		options.SnippetsEnabled = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("lfs_enabled"); ok {
		options.LFSEnabled = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("visibility_level"); ok {
		options.Visibility = stringToVisibilityLevel(v.(string))
	}

	if v, ok := d.GetOk("merge_method"); ok {
		options.MergeMethod = stringToMergeMethod(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("only_allow_merge_if_pipeline_succeeds"); ok {
		options.OnlyAllowMergeIfPipelineSucceeds = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("only_allow_merge_if_all_discussions_are_resolved"); ok {
		options.OnlyAllowMergeIfAllDiscussionsAreResolved = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("allow_merge_on_skipped_pipeline"); ok {
		options.AllowMergeOnSkippedPipeline = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("shared_runners_enabled"); ok {
		options.SharedRunnersEnabled = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("group_runners_enabled"); ok {
		options.GroupRunnersEnabled = gitlab.Bool(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("remove_source_branch_after_merge"); ok {
		options.RemoveSourceBranchAfterMerge = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("packages_enabled"); ok {
		options.PackagesEnabled = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("printing_merge_request_link_enabled"); ok {
		options.PrintingMergeRequestLinkEnabled = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("mirror"); ok {
		options.Mirror = gitlab.Ptr(v.(bool))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("mirror_trigger_builds"); ok {
		options.MirrorTriggerBuilds = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("ci_config_path"); ok {
		options.CIConfigPath = gitlab.Ptr(v.(string))
	}

	if supportsSquashOption, err := api.IsGitLabVersionAtLeast(ctx, client, "14.1")(); err != nil {
		return nil, diag.FromErr(err)
	} else if supportsSquashOption {
		if v, ok := d.GetOk("squash_option"); ok {
			options.SquashOption = stringToSquashOptionValue(v.(string))
		}
	}

	avatar, err := handleAvatarOnCreate(d)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	if avatar != nil {
		options.Avatar = &gitlab.ProjectAvatar{
			Filename: avatar.Filename,
			Image:    avatar.Image,
		}
	}

	if v, ok := d.GetOk("releases_access_level"); ok {
		options.ReleasesAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("environments_access_level"); ok {
		options.EnvironmentsAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("feature_flags_access_level"); ok {
		options.FeatureFlagsAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("infrastructure_access_level"); ok {
		options.InfrastructureAccessLevel = stringToAccessControlValue(v.(string))
	}

	if v, ok := d.GetOk("monitor_access_level"); ok {
		options.MonitorAccessLevel = stringToAccessControlValue(v.(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab project %q", *options.Name))

	project, _, err := client.Projects.CreateProject(options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return project, nil
}

// Create a forked project. Extracted from the main `create` function for readability and to differentiate from
// creating a "normal" project
func createForkedProject(ctx context.Context, forkedFromProjectID int, d *schema.ResourceData, client *gitlab.Client) (*gitlab.Project, diag.Diagnostics) {

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] forking project %d", forkedFromProjectID))

	options := gitlab.ForkProjectOptions{}
	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.Ptr(v.(string))
	}
	if v, ok := d.GetOk("name"); ok {
		options.Name = gitlab.Ptr(v.(string))
	}
	if v, ok := d.GetOk("path"); ok {
		options.Path = gitlab.Ptr(v.(string))
	}
	if v, ok := d.GetOk("namespace_id"); ok {
		options.NamespaceID = gitlab.Ptr(v.(int))
	}
	if v, ok := d.GetOk("visibility_level"); ok {
		options.Visibility = stringToVisibilityLevel(v.(string))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("mr_default_target_self"); ok {
		options.MergeRequestDefaultTargetSelf = gitlab.Ptr(v.(bool))
	}

	var err error
	project, _, err := client.Projects.ForkProject(forkedFromProjectID, &options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, diag.Errorf("Unable to fork project %d: %v", forkedFromProjectID, err)
	}

	return project, nil
}

// There are options during the `resourceGitlabProjectCreate` operation that cannot be set because they're
// only supported in the `Update` API. This function handles updating the `editPojectOptions` to include
// those options.
func updatePostCreateEditOptions(ctx context.Context, editProjectOptions *gitlab.EditProjectOptions, d *schema.ResourceData, client *gitlab.Client, project *gitlab.Project) diag.Diagnostics {

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("mirror_overwrites_diverged_branches"); ok {
		editProjectOptions.MirrorOverwritesDivergedBranches = gitlab.Ptr(v.(bool))
		importURL, err := constructImportUrl(d.Get("import_url").(string), d.Get("import_url_username").(string), d.Get("import_url_password").(string))
		if err != nil {
			return diag.Errorf("Unable to construct import URL for API: %s", err)
		}
		editProjectOptions.ImportURL = gitlab.Ptr(importURL)
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("only_mirror_protected_branches"); ok {
		editProjectOptions.OnlyMirrorProtectedBranches = gitlab.Ptr(v.(bool))
		importURL, err := constructImportUrl(d.Get("import_url").(string), d.Get("import_url_username").(string), d.Get("import_url_password").(string))
		if err != nil {
			return diag.Errorf("Unable to construct import URL for API: %s", err)
		}
		editProjectOptions.ImportURL = gitlab.Ptr(importURL)
	}

	if v, ok := d.GetOk("description"); ok {
		editProjectOptions.Description = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("issues_template"); ok {
		editProjectOptions.IssuesTemplate = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("merge_requests_template"); ok {
		editProjectOptions.MergeRequestsTemplate = gitlab.Ptr(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("merge_pipelines_enabled"); ok {
		editProjectOptions.MergePipelinesEnabled = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("merge_trains_enabled"); ok {
		editProjectOptions.MergeTrainsEnabled = gitlab.Ptr(v.(bool))
	}

	if v, ok := d.GetOk("ci_default_git_depth"); ok {
		editProjectOptions.CIDefaultGitDepth = gitlab.Ptr(v.(int))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("ci_forward_deployment_enabled"); ok {
		editProjectOptions.CIForwardDeploymentEnabled = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("ci_separated_caches"); ok {
		editProjectOptions.CISeperateCache = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("keep_latest_artifact"); ok {
		editProjectOptions.KeepLatestArtifact = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("restrict_user_defined_variables"); ok {
		editProjectOptions.RestrictUserDefinedVariables = gitlab.Ptr(v.(bool))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("ci_restrict_pipeline_cancellation_role"); ok {
		editProjectOptions.CIRestrictPipelineCancellationRole = gitlab.Ptr(api.AccessControlLevelValueToName(v.(string)))
	}
	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("ci_pipeline_variables_minimum_override_role"); ok {
		editProjectOptions.CIPipelineVariablesMinimumOverrideRole = gitlab.Ptr(v.(string))
	}

	// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
	// lintignore: XR001 // TODO: replace with alternative for GetOkExists
	if v, ok := d.GetOkExists("allow_pipeline_trigger_approve_deployment"); ok {
		editProjectOptions.AllowPipelineTriggerApproveDeployment = gitlab.Ptr(v.(bool))
	}

	// If we forked the project we could apply lots of the attributes,
	// thus, we have to do this now.
	if project.ForkedFromProject != nil {

		if v, ok := d.GetOk("default_branch"); ok {
			editProjectOptions.DefaultBranch = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("merge_method"); ok {
			editProjectOptions.MergeMethod = stringToMergeMethod(v.(string))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("only_allow_merge_if_pipeline_succeeds"); ok {
			editProjectOptions.OnlyAllowMergeIfPipelineSucceeds = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("only_allow_merge_if_all_discussions_are_resolved"); ok {
			editProjectOptions.OnlyAllowMergeIfAllDiscussionsAreResolved = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("allow_merge_on_skipped_pipeline"); ok {
			editProjectOptions.AllowMergeOnSkippedPipeline = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("request_access_enabled"); ok {
			editProjectOptions.RequestAccessEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("issues_enabled"); ok {
			editProjectOptions.IssuesEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("merge_requests_enabled"); ok {
			editProjectOptions.MergeRequestsEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("pipelines_enabled"); ok {
			// nolint:staticcheck // SA1019
			editProjectOptions.JobsEnabled = gitlab.Ptr(v.(bool))
		}

		if v, ok := d.GetOk("approvals_before_merge"); ok {
			editProjectOptions.ApprovalsBeforeMerge = gitlab.Ptr(v.(int))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("wiki_enabled"); ok {
			// nolint:staticcheck // SA1019
			editProjectOptions.WikiEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("snippets_enabled"); ok {
			// nolint:staticcheck // SA1019
			editProjectOptions.SnippetsEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("shared_runners_enabled"); ok {
			editProjectOptions.SharedRunnersEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("group_runners_enabled"); ok {
			editProjectOptions.GroupRunnersEnabled = gitlab.Bool(v.(bool))
		}

		if v, ok := d.GetOk("tags"); ok {
			// TODO: Remove TagList on next breaking change. TagList and Topics aren't completely synonymous.
			// nolint:staticcheck // SA1019
			editProjectOptions.TagList = stringSetToStringSlice(v.(*schema.Set))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("container_registry_enabled"); ok {
			editProjectOptions.ContainerRegistryEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("lfs_enabled"); ok {
			editProjectOptions.LFSEnabled = gitlab.Ptr(v.(bool))
		}

		if supportsSquashOption, err := api.IsGitLabVersionAtLeast(ctx, client, "14.1")(); err != nil {
			return diag.FromErr(err)
		} else if supportsSquashOption {
			if v, ok := d.GetOk("squash_option"); ok {
				editProjectOptions.SquashOption = stringToSquashOptionValue(v.(string))
			}
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("remove_source_branch_after_merge"); ok {
			editProjectOptions.RemoveSourceBranchAfterMerge = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("printing_merge_request_link_enabled"); ok {
			editProjectOptions.PrintingMergeRequestLinkEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("packages_enabled"); ok {
			editProjectOptions.PackagesEnabled = gitlab.Ptr(v.(bool))
		}

		if v, ok := d.GetOk("pages_access_level"); ok {
			editProjectOptions.PagesAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("build_coverage_regex"); ok {
			editProjectOptions.IssuesTemplate = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("ci_config_path"); ok {
			editProjectOptions.CIConfigPath = gitlab.Ptr(v.(string))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("ci_forward_deployment_enabled"); ok {
			editProjectOptions.CIForwardDeploymentEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("resolve_outdated_diff_discussions"); ok {
			editProjectOptions.ResolveOutdatedDiffDiscussions = gitlab.Ptr(v.(bool))
		}

		if v, ok := d.GetOk("analytics_access_level"); ok {
			editProjectOptions.AnalyticsAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("auto_cancel_pending_pipelines"); ok {
			editProjectOptions.AutoCancelPendingPipelines = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("auto_devops_deploy_strategy"); ok {
			editProjectOptions.AutoDevopsDeployStrategy = gitlab.Ptr(v.(string))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("auto_devops_enabled"); ok {
			editProjectOptions.AutoDevopsEnabled = gitlab.Ptr(v.(bool))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("autoclose_referenced_issues"); ok {
			editProjectOptions.AutocloseReferencedIssues = gitlab.Ptr(v.(bool))
		}

		if v, ok := d.GetOk("build_git_strategy"); ok {
			editProjectOptions.BuildGitStrategy = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("build_timeout"); ok {
			editProjectOptions.BuildTimeout = gitlab.Ptr(v.(int))
		}

		if v, ok := d.GetOk("builds_access_level"); ok {
			editProjectOptions.BuildsAccessLevel = stringToAccessControlValue(v.(string))
		}

		if _, ok := d.GetOk("container_expiration_policy"); ok {
			editProjectOptions.ContainerExpirationPolicyAttributes = expandContainerExpirationPolicyAttributes(d)
		}

		if v, ok := d.GetOk("container_registry_access_level"); ok {
			editProjectOptions.ContainerRegistryAccessLevel = stringToAccessControlValue(v.(string))
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("emails_enabled"); ok {
			editProjectOptions.EmailsEnabled = gitlab.Ptr(v.(bool))
		}

		if v, ok := d.GetOk("external_authorization_classification_label"); ok {
			editProjectOptions.ExternalAuthorizationClassificationLabel = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("forking_access_level"); ok {
			editProjectOptions.ForkingAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("issues_access_level"); ok {
			editProjectOptions.IssuesAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("merge_requests_access_level"); ok {
			editProjectOptions.MergeRequestsAccessLevel = stringToAccessControlValue(v.(string))
		}

		// Ignore deprecated public_builds in favor of public_jobs.
		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("public_jobs"); ok {
			editProjectOptions.PublicBuilds = gitlab.Ptr(v.(bool))
		} else if v, ok := d.GetOkExists("public_builds"); ok {
			editProjectOptions.PublicBuilds = gitlab.Ptr(v.(bool))
		}

		if v, ok := d.GetOk("repository_access_level"); ok {
			editProjectOptions.RepositoryAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("repository_storage"); ok {
			editProjectOptions.RepositoryStorage = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("requirements_access_level"); ok {
			editProjectOptions.RequirementsAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("security_and_compliance_access_level"); ok {
			editProjectOptions.SecurityAndComplianceAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("snippets_access_level"); ok {
			editProjectOptions.SnippetsAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("suggestion_commit_message"); ok {
			editProjectOptions.SuggestionCommitMessage = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("topics"); ok {
			editProjectOptions.Topics = stringSetToStringSlice(v.(*schema.Set))
		}

		if v, ok := d.GetOk("wiki_access_level"); ok {
			editProjectOptions.WikiAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("squash_commit_template"); ok {
			editProjectOptions.SquashCommitTemplate = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("merge_commit_template"); ok {
			editProjectOptions.MergeCommitTemplate = gitlab.Ptr(v.(string))
		}

		if v, ok := d.GetOk("import_url"); ok {
			importURL, err := constructImportUrl(v.(string), d.Get("import_url_username").(string), d.Get("import_url_password").(string))
			if err != nil {
				return diag.Errorf("Unable to construct import URL for API: %s", err)
			}
			editProjectOptions.ImportURL = gitlab.Ptr(importURL)
		}

		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("mirror"); ok {
			editProjectOptions.Mirror = gitlab.Ptr(v.(bool))
		}
		// nolint:staticcheck // SA1019 ignore deprecated GetOkExists
		// lintignore: XR001 // TODO: replace with alternative for GetOkExists
		if v, ok := d.GetOkExists("mirror_trigger_builds"); ok {
			editProjectOptions.MirrorTriggerBuilds = gitlab.Ptr(v.(bool))
		}

		avatar, err := handleAvatarOnUpdate(d)
		if err != nil {
			return diag.FromErr(err)
		}
		if avatar != nil {
			editProjectOptions.Avatar = &gitlab.ProjectAvatar{
				Filename: avatar.Filename,
				Image:    avatar.Image,
			}
		}

		if v, ok := d.GetOk("releases_access_level"); ok {
			editProjectOptions.ReleasesAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("environments_access_level"); ok {
			editProjectOptions.EnvironmentsAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("feature_flags_access_level"); ok {
			editProjectOptions.FeatureFlagsAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("infrastructure_access_level"); ok {
			editProjectOptions.InfrastructureAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("monitor_access_level"); ok {
			editProjectOptions.MonitorAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("model_experiments_access_level"); ok {
			editProjectOptions.ModelExperimentsAccessLevel = stringToAccessControlValue(v.(string))
		}

		if v, ok := d.GetOk("model_registry_access_level"); ok {
			editProjectOptions.ModelRegistryAccessLevel = stringToAccessControlValue(v.(string))
		}
	}

	return nil
}

func updateProjectSecretDetectionValue(ctx context.Context, client *gitlab.Client, projectPath string, input bool) error {

	// check if the version of GitLab is at least 17.3 before the call is attempted, and return with no error if lower than
	// 17.3 to skip the call
	versionOk, err := api.IsGitLabVersionAtLeast(ctx, client, "17.3")()
	if err != nil {
		return fmt.Errorf("failed to determine GitLab version when checking if Secret Push Detection is supported. Error: %v", err)
	}
	if !versionOk {
		// A earlier version of GitLab is being used, so exit before we try to enable the setting
		return nil
	}

	tflog.Debug(ctx, "Attempting to update Secrets Detection for project", map[string]interface{}{
		"project": projectPath,
		"value":   input,
	})

	// Create the query for enabling secrets detection via GraphQL
	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
		mutation {
			setPreReceiveSecretDetection(
				input: {
					enable: %v,
					namespacePath: "%s"
				}
			) {
				errors
			}
		}`, input, projectPath),
	}

	var response *updateSecretDetectionGraphQLResponse
	_, err = api.SendGraphQLRequest(ctx, client, query, &response)
	if err != nil {
		return err
	}

	// Check if errors were returned, and respond with the error message if they were
	if len(response.Data.SetPreReceiveSecretDetection.Errors) > 0 {
		return fmt.Errorf("error setting secrets detection: %s", response.Data.SetPreReceiveSecretDetection.Errors[0].Message)
	}
	return nil
}

// The GraphQL response struct when setting Secrets Detection
// Example payload:
//
//		{
//		  "data": {
//		    "setPreReceiveSecretDetection": {
//		      "errors": [{
//	          "message": "example error"
//	          }]
//		    }
//		  }
//		}
type updateSecretDetectionGraphQLResponse struct {
	Data struct {
		SetPreReceiveSecretDetection struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"setPreReceiveSecretDetection"`
	} `json:"data"`
}
