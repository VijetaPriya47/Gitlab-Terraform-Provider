package provider

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabProjectDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectDataSource)
}

func NewGitlabProjectDataSource() datasource.DataSource {
	return &gitlabProjectDataSource{}
}

type gitlabProjectDataSource struct {
	client *gitlab.Client
}

type gitlabProjectDataSourceModel struct {
	ID                                       types.String                                            `tfsdk:"id"`
	Name                                     types.String                                            `tfsdk:"name"`
	Path                                     types.String                                            `tfsdk:"path"`
	PathWithNamespace                        types.String                                            `tfsdk:"path_with_namespace"`
	Description                              types.String                                            `tfsdk:"description"`
	DefaultBranch                            types.String                                            `tfsdk:"default_branch"`
	RequestAccessEnabled                     types.Bool                                              `tfsdk:"request_access_enabled"`
	IssuesEnabled                            types.Bool                                              `tfsdk:"issues_enabled"`
	MergeRequestsEnabled                     types.Bool                                              `tfsdk:"merge_requests_enabled"`
	PipelinesEnabled                         types.Bool                                              `tfsdk:"pipelines_enabled"`
	WikiEnabled                              types.Bool                                              `tfsdk:"wiki_enabled"`
	SnippetsEnabled                          types.Bool                                              `tfsdk:"snippets_enabled"`
	LFSEnabled                               types.Bool                                              `tfsdk:"lfs_enabled"`
	VisibilityLevel                          types.String                                            `tfsdk:"visibility_level"`
	NamespaceID                              types.Int64                                             `tfsdk:"namespace_id"`
	SSHURLToRepo                             types.String                                            `tfsdk:"ssh_url_to_repo"`
	HTTPURLToRepo                            types.String                                            `tfsdk:"http_url_to_repo"`
	WebURL                                   types.String                                            `tfsdk:"web_url"`
	RunnersToken                             types.String                                            `tfsdk:"runners_token"`
	EmptyRepo                                types.Bool                                              `tfsdk:"empty_repo"`
	Archived                                 types.Bool                                              `tfsdk:"archived"`
	RemoveSourceBranchAfterMerge             types.Bool                                              `tfsdk:"remove_source_branch_after_merge"`
	RestrictUserDefinedVariables             types.Bool                                              `tfsdk:"restrict_user_defined_variables"`
	PrintingMergeRequestLinkEnabled          types.Bool                                              `tfsdk:"printing_merge_request_link_enabled"`
	MergePipelinesEnabled                    types.Bool                                              `tfsdk:"merge_pipelines_enabled"`
	MergeTrainsEnabled                       types.Bool                                              `tfsdk:"merge_trains_enabled"`
	MergeTrainsSkipTrainAllowed              types.Bool                                              `tfsdk:"merge_trains_skip_train_allowed"`
	ResolveOutdatedDiffDiscussions           types.Bool                                              `tfsdk:"resolve_outdated_diff_discussions"`
	AnalyticsAccessLevel                     types.String                                            `tfsdk:"analytics_access_level"`
	AutoCancelPendingPipelines               types.String                                            `tfsdk:"auto_cancel_pending_pipelines"`
	AutoDevopsDeployStrategy                 types.String                                            `tfsdk:"auto_devops_deploy_strategy"`
	AutoDevopsEnabled                        types.Bool                                              `tfsdk:"auto_devops_enabled"`
	AutocloseReferencedIssues                types.Bool                                              `tfsdk:"autoclose_referenced_issues"`
	BuildGitStrategy                         types.String                                            `tfsdk:"build_git_strategy"`
	BuildTimeout                             types.Int64                                             `tfsdk:"build_timeout"`
	BuildsAccessLevel                        types.String                                            `tfsdk:"builds_access_level"`
	ContainerExpirationPolicy                []gitlabProjectContainerExpirationPolicyDataSourceModel `tfsdk:"container_expiration_policy"`
	ContainerRegistryAccessLevel             types.String                                            `tfsdk:"container_registry_access_level"`
	EmailsEnabled                            types.Bool                                              `tfsdk:"emails_enabled"`
	ExternalAuthorizationClassificationLabel types.String                                            `tfsdk:"external_authorization_classification_label"`
	ForkingAccessLevel                       types.String                                            `tfsdk:"forking_access_level"`
	IssuesAccessLevel                        types.String                                            `tfsdk:"issues_access_level"`
	MergeRequestsAccessLevel                 types.String                                            `tfsdk:"merge_requests_access_level"`
	PublicBuilds                             types.Bool                                              `tfsdk:"public_builds"`
	RepositoryAccessLevel                    types.String                                            `tfsdk:"repository_access_level"`
	RepositoryStorage                        types.String                                            `tfsdk:"repository_storage"`
	RequirementsAccessLevel                  types.String                                            `tfsdk:"requirements_access_level"`
	SecurityAndComplianceAccessLevel         types.String                                            `tfsdk:"security_and_compliance_access_level"`
	SnippetsAccessLevel                      types.String                                            `tfsdk:"snippets_access_level"`
	SuggestionCommitMessage                  types.String                                            `tfsdk:"suggestion_commit_message"`
	Topics                                   types.Set                                               `tfsdk:"topics"`
	WikiAccessLevel                          types.String                                            `tfsdk:"wiki_access_level"`
	SquashCommitTemplate                     types.String                                            `tfsdk:"squash_commit_template"`
	MergeCommitTemplate                      types.String                                            `tfsdk:"merge_commit_template"`
	AllowPipelineTriggerApproveDeployment    types.Bool                                              `tfsdk:"allow_pipeline_trigger_approve_deployment"`
	CIDefaultGitDepth                        types.Int64                                             `tfsdk:"ci_default_git_depth"`
	CIDeletePipelinesInSeconds               types.Int64                                             `tfsdk:"ci_delete_pipelines_in_seconds"`
	CIConfigPath                             types.String                                            `tfsdk:"ci_config_path"`
	CISeparatedCaches                        types.Bool                                              `tfsdk:"ci_separated_caches"`
	CIIDTokenSubClaimComponents              types.List                                              `tfsdk:"ci_id_token_sub_claim_components"`
	CIRestrictPipelineCancellationRole       types.String                                            `tfsdk:"ci_restrict_pipeline_cancellation_role"`
	CIPipelineVariablesMinimumOverrideRole   types.String                                            `tfsdk:"ci_pipeline_variables_minimum_override_role"`
	KeepLatestArtifact                       types.Bool                                              `tfsdk:"keep_latest_artifact"`
	ImportURL                                types.String                                            `tfsdk:"import_url"`
	PushRules                                []gitlabProjectPushRuleDataSourceModel                  `tfsdk:"push_rules"`
	ReleasesAccessLevel                      types.String                                            `tfsdk:"releases_access_level"`
	EnvironmentsAccessLevel                  types.String                                            `tfsdk:"environments_access_level"`
	FeatureFlagsAccessLevel                  types.String                                            `tfsdk:"feature_flags_access_level"`
	InfrastructureAccessLevel                types.String                                            `tfsdk:"infrastructure_access_level"`
	MonitorAccessLevel                       types.String                                            `tfsdk:"monitor_access_level"`
	ModelExperimentsAccessLevel              types.String                                            `tfsdk:"model_experiments_access_level"`
	ModelRegistryAccessLevel                 types.String                                            `tfsdk:"model_registry_access_level"`
	PreventMergeWithoutJiraIssue             types.Bool                                              `tfsdk:"prevent_merge_without_jira_issue"`
	SharedWithGroups                         []gitlabProjectSharedWithGroupDataSourceModel           `tfsdk:"shared_with_groups"`
}

