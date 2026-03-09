package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIntegrationJiraResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationJiraResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationJiraResource{}
	_ resource.ResourceWithMoveState   = &gitlabProjectIntegrationJiraResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationJiraResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationJiraResource)
}

func NewGitLabProjectIntegrationJiraResource() resource.Resource {
	return &gitlabProjectIntegrationJiraResource{
		ResourceName: "_project_integration_jira",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_jira` + "`" + ` resource manages the lifecycle of a project integration with Jira.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#jira-issues)`,
	}
}

// Remove in 19.0
func NewGitLabIntegrationJiraResource() resource.Resource {
	return &gitlabProjectIntegrationJiraResource{
		ResourceName: "_integration_jira",
		ResourceDescription: `The ` + "`" + `gitlab_integration_jira` + "`" + ` resource manages the lifecycle of a project integration with Jira.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_jira` + "`" + ` instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#jira-issues)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_jira` instead.",
	}
}

type gitlabProjectIntegrationJiraResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_jira`
	// and `gitlab_integration_jira` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationJiraResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	Project                      types.String `tfsdk:"project"`
	URL                          types.String `tfsdk:"url"`
	APIURL                       types.String `tfsdk:"api_url"`
	Username                     types.String `tfsdk:"username"`
	Password                     types.String `tfsdk:"password"`
	JiraAuthType                 types.Int64  `tfsdk:"jira_auth_type"`
	JiraIssuePrefix              types.String `tfsdk:"jira_issue_prefix"`
	JiraIssueRegex               types.String `tfsdk:"jira_issue_regex"`
	JiraIssueTransitionAutomatic types.Bool   `tfsdk:"jira_issue_transition_automatic"`
	JiraIssueTransitionID        types.String `tfsdk:"jira_issue_transition_id"`
	CommitEvents                 types.Bool   `tfsdk:"commit_events"`
	MergeRequestsEvents          types.Bool   `tfsdk:"merge_requests_events"`
	CommentOnEventEnabled        types.Bool   `tfsdk:"comment_on_event_enabled"`
	IssuesEnabled                types.Bool   `tfsdk:"issues_enabled"`
	ProjectKeys                  types.List   `tfsdk:"project_keys"`
	UseInheritedSettings         types.Bool   `tfsdk:"use_inherited_settings"`
	Title                        types.String `tfsdk:"title"`
	CreatedAt                    types.String `tfsdk:"created_at"`
	UpdatedAt                    types.String `tfsdk:"updated_at"`
	Active                       types.Bool   `tfsdk:"active"`
}

func (r *gitlabProjectIntegrationJiraResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationJiraResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: r.ResourceDescription,
		DeprecationMessage:  r.DeprecationMessage,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID of the project you want to activate integration on.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to the JIRA project which is being linked to this GitLab project. For example, https://jira.example.com.",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The base URL to the Jira instance API. Web URL value is used if not set. For example, https://jira-api.example.com.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The email or username to be used with Jira. For Jira Cloud use an email, for Jira Data Center and Jira Server use a username. Required when using Basic authentication (jira_auth_type is 0).",
				Optional:            true,
				Computed:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The Jira API token, password, or personal access token to be used with Jira. When your authentication method is basic (jira_auth_type is 0), use an API token for Jira Cloud or a password for Jira Data Center or Jira Server. When your authentication method is a Jira personal access token (jira_auth_type is 1), use the personal access token.",
				Required:            true,
				Sensitive:           true,
			},
			"jira_auth_type": schema.Int64Attribute{
				MarkdownDescription: "The authentication method to be used with Jira. 0 means Basic Authentication. 1 means Jira personal access token. Defaults to 0.",
				Optional:            true,
				Computed:            true,
			},
			"jira_issue_prefix": schema.StringAttribute{
				MarkdownDescription: "Prefix to match Jira issue keys.",
				Optional:            true,
				Computed:            true,
			},
			"jira_issue_regex": schema.StringAttribute{
				MarkdownDescription: "Regular expression to match Jira issue keys.",
				Optional:            true,
				Computed:            true,
			},
			"jira_issue_transition_automatic": schema.BoolAttribute{
				MarkdownDescription: "Enable automatic issue transitions. Takes precedence over jira_issue_transition_id if enabled. Defaults to false. This value cannot be imported, and will not perform drift detection if changed outside Terraform.",
				Optional:            true,
			},
			"jira_issue_transition_id": schema.StringAttribute{
				MarkdownDescription: "The ID of a transition that moves issues to a closed state. You can find this number under the JIRA workflow administration (Administration > Issues > Workflows) by selecting View under Operations of the desired workflow of your project. By default, this ID is set to 2.",
				Optional:            true,
				Computed:            true,
			},
			"commit_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for commit events",
				Optional:            true,
				Computed:            true,
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge request events",
				Optional:            true,
				Computed:            true,
			},
			"comment_on_event_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable comments inside Jira issues on each GitLab event (commit / merge request)",
				Optional:            true,
				Computed:            true,
			},
			"issues_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable viewing Jira issues in GitLab.",
				Optional:            true,
				Computed:            true,
			},
			"project_keys": schema.ListAttribute{
				MarkdownDescription: "Keys of Jira projects. When issues_enabled is true, this setting specifies which Jira projects to view issues from in GitLab.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"use_inherited_settings": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether or not to inherit default settings. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Title of the integration.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this integration was activated at in UTC.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this integration was last updated at in UTC.",
				Computed:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the integration is active.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectIntegrationJiraResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationJiraResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectIntegrationJiraResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error creating Jira integration for project: %s", err.Error()))
		return
	}

	projectID := data.Project.ValueString()
	data.ID = types.StringValue(projectID)
	resp.Diagnostics.Append(data.modelToStateModel(service, projectID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationJiraResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectIntegrationJiraResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()
	service, _, err := r.client.Services.GetJiraService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Jira integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading Jira integration for project: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(service, projectID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationJiraResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectIntegrationJiraResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Jira integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error updating Jira integration for project: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(service, data.Project.ValueString())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationJiraResource) update(ctx context.Context, data *gitlabProjectIntegrationJiraResourceModel) (*gitlab.JiraService, error) {
	projectID := data.Project.ValueString()
	options := &gitlab.SetJiraServiceOptions{
		URL:                   gitlab.Ptr(data.URL.ValueString()),
		Password:              gitlab.Ptr(data.Password.ValueString()),
		CommitEvents:          data.CommitEvents.ValueBoolPointer(),
		MergeRequestsEvents:   data.MergeRequestsEvents.ValueBoolPointer(),
		CommentOnEventEnabled: data.CommentOnEventEnabled.ValueBoolPointer(),
		UseInheritedSettings:  data.UseInheritedSettings.ValueBoolPointer(),
	}

	if !data.APIURL.IsNull() && !data.APIURL.IsUnknown() {
		options.APIURL = gitlab.Ptr(data.APIURL.ValueString())
	}

	if !data.JiraAuthType.IsNull() && !data.JiraAuthType.IsUnknown() {
		options.JiraAuthType = gitlab.Ptr(data.JiraAuthType.ValueInt64())
	}

	// Handle authentication based on jira_auth_type
	jiraAuthType := int64(0)
	if !data.JiraAuthType.IsNull() && !data.JiraAuthType.IsUnknown() {
		jiraAuthType = data.JiraAuthType.ValueInt64()
	}

	if jiraAuthType == 0 {
		// Basic Auth: send both username and password
		if !data.Username.IsNull() && !data.Username.IsUnknown() {
			options.Username = gitlab.Ptr(data.Username.ValueString())
		}
	}
	// For Token Auth (jiraAuthType == 1), only password is sent (already set above)

	if !data.JiraIssuePrefix.IsNull() && !data.JiraIssuePrefix.IsUnknown() {
		options.JiraIssuePrefix = gitlab.Ptr(data.JiraIssuePrefix.ValueString())
	}

	if !data.JiraIssueRegex.IsNull() && !data.JiraIssueRegex.IsUnknown() {
		options.JiraIssueRegex = gitlab.Ptr(data.JiraIssueRegex.ValueString())
	}

	if !data.JiraIssueTransitionAutomatic.IsNull() && !data.JiraIssueTransitionAutomatic.IsUnknown() {
		options.JiraIssueTransitionAutomatic = gitlab.Ptr(data.JiraIssueTransitionAutomatic.ValueBool())
	}

	if !data.JiraIssueTransitionID.IsNull() && !data.JiraIssueTransitionID.IsUnknown() {
		options.JiraIssueTransitionID = gitlab.Ptr(data.JiraIssueTransitionID.ValueString())
	}

	if !data.IssuesEnabled.IsNull() && !data.IssuesEnabled.IsUnknown() {
		options.IssuesEnabled = gitlab.Ptr(data.IssuesEnabled.ValueBool())
	}

	if !data.ProjectKeys.IsNull() && !data.ProjectKeys.IsUnknown() {
		var projectKeys []string
		data.ProjectKeys.ElementsAs(ctx, &projectKeys, false)
		options.ProjectKeys = &projectKeys
	}

	tflog.Debug(ctx, "Update GitLab Jira integration", map[string]any{
		"options": options,
	})

	_, _, err := r.client.Services.SetJiraService(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	service, _, err := r.client.Services.GetJiraService(projectID, gitlab.WithContext(ctx))
	return service, err
}

func (r *gitlabProjectIntegrationJiraResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationJiraResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	_, err := r.client.Services.DeleteJiraService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Jira integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting Jira integration for project: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationJiraResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectIntegrationJiraResourceModel) modelToStateModel(service *gitlab.JiraService, projectID string) diag.Diagnostics {
	d.Project = types.StringValue(projectID)
	d.URL = types.StringValue(service.Properties.URL)
	d.APIURL = types.StringValue(service.Properties.APIURL)
	d.Username = types.StringValue(service.Properties.Username)
	d.JiraAuthType = types.Int64Value(service.Properties.JiraAuthType)
	d.JiraIssuePrefix = types.StringValue(service.Properties.JiraIssuePrefix)
	d.JiraIssueRegex = types.StringValue(service.Properties.JiraIssueRegex)
	d.JiraIssueTransitionID = types.StringValue(service.Properties.JiraIssueTransitionID)
	// Note: jira_issue_transition_automatic is not returned via API so we cannot read it for the state.
	d.CommitEvents = types.BoolValue(service.CommitEvents)
	d.MergeRequestsEvents = types.BoolValue(service.MergeRequestsEvents)
	d.CommentOnEventEnabled = types.BoolValue(service.CommentOnEventEnabled)
	d.IssuesEnabled = types.BoolValue(service.Properties.IssuesEnabled)
	d.UseInheritedSettings = types.BoolValue(service.Inherited)
	d.Title = types.StringValue(service.Title)
	d.Active = types.BoolValue(service.Active)
	d.CreatedAt = types.StringValue(service.CreatedAt.Format(time.RFC3339))
	if service.UpdatedAt == nil {
		d.UpdatedAt = types.StringNull()
	} else {
		d.UpdatedAt = types.StringValue(service.UpdatedAt.Format(time.RFC3339))
	}

	// Convert project_keys to types.List
	if len(service.Properties.ProjectKeys) > 0 {
		projectKeysList := make([]types.String, len(service.Properties.ProjectKeys))
		for i, key := range service.Properties.ProjectKeys {
			projectKeysList[i] = types.StringValue(key)
		}
		projectKeys, diag := types.ListValueFrom(context.Background(), types.StringType, projectKeysList)
		if diag.HasError() {
			return diag
		}
		d.ProjectKeys = projectKeys
	} else {
		d.ProjectKeys = types.ListNull(types.StringType)
	}
	return nil
}

// MoveState implements the ResourceWithMoveState interface to support moving state from the deprecated gitlab_integration_jira resource.
// This enables users to migrate from gitlab_integration_jira to gitlab_project_integration_jira using Terraform's moved block.
// Note: Cross-resource-type state moves require Terraform 1.8 or later.
func (r *gitlabProjectIntegrationJiraResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		// This first StateMover implements the migration from
		// `gitlab_integration_jira` -> `gitlab_project_integration_jira`.
		// The SourceSchema needs to match the deprecated `gitlab_integration_jira` as a result.
		{
			SourceSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"project": schema.StringAttribute{
						Required: true,
					},
					"url": schema.StringAttribute{
						Required: true,
					},
					"api_url": schema.StringAttribute{
						Computed: true,
					},
					"username": schema.StringAttribute{
						Optional: true,
					},
					"password": schema.StringAttribute{
						Required:  true,
						Sensitive: true,
					},
					"jira_auth_type": schema.Int64Attribute{
						Computed: true,
					},
					"jira_issue_prefix": schema.StringAttribute{
						Optional: true,
					},
					"jira_issue_regex": schema.StringAttribute{
						Optional: true,
					},
					"jira_issue_transition_automatic": schema.BoolAttribute{
						Optional: true,
					},
					"jira_issue_transition_id": schema.StringAttribute{
						Optional: true,
					},
					"commit_events": schema.BoolAttribute{
						Computed: true,
					},
					"merge_requests_events": schema.BoolAttribute{
						Computed: true,
					},
					"comment_on_event_enabled": schema.BoolAttribute{
						Computed: true,
					},
					"issues_enabled": schema.BoolAttribute{
						Optional: true,
					},
					"project_keys": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"use_inherited_settings": schema.BoolAttribute{
						Computed: true,
					},
					"title": schema.StringAttribute{
						Computed: true,
					},
					"created_at": schema.StringAttribute{
						Computed: true,
					},
					"updated_at": schema.StringAttribute{
						Computed: true,
					},
					"active": schema.BoolAttribute{
						Computed: true,
					},
				},
			},
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Only handle moves from gitlab_integration_jira resource
				if req.SourceTypeName != "gitlab_integration_jira" {
					resp.Diagnostics.AddError("Invalid source resource type", fmt.Sprintf("Expected source type 'gitlab_integration_jira', got '%s'", req.SourceTypeName))
					return
				}

				// Check provider address (without hostname for compatibility)
				// Accept anything that ends with gitlab, which seems the safest.
				//  hashicorp/gitlab is used in tests
				//  gitlab-org/gitlab is used in production
				//  gitlabhq/gitlab is referenced on the provider docs.
				if !strings.HasSuffix(req.SourceProviderAddress, "gitlab") {
					resp.Diagnostics.AddError("Invalid source provider address", fmt.Sprintf("Expected provider address ending with 'gitlab', got '%s'", req.SourceProviderAddress))
					return
				}

				// Define the source model matching the old gitlab_integration_jira schema
				type sourceModel struct {
					ID                           types.String `tfsdk:"id"`
					Project                      types.String `tfsdk:"project"`
					URL                          types.String `tfsdk:"url"`
					APIURL                       types.String `tfsdk:"api_url"`
					Username                     types.String `tfsdk:"username"`
					Password                     types.String `tfsdk:"password"`
					JiraAuthType                 types.Int64  `tfsdk:"jira_auth_type"`
					JiraIssuePrefix              types.String `tfsdk:"jira_issue_prefix"`
					JiraIssueRegex               types.String `tfsdk:"jira_issue_regex"`
					JiraIssueTransitionAutomatic types.Bool   `tfsdk:"jira_issue_transition_automatic"`
					JiraIssueTransitionID        types.String `tfsdk:"jira_issue_transition_id"`
					CommitEvents                 types.Bool   `tfsdk:"commit_events"`
					MergeRequestsEvents          types.Bool   `tfsdk:"merge_requests_events"`
					CommentOnEventEnabled        types.Bool   `tfsdk:"comment_on_event_enabled"`
					IssuesEnabled                types.Bool   `tfsdk:"issues_enabled"`
					ProjectKeys                  types.List   `tfsdk:"project_keys"`
					UseInheritedSettings         types.Bool   `tfsdk:"use_inherited_settings"`
					Title                        types.String `tfsdk:"title"`
					CreatedAt                    types.String `tfsdk:"created_at"`
					UpdatedAt                    types.String `tfsdk:"updated_at"`
					Active                       types.Bool   `tfsdk:"active"`
				}

				var sourceStateData sourceModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				project := sourceStateData.ID.ValueString()

				// Create the target state data
				targetStateData := gitlabProjectIntegrationJiraResourceModel{
					ID:                           types.StringValue(project),
					Project:                      sourceStateData.Project,
					URL:                          sourceStateData.URL,
					APIURL:                       sourceStateData.APIURL,
					Username:                     sourceStateData.Username,
					Password:                     sourceStateData.Password,
					JiraAuthType:                 sourceStateData.JiraAuthType,
					JiraIssuePrefix:              sourceStateData.JiraIssuePrefix,
					JiraIssueRegex:               sourceStateData.JiraIssueRegex,
					JiraIssueTransitionAutomatic: sourceStateData.JiraIssueTransitionAutomatic,
					JiraIssueTransitionID:        sourceStateData.JiraIssueTransitionID,
					CommitEvents:                 sourceStateData.CommitEvents,
					MergeRequestsEvents:          sourceStateData.MergeRequestsEvents,
					CommentOnEventEnabled:        sourceStateData.CommentOnEventEnabled,
					IssuesEnabled:                sourceStateData.IssuesEnabled,
					ProjectKeys:                  sourceStateData.ProjectKeys,
					UseInheritedSettings:         sourceStateData.UseInheritedSettings,
					Title:                        sourceStateData.Title,
					CreatedAt:                    sourceStateData.CreatedAt,
					UpdatedAt:                    sourceStateData.UpdatedAt,
					Active:                       sourceStateData.Active,
				}

				tflog.Debug(ctx, "Moving state from gitlab_integration_jira to gitlab_project_integration_jira")
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}
