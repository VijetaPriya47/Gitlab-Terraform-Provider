package sdk

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerDataSource("gitlab_project", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_project`" + ` data source allows details of a project to be retrieved by either its ID or its path with namespace.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/projects/#get-single-project)`,

		ReadContext: dataSourceGitlabProjectRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Description:      "The integer that uniquely identifies the project within the gitlab install.",
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile(`^\d+$`), "`id` must be an integer string and not a path.")),
				ExactlyOneOf: []string{
					"id",
					"path_with_namespace",
				},
			},
			"name": {
				Description: "The name of the project.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"path": {
				Description: "The path of the repository.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"path_with_namespace": {
				Description: "The path of the repository with namespace.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ExactlyOneOf: []string{
					"id",
					"path_with_namespace",
				},
			},
			"description": {
				Description: "A description of the project.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"default_branch": {
				Description: "The default branch for the project.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"request_access_enabled": {
				Description: "Allow users to request member access.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"issues_enabled": {
				Description: "Enable issue tracking for the project.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"merge_requests_enabled": {
				Description: "Enable merge requests for the project.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"pipelines_enabled": {
				Description: "Enable pipelines for the project.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"wiki_enabled": {
				Description: "Enable wiki for the project.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"snippets_enabled": {
				Description: "Enable snippets for the project.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"lfs_enabled": {
				Description: "Enable LFS for the project.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"visibility_level": {
				Description: "Repositories are created as private by default.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"namespace_id": {
				Description: "The namespace (group or user) of the project. Defaults to your user.",
				Type:        schema.TypeInt,
				Computed:    true,
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
			"empty_repo": {
				Description: "Whether the project is empty.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"archived": {
				Description: "Whether the project is in read-only mode (archived).",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"remove_source_branch_after_merge": {
				Description: "Enable `Delete source branch` option by default for all new merge requests",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"restrict_user_defined_variables": {
				Description: "Allow only users with the Maintainer role to pass user-defined variables when triggering a pipeline.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"printing_merge_request_link_enabled": {
				Description: "Show link to create/view merge request when pushing from the command line",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"merge_pipelines_enabled": {
				Description: "Enable or disable merge pipelines.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"merge_trains_enabled": {
				Description: "Enable or disable merge trains.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"resolve_outdated_diff_discussions": {
				Description: "Automatically resolve merge request diffs discussions on lines changed with a push.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"analytics_access_level": {
				Description: fmt.Sprintf("Set the analytics access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"auto_cancel_pending_pipelines": {
				Description: "Auto-cancel pending pipelines. This isn’t a boolean, but enabled/disabled.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"auto_devops_deploy_strategy": {
				Description: fmt.Sprintf("Auto Deploy strategy. Valid values are %s.", utils.RenderValueListForDocs(validProjectAutoDevOpsDeployStrategyValues)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"auto_devops_enabled": {
				Description: "Enable Auto DevOps for this project.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"autoclose_referenced_issues": {
				Description: "Set whether auto-closing referenced issues on default branch.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"build_git_strategy": {
				Description: "The Git strategy. Defaults to fetch.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"build_timeout": {
				Description: "The maximum amount of time, in seconds, that a job can run.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"builds_access_level": {
				Description: fmt.Sprintf("Set the builds access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"container_expiration_policy": {
				Description: "Set the image cleanup policy for this project. **Note**: this field is sometimes named `container_expiration_policy_attributes` in the GitLab Upstream API.",
				Type:        schema.TypeList,
				Elem:        datasourceContainerExpirationPolicyAttributesSchema,
				Computed:    true,
			},
			"container_registry_access_level": {
				Description: fmt.Sprintf("Set visibility of container registry, for this project. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"emails_enabled": {
				Description: "Enable email notifications.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"external_authorization_classification_label": {
				Description: "The classification label for the project.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"forking_access_level": {
				Description: fmt.Sprintf("Set the forking access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"issues_access_level": {
				Description: fmt.Sprintf("Set the issues access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"merge_requests_access_level": {
				Description: fmt.Sprintf("Set the merge requests access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"public_builds": {
				Description: "If true, jobs can be viewed by non-project members.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"repository_access_level": {
				Description: fmt.Sprintf("Set the repository access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"repository_storage": {
				Description: "Which storage shard the repository is on. (administrator only)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"requirements_access_level": {
				Description: fmt.Sprintf("Set the requirements access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"security_and_compliance_access_level": {
				Description: fmt.Sprintf("Set the security and compliance access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"snippets_access_level": {
				Description: fmt.Sprintf("Set the snippets access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"suggestion_commit_message": {
				Description: "The commit message used to apply merge request suggestions.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"topics": {
				Description: "The list of topics for the project.",
				Type:        schema.TypeSet,
				Set:         schema.HashString,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
			"wiki_access_level": {
				Description: fmt.Sprintf("Set the wiki access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"squash_commit_template": {
				Description: "Template used to create squash commit message in merge requests.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"merge_commit_template": {
				Description: "Template used to create merge commit message in merge requests.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"allow_pipeline_trigger_approve_deployment": {
				Description: "Set whether or not a pipeline triggerer is allowed to approve deployments. Premium and Ultimate only.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"ci_default_git_depth": {
				Description: "Default number of revisions for shallow cloning.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"ci_delete_pipelines_in_seconds": {
				Description: "Pipelines older than the configured time are deleted.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"ci_config_path": {
				Description: "CI config file path for the project.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ci_separated_caches": {
				Description: "Use separate caches for protected branches.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"ci_id_token_sub_claim_components": {
				Description: `Fields included in the sub claim of the ID Token. Accepts an array starting with project_path. The array might also include ref_type and ref. Defaults to ["project_path", "ref_type", "ref"]. Introduced in GitLab 17.10.`,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Computed:    true,
			},
			"ci_restrict_pipeline_cancellation_role": {
				Description: fmt.Sprintf("The role required to cancel a pipeline or job. Premium and Ultimate only. Valid values are %s", utils.RenderValueListForDocs(api.ValidCIRestrictPipelineCancellationRoleValues)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ci_pipeline_variables_minimum_override_role": {
				Description: fmt.Sprintf("The minimum role required to set variables when running pipelines and jobs. Introduced in GitLab 17.1. Valid values are %s", utils.RenderValueListForDocs(api.ValidCIPipelineVariablesMinimumOverrideRoleValues)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"keep_latest_artifact": {
				Description: "Disable or enable the ability to keep the latest artifact for this project.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"import_url": {
				Description: "URL the project was imported from.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"push_rules": {
				Description: "Push rules for the project. Push rules are only available on Enterprise plans and if the authenticated has permissions to read them.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"author_email_regex": {
							Description: "All commit author emails must match this regex, e.g. `@my-company.com$`.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"branch_name_regex": {
							Description: "All branch names must match this regex, e.g. `(feature|hotfix)\\/*`.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"commit_message_regex": {
							Description: "All commit messages must match this regex, e.g. `Fixed \\d+\\..*`.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"commit_message_negative_regex": {
							Description: "No commit message is allowed to match this regex, for example `ssh\\:\\/\\/`.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"file_name_regex": {
							Description: "All committed filenames must not match this regex, e.g. `(jar|exe)$`.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"commit_committer_check": {
							Description: "Users can only push commits to this repository that were committed with one of their own verified emails.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"commit_committer_name_check": {
							Description: "Users can only push commits to this repository if the commit author name is consistent with their GitLab account name.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"deny_delete_tag": {
							Description: "Deny deleting a tag.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"member_check": {
							Description: "Restrict commits by author (email) to existing GitLab users.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"prevent_secrets": {
							Description: "GitLab will reject any files that are likely to contain secrets.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"reject_unsigned_commits": {
							Description: "Reject commit when it’s not signed through GPG.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"reject_non_dco_commits": {
							Description: "Reject commit when it’s not DCO certified.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"max_file_size": {
							Description: "Maximum file size (MB).",
							Type:        schema.TypeInt,
							Computed:    true,
						},
					},
				},
			},
			"releases_access_level": {
				Description: fmt.Sprintf("Set the releases access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"environments_access_level": {
				Description: fmt.Sprintf("Set the environments access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"feature_flags_access_level": {
				Description: fmt.Sprintf("Set the feature flags access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"infrastructure_access_level": {
				Description: fmt.Sprintf("Set the infrastructure access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"monitor_access_level": {
				Description: fmt.Sprintf("Set the monitor access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Type:        schema.TypeString,
				Computed:    true,
			},
			"model_experiments_access_level": {
				Description: "The visibility of machine learning model experiments.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"model_registry_access_level": {
				Description: "The visibility of machine learning model registry.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"prevent_merge_without_jira_issue": {
				Description: "Whether merge requests require an associated issue from Jira. Premium and Ultimate only.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"shared_with_groups": {
				Description: "Describes groups which have access shared to this project.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"group_id": {
							Description: "The ID of the group shared with.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"group_name": {
							Description: "The name of the group shared with.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"group_full_path": {
							Description: "The full path of the group shared with.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"group_access_level": {
							Description: "The access_level permission level of the shared group.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
					},
				},
			},
		},
	}
})

var datasourceContainerExpirationPolicyAttributesSchema = &schema.Resource{
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
			Optional:    true,
			Computed:    true,
		},
		"name_regex_delete": {
			Description: "The regular expression to match image names to delete.",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
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

func dataSourceGitlabProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	tflog.Debug(ctx, "[INFO] Reading Gitlab project")

	var pid interface{}
	if v, ok := d.GetOk("id"); ok {
		pid = v
	} else if v, ok := d.GetOk("path_with_namespace"); ok {
		pid = v
	} else {
		return diag.Errorf("Must specify either id or path_with_namespace")
	}

	found, _, err := client.Projects.GetProject(pid, nil, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", found.ID))
	d.Set("name", found.Name)
	d.Set("path", found.Path)
	d.Set("path_with_namespace", found.PathWithNamespace)
	d.Set("description", found.Description)
	d.Set("default_branch", found.DefaultBranch)
	d.Set("request_access_enabled", found.RequestAccessEnabled)
	d.Set("issues_enabled", found.IssuesEnabled)
	d.Set("merge_requests_enabled", found.MergeRequestsEnabled)
	d.Set("pipelines_enabled", found.JobsEnabled)
	d.Set("wiki_enabled", found.WikiEnabled)
	d.Set("snippets_enabled", found.SnippetsEnabled)
	d.Set("visibility_level", string(found.Visibility))
	d.Set("namespace_id", found.Namespace.ID)
	d.Set("ssh_url_to_repo", found.SSHURLToRepo)
	d.Set("http_url_to_repo", found.HTTPURLToRepo)
	d.Set("web_url", found.WebURL)
	d.Set("runners_token", found.RunnersToken)
	d.Set("empty_repo", found.EmptyRepo)
	d.Set("archived", found.Archived)
	d.Set("remove_source_branch_after_merge", found.RemoveSourceBranchAfterMerge)
	d.Set("restrict_user_defined_variables", found.RestrictUserDefinedVariables)
	d.Set("merge_pipelines_enabled", found.MergePipelinesEnabled)
	d.Set("merge_trains_enabled", found.MergeTrainsEnabled)
	d.Set("resolve_outdated_diff_discussions", found.ResolveOutdatedDiffDiscussions)
	d.Set("analytics_access_level", string(found.AnalyticsAccessLevel))
	d.Set("auto_cancel_pending_pipelines", found.AutoCancelPendingPipelines)
	d.Set("auto_devops_deploy_strategy", found.AutoDevopsDeployStrategy)
	d.Set("auto_devops_enabled", found.AutoDevopsEnabled)
	d.Set("autoclose_referenced_issues", found.AutocloseReferencedIssues)
	d.Set("build_git_strategy", found.BuildGitStrategy)
	d.Set("build_timeout", found.BuildTimeout)
	d.Set("builds_access_level", string(found.BuildsAccessLevel))
	if err := d.Set("container_expiration_policy", flattenContainerExpirationPolicy(found.ContainerExpirationPolicy)); err != nil {
		return diag.Errorf("error setting container_expiration_policy: %v", err)
	}
	d.Set("container_registry_access_level", string(found.ContainerRegistryAccessLevel))
	d.Set("external_authorization_classification_label", found.ExternalAuthorizationClassificationLabel)
	d.Set("forking_access_level", string(found.ForkingAccessLevel))
	d.Set("issues_access_level", string(found.IssuesAccessLevel))
	d.Set("merge_requests_access_level", string(found.MergeRequestsAccessLevel))
	d.Set("emails_enabled", found.EmailsEnabled)

	// Map PublicJobs -> PublicBuild until we have a breaking version.
	d.Set("public_builds", found.PublicJobs)
	d.Set("repository_access_level", string(found.RepositoryAccessLevel))
	d.Set("repository_storage", found.RepositoryStorage)
	d.Set("requirements_access_level", string(found.RequirementsAccessLevel))
	d.Set("security_and_compliance_access_level", string(found.SecurityAndComplianceAccessLevel))
	d.Set("snippets_access_level", string(found.SnippetsAccessLevel))
	d.Set("suggestion_commit_message", found.SuggestionCommitMessage)
	if err := d.Set("topics", found.Topics); err != nil {
		return diag.Errorf("error setting topics: %v", err)
	}
	d.Set("wiki_access_level", string(found.WikiAccessLevel))
	d.Set("squash_commit_template", found.SquashCommitTemplate)
	d.Set("merge_commit_template", found.MergeCommitTemplate)
	d.Set("allow_pipeline_trigger_approve_deployment", found.AllowPipelineTriggerApproveDeployment)
	d.Set("ci_default_git_depth", found.CIDefaultGitDepth)
	d.Set("ci_delete_pipelines_in_seconds", found.CIDeletePipelinesInSeconds)
	d.Set("ci_config_path", found.CIConfigPath)
	d.Set("ci_separated_caches", found.CISeperateCache)
	if err := d.Set("ci_id_token_sub_claim_components", found.CIIdTokenSubClaimComponents); err != nil {
		return diag.Errorf("error setting ci_id_token_sub_claim_components: %v", err)
	}
	d.Set("ci_restrict_pipeline_cancellation_role", found.CIRestrictPipelineCancellationRole)
	d.Set("ci_pipeline_variables_minimum_override_role", found.CIPipelineVariablesMinimumOverrideRole)
	d.Set("keep_latest_artifact", found.KeepLatestArtifact)
	d.Set("import_url", found.ImportURL)
	d.Set("releases_access_level", string(found.ReleasesAccessLevel))
	d.Set("environments_access_level", string(found.EnvironmentsAccessLevel))
	d.Set("feature_flags_access_level", string(found.FeatureFlagsAccessLevel))
	d.Set("infrastructure_access_level", string(found.InfrastructureAccessLevel))
	d.Set("monitor_access_level", string(found.MonitorAccessLevel))
	d.Set("model_experiments_access_level", string(found.ModelExperimentsAccessLevel))
	d.Set("model_registry_access_level", string(found.ModelRegistryAccessLevel))
	d.Set("prevent_merge_without_jira_issue", found.PreventMergeWithoutJiraIssue)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Reading Gitlab project %q push rules", d.Id()))

	pushRules, _, err := client.Projects.GetProjectPushRules(d.Id(), gitlab.WithContext(ctx))
	if api.Is404(err) || api.Is403(err) {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Failed to get push rules for project %q: %v", d.Id(), err))
	} else if err != nil {
		return diag.Errorf("Failed to get push rules for project %q: %v", d.Id(), err)
	}

	d.Set("push_rules", flattenProjectPushRules(pushRules)) // lintignore: XR004 // TODO: Resolve this tfproviderlint issue

	if err := d.Set("shared_with_groups", flattenProjectSharedWithGroups(found)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func flattenProjectSharedWithGroups(project *gitlab.Project) (values []map[string]interface{}) {
	for _, sharedGroup := range project.SharedWithGroups {
		v := map[string]interface{}{
			"group_id":           sharedGroup.GroupID,
			"group_name":         sharedGroup.GroupName,
			"group_full_path":    sharedGroup.GroupFullPath,
			"group_access_level": sharedGroup.GroupAccessLevel,
		}
		values = append(values, v)
	}

	return values
}