type gitlabProjectContainerExpirationPolicyDataSourceModel struct {
	Cadence         types.String `tfsdk:"cadence"`
	KeepN           types.Int64  `tfsdk:"keep_n"`
	OlderThan       types.String `tfsdk:"older_than"`
	NameRegexDelete types.String `tfsdk:"name_regex_delete"`
	NameRegexKeep   types.String `tfsdk:"name_regex_keep"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	NextRunAt       types.String `tfsdk:"next_run_at"`
}

type gitlabProjectPushRuleDataSourceModel struct {
	AuthorEmailRegex           types.String `tfsdk:"author_email_regex"`
	BranchNameRegex            types.String `tfsdk:"branch_name_regex"`
	CommitMessageRegex         types.String `tfsdk:"commit_message_regex"`
	CommitMessageNegativeRegex types.String `tfsdk:"commit_message_negative_regex"`
	FileNameRegex              types.String `tfsdk:"file_name_regex"`
	CommitCommitterCheck       types.Bool   `tfsdk:"commit_committer_check"`
	CommitCommitterNameCheck   types.Bool   `tfsdk:"commit_committer_name_check"`
	DenyDeleteTag              types.Bool   `tfsdk:"deny_delete_tag"`
	MemberCheck                types.Bool   `tfsdk:"member_check"`
	PreventSecrets             types.Bool   `tfsdk:"prevent_secrets"`
	RejectUnsignedCommits      types.Bool   `tfsdk:"reject_unsigned_commits"`
	RejectNonDCOCommits        types.Bool   `tfsdk:"reject_non_dco_commits"`
	MaxFileSize                types.Int64  `tfsdk:"max_file_size"`
}

type gitlabProjectSharedWithGroupDataSourceModel struct {
	GroupID          types.Int64  `tfsdk:"group_id"`
	GroupName        types.String `tfsdk:"group_name"`
	GroupFullPath    types.String `tfsdk:"group_full_path"`
	GroupAccessLevel types.Int64  `tfsdk:"group_access_level"`
}

func (d *gitlabProjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

var (
	validProjectAccessLevels = []string{
		"disabled",
		"private",
		"enabled",
	}
	validProjectAutoDevOpsDeployStrategyValues = []string{
		"continuous",
		"manual",
		"timed_incremental",
	}
	validContainerExpirationPolicyAttributesCadenceValues = []string{
		"1d", "7d", "14d", "1month", "3month",
	}
)

func (d *gitlabProjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project`" + ` data source allows details of a project to be retrieved by either its ID or its path with namespace.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/projects/#get-a-single-project)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The integer that uniquely identifies the project within the gitlab install.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("path_with_namespace")),
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "`id` must be an integer string and not a path."),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Computed:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The path of the repository.",
				Computed:            true,
			},
			"path_with_namespace": schema.StringAttribute{
				MarkdownDescription: "The path of the repository with namespace.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("id"))},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the project.",
				Computed:            true,
			},
			"default_branch": schema.StringAttribute{
				MarkdownDescription: "The default branch for the project.",
				Computed:            true,
			},
			"request_access_enabled": schema.BoolAttribute{
				MarkdownDescription: "Allow users to request member access.",
				Computed:            true,
			},
			"issues_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable issue tracking for the project. Use `issues_access_level` instead. This attribute will be removed in 19.0.",
				Computed:            true,
				DeprecationMessage:  "Use `issues_access_level` instead. This attribute will be removed in 19.0.",
			},
			"merge_requests_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable merge requests for the project. Use `merge_requests_access_level` instead. This attribute will be removed in 19.0.",
				Computed:            true,
				DeprecationMessage:  "Use `merge_requests_access_level` instead. This attribute will be removed in 19.0.",
			},
			"pipelines_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable pipelines for the project. Use `pipelines_access_level` instead. This attribute will be removed in 19.0.",
				Computed:            true,
				DeprecationMessage:  "Use `pipelines_access_level` instead. This attribute will be removed in 19.0.",
			},
			"wiki_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable wiki for the project. Use `wiki_access_level` instead. This attribute will be removed in 19.0.",
				Computed:            true,
				DeprecationMessage:  "Use `wiki_access_level` instead. This attribute will be removed in 19.0.",
			},
			"snippets_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable snippets for the project. Use `snippets_access_level` instead. This attribute will be removed in 19.0.",
				Computed:            true,
				DeprecationMessage:  "Use `snippets_access_level` instead. This attribute will be removed in 19.0.",
			},
			"lfs_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable LFS for the project.",
				Computed:            true,
			},
			"visibility_level": schema.StringAttribute{
				MarkdownDescription: "Repositories are created as private by default.",
				Computed:            true,
			},
			"namespace_id": schema.Int64Attribute{
				MarkdownDescription: "The namespace (group or user) of the project. Defaults to your user.",
				Computed:            true,
			},
			"ssh_url_to_repo": schema.StringAttribute{
				MarkdownDescription: "URL that can be provided to `git clone` to clone the",
				Computed:            true,
			},
			"http_url_to_repo": schema.StringAttribute{
				MarkdownDescription: "URL that can be provided to `git clone` to clone the",
				Computed:            true,
			},
			"web_url": schema.StringAttribute{
				MarkdownDescription: "URL that can be used to find the project in a browser.",
				Computed:            true,
			},
			"runners_token": schema.StringAttribute{
				MarkdownDescription: "Registration token to use during runner setup.",
				Computed:            true,
				Sensitive:           true,
			},
			"empty_repo": schema.BoolAttribute{
				MarkdownDescription: "Whether the project is empty.",
				Computed:            true,
			},
			"archived": schema.BoolAttribute{
				MarkdownDescription: "Whether the project is in read-only mode (archived).",
				Computed:            true,
			},
			"remove_source_branch_after_merge": schema.BoolAttribute{
				MarkdownDescription: "Enable `Delete source branch` option by default for all new merge requests",
				Computed:            true,
			},
			"restrict_user_defined_variables": schema.BoolAttribute{
				MarkdownDescription: "Allow only users with the Maintainer role to pass user-defined variables when triggering a pipeline. Use `ci_restrict_pipeline_variables_role` instead. This attribute will be removed in 19.0.",
				Computed:            true,
				DeprecationMessage:  "Use `ci_restrict_pipeline_variables_role` instead. This attribute will be removed in 19.0.",
			},
			"printing_merge_request_link_enabled": schema.BoolAttribute{
				MarkdownDescription: "Show link to create/view merge request when pushing from the command line",
				Computed:            true,
			},
			"merge_pipelines_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable or disable merge pipelines.",
				Computed:            true,
			},
			"merge_trains_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable or disable merge trains.",
				Computed:            true,
			},
			"merge_trains_skip_train_allowed": schema.BoolAttribute{
				MarkdownDescription: "Allows merge train merge requests to be merged without waiting for pipelines to finish.",
				Computed:            true,
			},
			"resolve_outdated_diff_discussions": schema.BoolAttribute{
				MarkdownDescription: "Automatically resolve merge request diffs discussions on lines changed with a push.",
				Computed:            true,
			},
			"analytics_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the analytics access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"auto_cancel_pending_pipelines": schema.StringAttribute{
				MarkdownDescription: "Auto-cancel pending pipelines. This isn’t a boolean, but enabled/disabled.",
				Computed:            true,
			},
			"auto_devops_deploy_strategy": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Auto Deploy strategy. Valid values are %s.", utils.RenderValueListForDocs(validProjectAutoDevOpsDeployStrategyValues)),
				Computed:            true,
			},
			"auto_devops_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable Auto DevOps for this project.",
				Computed:            true,
			},
			"autoclose_referenced_issues": schema.BoolAttribute{
				MarkdownDescription: "Set whether auto-closing referenced issues on default branch.",
				Computed:            true,
			},
			"build_git_strategy": schema.StringAttribute{
				MarkdownDescription: "The Git strategy. Defaults to fetch.",
				Computed:            true,
			},
			"build_timeout": schema.Int64Attribute{
				MarkdownDescription: "The maximum amount of time, in seconds, that a job can run.",
				Computed:            true,
			},
			"builds_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the builds access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"container_expiration_policy": schema.ListNestedAttribute{
				MarkdownDescription: "Set the image cleanup policy for this project. **Note**: this field is sometimes named `container_expiration_policy_attributes` in the GitLab Upstream API.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cadence": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("The cadence of the policy. Valid values are: %s.", utils.RenderValueListForDocs(validContainerExpirationPolicyAttributesCadenceValues)),
							Computed:            true,
							Validators:          []validator.String{stringvalidator.OneOf(validContainerExpirationPolicyAttributesCadenceValues...)},
						},
						"keep_n": schema.Int64Attribute{
							MarkdownDescription: "The number of images to keep.",
							Computed:            true,
							Validators:          []validator.Int64{int64validator.AtLeast(0)},
						},
						"older_than": schema.StringAttribute{
							MarkdownDescription: "The number of days to keep images.",
							Computed:            true,
						},
						"name_regex_delete": schema.StringAttribute{
							MarkdownDescription: "The regular expression to match image names to delete.",
							Computed:            true,
						},
						"name_regex_keep": schema.StringAttribute{
							MarkdownDescription: "The regular expression to match image names to keep.",
							Computed:            true,
						},
						"enabled": schema.BoolAttribute{
							MarkdownDescription: "If true, the policy is enabled.",
							Computed:            true,
						},
						"next_run_at": schema.StringAttribute{
							MarkdownDescription: "The next time the policy will run.",
							Computed:            true,
						},
					},
				},
			},
			"container_registry_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set visibility of container registry, for this project. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"emails_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable email notifications.",
				Computed:            true,
			},
			"external_authorization_classification_label": schema.StringAttribute{
				MarkdownDescription: "The classification label for the project.",
				Computed:            true,
			},
			"forking_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the forking access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"issues_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the issues access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"merge_requests_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the merge requests access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"public_builds": schema.BoolAttribute{
				MarkdownDescription: "If true, jobs can be viewed by non-project members.",
				Optional:            true,
			},
			"repository_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the repository access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"repository_storage": schema.StringAttribute{
				MarkdownDescription: "Which storage shard the repository is on. (administrator only)",
				Computed:            true,
			},
			"requirements_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the requirements access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"security_and_compliance_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the security and compliance access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"snippets_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the snippets access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"suggestion_commit_message": schema.StringAttribute{
				MarkdownDescription: "The commit message used to apply merge request suggestions.",
				Computed:            true,
			},
			"topics": schema.SetAttribute{
				MarkdownDescription: "The list of topics for the project.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"wiki_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the wiki access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"squash_commit_template": schema.StringAttribute{
				MarkdownDescription: "Template used to create squash commit message in merge requests.",
				Computed:            true,
			},
			"merge_commit_template": schema.StringAttribute{
				MarkdownDescription: "Template used to create merge commit message in merge requests.",
				Computed:            true,
			},
			"allow_pipeline_trigger_approve_deployment": schema.BoolAttribute{
				MarkdownDescription: "Set whether or not a pipeline triggerer is allowed to approve deployments. Premium and Ultimate only.",
				Computed:            true,
			},
			"ci_default_git_depth": schema.Int64Attribute{
				MarkdownDescription: "Default number of revisions for shallow cloning.",
				Optional:            true,
				Computed:            true,
			},
			"ci_delete_pipelines_in_seconds": schema.Int64Attribute{
				MarkdownDescription: "Pipelines older than the configured time are deleted.",
				Computed:            true,
			},
			"ci_config_path": schema.StringAttribute{
				MarkdownDescription: "CI config file path for the project.",
				Computed:            true,
			},
			"ci_separated_caches": schema.BoolAttribute{
				MarkdownDescription: "Use separate caches for protected branches.",
				Computed:            true,
			},
			"ci_id_token_sub_claim_components": schema.ListAttribute{
				MarkdownDescription: `Fields included in the sub claim of the ID Token. Accepts an array starting with project_path. The array might also include ref_type and ref. Defaults to ["project_path", "ref_type", "ref"]. Introduced in GitLab 17.10.`,
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"ci_restrict_pipeline_cancellation_role": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The role required to cancel a pipeline or job. Premium and Ultimate only. Valid values are %s", utils.RenderValueListForDocs(api.ValidCIRestrictPipelineCancellationRoleValues)),
				Computed:            true,
			},
			"ci_pipeline_variables_minimum_override_role": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The minimum role required to set variables when running pipelines and jobs. Introduced in GitLab 17.1. Valid values are %s", utils.RenderValueListForDocs(api.ValidCIPipelineVariablesMinimumOverrideRoleValues)),
				Computed:            true,
			},
			"keep_latest_artifact": schema.BoolAttribute{
				MarkdownDescription: "Disable or enable the ability to keep the latest artifact for this project.",
				Computed:            true,
			},
			"import_url": schema.StringAttribute{
				MarkdownDescription: "URL the project was imported from.",
				Computed:            true,
			},
			"push_rules": schema.ListNestedAttribute{
				MarkdownDescription: "Push rules for the project. Push rules are only available on Enterprise plans and if the authenticated has permissions to read them.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"author_email_regex": schema.StringAttribute{
							MarkdownDescription: "All commit author emails must match this regex, e.g. `@my-company.com$`.",
							Computed:            true,
						},
						"branch_name_regex": schema.StringAttribute{
							MarkdownDescription: "All branch names must match this regex, e.g. `(feature|hotfix)\\/*`.",
							Computed:            true,
						},
						"commit_message_regex": schema.StringAttribute{
							MarkdownDescription: "All commit messages must match this regex, e.g. `Fixed \\d+\\..*`.",
							Computed:            true,
						},
						"commit_message_negative_regex": schema.StringAttribute{
							MarkdownDescription: "No commit message is allowed to match this regex, for example `ssh\\:\\/\\/`.",
							Computed:            true,
						},
						"file_name_regex": schema.StringAttribute{
							MarkdownDescription: "All committed filenames must not match this regex, e.g. `(jar|exe)$`.",
							Computed:            true,
						},
						"commit_committer_check": schema.BoolAttribute{
							MarkdownDescription: "Users can only push commits to this repository that were committed with one of their own verified emails.",
							Computed:            true,
						},
						"commit_committer_name_check": schema.BoolAttribute{
							MarkdownDescription: "Users can only push commits to this repository if the commit author name is consistent with their GitLab account name.",
							Computed:            true,
						},
						"deny_delete_tag": schema.BoolAttribute{
							MarkdownDescription: "Deny deleting a tag.",
							Computed:            true,
						},
						"member_check": schema.BoolAttribute{
							MarkdownDescription: "Restrict commits by author (email) to existing GitLab users.",
							Computed:            true,
						},
						"prevent_secrets": schema.BoolAttribute{
							MarkdownDescription: "GitLab will reject any files that are likely to contain secrets.",
							Computed:            true,
						},
						"reject_unsigned_commits": schema.BoolAttribute{
							MarkdownDescription: "Reject commit when it's not signed through GPG.",
							Computed:            true,
						},
						"reject_non_dco_commits": schema.BoolAttribute{
							MarkdownDescription: "Reject commit when it's not DCO certified.",
							Computed:            true,
						},
						"max_file_size": schema.Int64Attribute{
							MarkdownDescription: "Maximum file size (MB).",
							Computed:            true,
						},
					},
				},
			},
			"releases_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the releases access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"environments_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the environments access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"feature_flags_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the feature flags access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"infrastructure_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the infrastructure access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"monitor_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Set the monitor access level. Valid values are %s.", utils.RenderValueListForDocs(validProjectAccessLevels)),
				Computed:            true,
			},
			"model_experiments_access_level": schema.StringAttribute{
				MarkdownDescription: "The visibility of machine learning model experiments.",
				Computed:            true,
			},
			"model_registry_access_level": schema.StringAttribute{
				MarkdownDescription: "The visibility of machine learning model registry.",
				Computed:            true,
			},
			"prevent_merge_without_jira_issue": schema.BoolAttribute{
				MarkdownDescription: "Whether merge requests require an associated issue from Jira. Premium and Ultimate only.",
				Computed:            true,
			},
			"shared_with_groups": schema.ListNestedAttribute{
				MarkdownDescription: "Describes groups which have access shared to this project.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the group shared with.",
							Computed:            true,
						},
						"group_name": schema.StringAttribute{
							MarkdownDescription: "The name of the group shared with.",
							Computed:            true,
						},
						"group_full_path": schema.StringAttribute{
							MarkdownDescription: "The full path of the group shared with.",
							Computed:            true,
						},
						"group_access_level": schema.Int64Attribute{
							MarkdownDescription: "The access_level permission level of the shared group.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabProjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "[INFO] Reading Gitlab project")

	var pid any
	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		pid = data.ID.ValueString()
	} else if !data.PathWithNamespace.IsNull() && !data.PathWithNamespace.IsUnknown() {
		pid = data.PathWithNamespace.ValueString()
	} else {
		resp.Diagnostics.AddError("Invalid attribute combination", "Must specify either id or path_with_namespace")
	}

	found, _, err := d.client.Projects.GetProject(pid, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", found.ID))
	data.Name = types.StringValue(found.Name)
	data.Path = types.StringValue(found.Path)
	data.PathWithNamespace = types.StringValue(found.PathWithNamespace)
	data.Description = types.StringValue(found.Description)
	data.DefaultBranch = types.StringValue(found.DefaultBranch)
	data.RequestAccessEnabled = types.BoolValue(found.RequestAccessEnabled)
	data.IssuesEnabled = types.BoolValue(found.IssuesEnabled)               //nolint:staticcheck
	data.MergeRequestsEnabled = types.BoolValue(found.MergeRequestsEnabled) //nolint:staticcheck
	data.PipelinesEnabled = types.BoolValue(found.JobsEnabled)              //nolint:staticcheck
	data.WikiEnabled = types.BoolValue(found.WikiEnabled)                   //nolint:staticcheck
	data.SnippetsEnabled = types.BoolValue(found.SnippetsEnabled)           //nolint:staticcheck
	data.VisibilityLevel = types.StringValue(string(found.Visibility))
	data.NamespaceID = types.Int64Value(int64(found.Namespace.ID))
	data.SSHURLToRepo = types.StringValue(found.SSHURLToRepo)
	data.HTTPURLToRepo = types.StringValue(found.HTTPURLToRepo)
	data.WebURL = types.StringValue(found.WebURL)
	data.RunnersToken = types.StringValue(found.RunnersToken)
	data.EmptyRepo = types.BoolValue(found.EmptyRepo)
	data.Archived = types.BoolValue(found.Archived)
	data.RemoveSourceBranchAfterMerge = types.BoolValue(found.RemoveSourceBranchAfterMerge)
	data.RestrictUserDefinedVariables = types.BoolValue(found.RestrictUserDefinedVariables) //nolint:staticcheck
	data.MergePipelinesEnabled = types.BoolValue(found.MergePipelinesEnabled)
	data.MergeTrainsEnabled = types.BoolValue(found.MergeTrainsEnabled)
	data.MergeTrainsSkipTrainAllowed = types.BoolValue(found.MergeTrainsSkipTrainAllowed)
	data.ResolveOutdatedDiffDiscussions = types.BoolValue(found.ResolveOutdatedDiffDiscussions)
	data.AnalyticsAccessLevel = types.StringValue(string(found.AnalyticsAccessLevel))
	data.AutoCancelPendingPipelines = types.StringValue(found.AutoCancelPendingPipelines)
	data.AutoDevopsDeployStrategy = types.StringValue(found.AutoDevopsDeployStrategy)
	data.AutoDevopsEnabled = types.BoolValue(found.AutoDevopsEnabled)
	data.AutocloseReferencedIssues = types.BoolValue(found.AutocloseReferencedIssues)
	data.BuildGitStrategy = types.StringValue(found.BuildGitStrategy)
	data.BuildTimeout = types.Int64Value(int64(found.BuildTimeout))
	data.BuildsAccessLevel = types.StringValue(string(found.BuildsAccessLevel))
	data.ContainerExpirationPolicy = []gitlabProjectContainerExpirationPolicyDataSourceModel{}

	if found.ContainerExpirationPolicy != nil {
		policy := gitlabProjectContainerExpirationPolicyDataSourceModel{
			Cadence:         types.StringValue(found.ContainerExpirationPolicy.Cadence),
			KeepN:           types.Int64Value(int64(found.ContainerExpirationPolicy.KeepN)),
			OlderThan:       types.StringValue(found.ContainerExpirationPolicy.OlderThan),
			NameRegexDelete: types.StringValue(found.ContainerExpirationPolicy.NameRegexDelete),
			NameRegexKeep:   types.StringValue(found.ContainerExpirationPolicy.NameRegexKeep),
			Enabled:         types.BoolValue(found.ContainerExpirationPolicy.Enabled),
		}
		if found.ContainerExpirationPolicy.NextRunAt == nil {
			policy.NextRunAt = types.StringNull()
		} else {
			policy.NextRunAt = types.StringValue(found.ContainerExpirationPolicy.NextRunAt.Format(time.RFC3339))
		}
		data.ContainerExpirationPolicy = append(data.ContainerExpirationPolicy, policy)
	}
	data.ContainerRegistryAccessLevel = types.StringValue(string(found.ContainerRegistryAccessLevel))
	data.ExternalAuthorizationClassificationLabel = types.StringValue(found.ExternalAuthorizationClassificationLabel)
	data.ForkingAccessLevel = types.StringValue(string(found.ForkingAccessLevel))
	data.IssuesAccessLevel = types.StringValue(string(found.IssuesAccessLevel))
	data.MergeRequestsAccessLevel = types.StringValue(string(found.MergeRequestsAccessLevel))
	data.EmailsEnabled = types.BoolValue(found.EmailsEnabled)

	// Map PublicJobs -> PublicBuild until we have a breaking version. Fix in 19.0.
	data.PublicBuilds = types.BoolValue(found.PublicJobs)
	data.RepositoryAccessLevel = types.StringValue(string(found.RepositoryAccessLevel))
	data.RepositoryStorage = types.StringValue(found.RepositoryStorage)
	data.RequirementsAccessLevel = types.StringValue(string(found.RequirementsAccessLevel))
	data.SecurityAndComplianceAccessLevel = types.StringValue(string(found.SecurityAndComplianceAccessLevel))
	data.SnippetsAccessLevel = types.StringValue(string(found.SnippetsAccessLevel))
	data.SuggestionCommitMessage = types.StringValue(found.SuggestionCommitMessage)
	topics, diags := types.SetValueFrom(ctx, types.StringType, found.Topics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Topics = topics
	data.WikiAccessLevel = types.StringValue(string(found.WikiAccessLevel))
	data.SquashCommitTemplate = types.StringValue(found.SquashCommitTemplate)
	data.MergeCommitTemplate = types.StringValue(found.MergeCommitTemplate)
	data.AllowPipelineTriggerApproveDeployment = types.BoolValue(found.AllowPipelineTriggerApproveDeployment)
	data.CIDefaultGitDepth = types.Int64Value(int64(found.CIDefaultGitDepth))
	data.CIDeletePipelinesInSeconds = types.Int64Value(int64(found.CIDeletePipelinesInSeconds))
	data.CIConfigPath = types.StringValue(found.CIConfigPath)
	data.CISeparatedCaches = types.BoolValue(found.CISeparatedCaches)
	claims, diags := types.ListValueFrom(ctx, types.StringType, found.CIIdTokenSubClaimComponents)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.CIIDTokenSubClaimComponents = claims
	data.CIRestrictPipelineCancellationRole = types.StringValue(string(found.CIRestrictPipelineCancellationRole))
	data.CIPipelineVariablesMinimumOverrideRole = types.StringValue(found.CIPipelineVariablesMinimumOverrideRole)
	data.KeepLatestArtifact = types.BoolValue(found.KeepLatestArtifact)
	data.ImportURL = types.StringValue(found.ImportURL)
	data.ReleasesAccessLevel = types.StringValue(string(found.ReleasesAccessLevel))
	data.EnvironmentsAccessLevel = types.StringValue(string(found.EnvironmentsAccessLevel))
	data.FeatureFlagsAccessLevel = types.StringValue(string(found.FeatureFlagsAccessLevel))
	data.InfrastructureAccessLevel = types.StringValue(string(found.InfrastructureAccessLevel))
	data.MonitorAccessLevel = types.StringValue(string(found.MonitorAccessLevel))
	data.ModelExperimentsAccessLevel = types.StringValue(string(found.ModelExperimentsAccessLevel))
	data.ModelRegistryAccessLevel = types.StringValue(string(found.ModelRegistryAccessLevel))
	data.PreventMergeWithoutJiraIssue = types.BoolValue(found.PreventMergeWithoutJiraIssue)

	tflog.Debug(ctx, fmt.Sprintf("Reading Gitlab project %d push rules", found.ID))

	pushRules, _, err := d.client.Projects.GetProjectPushRules(found.ID, gitlab.WithContext(ctx))
	if api.Is404(err) || api.Is403(err) {
		tflog.Debug(ctx, fmt.Sprintf("Failed to get push rules for project %d: %v", found.ID, err))
	} else if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project push rules: %s", err.Error()))
		return
	}
	data.PushRules = []gitlabProjectPushRuleDataSourceModel{}
	if pushRules != nil {
		pushRule := gitlabProjectPushRuleDataSourceModel{
			AuthorEmailRegex:           types.StringValue(pushRules.AuthorEmailRegex),
			BranchNameRegex:            types.StringValue(pushRules.BranchNameRegex),
			CommitMessageRegex:         types.StringValue(pushRules.CommitMessageRegex),
			CommitMessageNegativeRegex: types.StringValue(pushRules.CommitMessageNegativeRegex),
			FileNameRegex:              types.StringValue(pushRules.FileNameRegex),
			CommitCommitterCheck:       types.BoolValue(pushRules.CommitCommitterCheck),
			CommitCommitterNameCheck:   types.BoolValue(pushRules.CommitCommitterNameCheck),
			DenyDeleteTag:              types.BoolValue(pushRules.DenyDeleteTag),
			MemberCheck:                types.BoolValue(pushRules.MemberCheck),
			PreventSecrets:             types.BoolValue(pushRules.PreventSecrets),
			RejectUnsignedCommits:      types.BoolValue(pushRules.RejectUnsignedCommits),
			RejectNonDCOCommits:        types.BoolValue(pushRules.RejectNonDCOCommits),
			MaxFileSize:                types.Int64Value(int64(pushRules.MaxFileSize)),
		}
		data.PushRules = append(data.PushRules, pushRule)
	}
	data.SharedWithGroups = []gitlabProjectSharedWithGroupDataSourceModel{}
	for _, sharedGroup := range found.SharedWithGroups {
		share := gitlabProjectSharedWithGroupDataSourceModel{
			GroupID:          types.Int64Value(int64(sharedGroup.GroupID)),
			GroupName:        types.StringValue(sharedGroup.GroupName),
			GroupFullPath:    types.StringValue(sharedGroup.GroupFullPath),
			GroupAccessLevel: types.Int64Value(int64(sharedGroup.GroupAccessLevel)),
		}
		data.SharedWithGroups = append(data.SharedWithGroups, share)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
