package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIntegrationMattermostResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationMattermostResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationMattermostResource{}
	_ resource.ResourceWithMoveState   = &gitlabProjectIntegrationMattermostResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationMattermostResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationMattermostResource)
}

func NewGitLabProjectIntegrationMattermostResource() resource.Resource {
	return &gitlabProjectIntegrationMattermostResource{
		ResourceName: "_project_integration_mattermost",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_mattermost` + "`" + ` resource manages the lifecycle of a project integration with Mattermost.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#mattermost-notifications)`,
	}
}

// Remove in 19.0
func NewGitLabIntegrationMattermostResource() resource.Resource {
	return &gitlabProjectIntegrationMattermostResource{
		ResourceName: "_integration_mattermost",
		ResourceDescription: `The ` + "`" + `gitlab_integration_mattermost` + "`" + ` resource manages the lifecycle of a project integration with Mattermost.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_mattermost` + "`" + ` instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#mattermost-notifications)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_mattermost` instead.",
	}
}

type gitlabProjectIntegrationMattermostResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_mattermost`
	// and `gitlab_integration_mattermost` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationMattermostResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Project                   types.String `tfsdk:"project"`
	Webhook                   types.String `tfsdk:"webhook"`
	Username                  types.String `tfsdk:"username"`
	NotifyOnlyBrokenPipelines types.Bool   `tfsdk:"notify_only_broken_pipelines"`
	BranchesToBeNotified      types.String `tfsdk:"branches_to_be_notified"`
	PushEvents                types.Bool   `tfsdk:"push_events"`
	IssuesEvents              types.Bool   `tfsdk:"issues_events"`
	ConfidentialIssuesEvents  types.Bool   `tfsdk:"confidential_issues_events"`
	MergeRequestsEvents       types.Bool   `tfsdk:"merge_requests_events"`
	TagPushEvents             types.Bool   `tfsdk:"tag_push_events"`
	NoteEvents                types.Bool   `tfsdk:"note_events"`
	ConfidentialNoteEvents    types.Bool   `tfsdk:"confidential_note_events"`
	PipelineEvents            types.Bool   `tfsdk:"pipeline_events"`
	WikiPageEvents            types.Bool   `tfsdk:"wiki_page_events"`
	PushChannel               types.String `tfsdk:"push_channel"`
	IssueChannel              types.String `tfsdk:"issue_channel"`
	ConfidentialIssueChannel  types.String `tfsdk:"confidential_issue_channel"`
	MergeRequestChannel       types.String `tfsdk:"merge_request_channel"`
	NoteChannel               types.String `tfsdk:"note_channel"`
	ConfidentialNoteChannel   types.String `tfsdk:"confidential_note_channel"`
	TagPushChannel            types.String `tfsdk:"tag_push_channel"`
	PipelineChannel           types.String `tfsdk:"pipeline_channel"`
	WikiPageChannel           types.String `tfsdk:"wiki_page_channel"`
}

func (r *gitlabProjectIntegrationMattermostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationMattermostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"webhook": schema.StringAttribute{
				MarkdownDescription: "Webhook URL (Example, https://mattermost.yourdomain.com/hooks/...). This value cannot be imported.",
				Required:            true,
				Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username to use.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"notify_only_broken_pipelines": schema.BoolAttribute{
				MarkdownDescription: "Send notifications for broken pipelines.",
				Optional:            true,
				Computed:            true,
			},
			"branches_to_be_notified": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Branches to send notifications for. Valid values are %s.", utils.RenderValueListForDocs(api.ValidBranchesToBeNotified)),
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidBranchesToBeNotified...)},
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push events.",
				Optional:            true,
				Computed:            true,
			},
			"issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for issues events.",
				Optional:            true,
				Computed:            true,
			},
			"confidential_issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential issues events.",
				Optional:            true,
				Computed:            true,
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge requests events.",
				Optional:            true,
				Computed:            true,
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for tag push events.",
				Optional:            true,
				Computed:            true,
			},
			"note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for note events.",
				Optional:            true,
				Computed:            true,
			},
			"confidential_note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential note events.",
				Optional:            true,
				Computed:            true,
			},
			"pipeline_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for pipeline events.",
				Optional:            true,
				Computed:            true,
			},
			"wiki_page_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for wiki page events.",
				Optional:            true,
				Computed:            true,
			},
			"push_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive push events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"issue_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive issue events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"confidential_issue_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive confidential issue events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"merge_request_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive merge request events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"note_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive note events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"confidential_note_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive confidential note events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tag_push_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive tag push events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pipeline_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive pipeline events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wiki_page_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive wiki page events notifications.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *gitlabProjectIntegrationMattermostResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationMattermostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectIntegrationMattermostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error creating Mattermost integration for project: %s", err.Error()))
		return
	}

	projectID := data.Project.ValueString()
	data.ID = types.StringValue(projectID)
	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationMattermostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectIntegrationMattermostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()
	service, _, err := r.client.Services.GetMattermostService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Mattermost integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading Mattermost integration for project: %s", err.Error()))
		return
	}

	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationMattermostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectIntegrationMattermostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Mattermost integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error updating Mattermost integration for project: %s", err.Error()))
		return
	}

	data.modelToStateModel(service, data.Project.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationMattermostResource) update(ctx context.Context, data *gitlabProjectIntegrationMattermostResourceModel) (*gitlab.MattermostService, error) {
	projectID := data.Project.ValueString()
	options := &gitlab.SetMattermostServiceOptions{
		WebHook:                   data.Webhook.ValueStringPointer(),
		Username:                  data.Username.ValueStringPointer(),
		NotifyOnlyBrokenPipelines: data.NotifyOnlyBrokenPipelines.ValueBoolPointer(),
		BranchesToBeNotified:      data.BranchesToBeNotified.ValueStringPointer(),
		PushEvents:                data.PushEvents.ValueBoolPointer(),
		IssuesEvents:              data.IssuesEvents.ValueBoolPointer(),
		ConfidentialIssuesEvents:  data.ConfidentialIssuesEvents.ValueBoolPointer(),
		MergeRequestsEvents:       data.MergeRequestsEvents.ValueBoolPointer(),
		TagPushEvents:             data.TagPushEvents.ValueBoolPointer(),
		NoteEvents:                data.NoteEvents.ValueBoolPointer(),
		ConfidentialNoteEvents:    data.ConfidentialNoteEvents.ValueBoolPointer(),
		PipelineEvents:            data.PipelineEvents.ValueBoolPointer(),
		WikiPageEvents:            data.WikiPageEvents.ValueBoolPointer(),
		PushChannel:               data.PushChannel.ValueStringPointer(),
		IssueChannel:              data.IssueChannel.ValueStringPointer(),
		ConfidentialIssueChannel:  data.ConfidentialIssueChannel.ValueStringPointer(),
		MergeRequestChannel:       data.MergeRequestChannel.ValueStringPointer(),
		NoteChannel:               data.NoteChannel.ValueStringPointer(),
		ConfidentialNoteChannel:   data.ConfidentialNoteChannel.ValueStringPointer(),
		TagPushChannel:            data.TagPushChannel.ValueStringPointer(),
		PipelineChannel:           data.PipelineChannel.ValueStringPointer(),
		WikiPageChannel:           data.WikiPageChannel.ValueStringPointer(),
	}

	tflog.Debug(ctx, "Update GitLab Mattermost integration", map[string]any{
		"options": options,
	})

	_, _, err := r.client.Services.SetMattermostService(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	service, _, err := r.client.Services.GetMattermostService(projectID, gitlab.WithContext(ctx))
	return service, err
}

func (r *gitlabProjectIntegrationMattermostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectIntegrationMattermostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	_, err := r.client.Services.DeleteMattermostService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Mattermost integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting Mattermost integration for project: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationMattermostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectIntegrationMattermostResourceModel) modelToStateModel(service *gitlab.MattermostService, projectID string) {
	d.Project = types.StringValue(projectID)
	// The webhook is explicitly not set anymore, due to being removed from the API. It will now
	// use whatever is in the configuration to determine the value.
	// See https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/1421 for more info.
	d.Username = types.StringValue(service.Properties.Username)
	d.NotifyOnlyBrokenPipelines = types.BoolValue(bool(service.Properties.NotifyOnlyBrokenPipelines))
	d.BranchesToBeNotified = types.StringValue(service.Properties.BranchesToBeNotified)
	d.PushEvents = types.BoolValue(service.PushEvents)
	d.IssuesEvents = types.BoolValue(service.IssuesEvents)
	d.ConfidentialIssuesEvents = types.BoolValue(service.ConfidentialIssuesEvents)
	d.MergeRequestsEvents = types.BoolValue(service.MergeRequestsEvents)
	d.TagPushEvents = types.BoolValue(service.TagPushEvents)
	d.NoteEvents = types.BoolValue(service.NoteEvents)
	d.ConfidentialNoteEvents = types.BoolValue(service.ConfidentialNoteEvents)
	d.PipelineEvents = types.BoolValue(service.PipelineEvents)
	d.WikiPageEvents = types.BoolValue(service.WikiPageEvents)
	d.PushChannel = types.StringValue(service.Properties.PushChannel)
	d.IssueChannel = types.StringValue(service.Properties.IssueChannel)
	d.ConfidentialIssueChannel = types.StringValue(service.Properties.ConfidentialIssueChannel)
	d.MergeRequestChannel = types.StringValue(service.Properties.MergeRequestChannel)
	d.NoteChannel = types.StringValue(service.Properties.NoteChannel)
	d.ConfidentialNoteChannel = types.StringValue(service.Properties.ConfidentialNoteChannel)
	d.TagPushChannel = types.StringValue(service.Properties.TagPushChannel)
	d.PipelineChannel = types.StringValue(service.Properties.PipelineChannel)
	d.WikiPageChannel = types.StringValue(service.Properties.WikiPageChannel)
}

// MoveState implements the ResourceWithMoveState interface to support moving state from the deprecated gitlab_integration_mattermost resource.
// This enables users to migrate from gitlab_integration_mattermost to gitlab_project_integration_mattermost using Terraform's moved block.
// Note: Cross-resource-type state moves require Terraform 1.8 or later.
func (r *gitlabProjectIntegrationMattermostResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		// This first StateMover implements the migration from
		// `gitlab_integration_mattermost` -> `gitlab_project_integration_mattermost`.
		// The SourceSchema needs to match the deprecated `gitlab_integration_mattermost` as a result.
		{
			SourceSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"project": schema.StringAttribute{
						Required: true,
					},
					"webhook": schema.StringAttribute{
						Required:  true,
						Sensitive: true,
					},
					"username": schema.StringAttribute{
						Computed: true,
					},
					"notify_only_broken_pipelines": schema.BoolAttribute{
						Computed: true,
					},
					"branches_to_be_notified": schema.StringAttribute{
						Computed: true,
					},
					"push_events": schema.BoolAttribute{
						Computed: true,
					},
					"issues_events": schema.BoolAttribute{
						Computed: true,
					},
					"confidential_issues_events": schema.BoolAttribute{
						Computed: true,
					},
					"merge_requests_events": schema.BoolAttribute{
						Computed: true,
					},
					"tag_push_events": schema.BoolAttribute{
						Computed: true,
					},
					"note_events": schema.BoolAttribute{
						Computed: true,
					},
					"confidential_note_events": schema.BoolAttribute{
						Computed: true,
					},
					"pipeline_events": schema.BoolAttribute{
						Computed: true,
					},
					"wiki_page_events": schema.BoolAttribute{
						Computed: true,
					},
					"push_channel": schema.StringAttribute{
						Computed: true,
					},
					"issue_channel": schema.StringAttribute{
						Computed: true,
					},
					"confidential_issue_channel": schema.StringAttribute{
						Computed: true,
					},
					"merge_request_channel": schema.StringAttribute{
						Computed: true,
					},
					"note_channel": schema.StringAttribute{
						Computed: true,
					},
					"confidential_note_channel": schema.StringAttribute{
						Computed: true,
					},
					"tag_push_channel": schema.StringAttribute{
						Computed: true,
					},
					"pipeline_channel": schema.StringAttribute{
						Computed: true,
					},
					"wiki_page_channel": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Only handle moves from gitlab_integration_mattermost resource
				if req.SourceTypeName != "gitlab_integration_mattermost" {
					resp.Diagnostics.AddError("Invalid source resource type", fmt.Sprintf("Expected source type 'gitlab_integration_mattermost', got '%s'", req.SourceTypeName))
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

				// Define the source model matching the old gitlab_integration_mattermost schema
				type sourceModel struct {
					ID                        types.String `tfsdk:"id"`
					Project                   types.String `tfsdk:"project"`
					Webhook                   types.String `tfsdk:"webhook"`
					Username                  types.String `tfsdk:"username"`
					NotifyOnlyBrokenPipelines types.Bool   `tfsdk:"notify_only_broken_pipelines"`
					BranchesToBeNotified      types.String `tfsdk:"branches_to_be_notified"`
					PushEvents                types.Bool   `tfsdk:"push_events"`
					IssuesEvents              types.Bool   `tfsdk:"issues_events"`
					ConfidentialIssuesEvents  types.Bool   `tfsdk:"confidential_issues_events"`
					MergeRequestsEvents       types.Bool   `tfsdk:"merge_requests_events"`
					TagPushEvents             types.Bool   `tfsdk:"tag_push_events"`
					NoteEvents                types.Bool   `tfsdk:"note_events"`
					ConfidentialNoteEvents    types.Bool   `tfsdk:"confidential_note_events"`
					PipelineEvents            types.Bool   `tfsdk:"pipeline_events"`
					WikiPageEvents            types.Bool   `tfsdk:"wiki_page_events"`
					PushChannel               types.String `tfsdk:"push_channel"`
					IssueChannel              types.String `tfsdk:"issue_channel"`
					ConfidentialIssueChannel  types.String `tfsdk:"confidential_issue_channel"`
					MergeRequestChannel       types.String `tfsdk:"merge_request_channel"`
					NoteChannel               types.String `tfsdk:"note_channel"`
					ConfidentialNoteChannel   types.String `tfsdk:"confidential_note_channel"`
					TagPushChannel            types.String `tfsdk:"tag_push_channel"`
					PipelineChannel           types.String `tfsdk:"pipeline_channel"`
					WikiPageChannel           types.String `tfsdk:"wiki_page_channel"`
				}

				var sourceStateData sourceModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				project := sourceStateData.ID.ValueString()

				// Create the target state data
				targetStateData := gitlabProjectIntegrationMattermostResourceModel{
					ID:                        types.StringValue(project),
					Project:                   sourceStateData.Project,
					Webhook:                   sourceStateData.Webhook,
					Username:                  sourceStateData.Username,
					NotifyOnlyBrokenPipelines: sourceStateData.NotifyOnlyBrokenPipelines,
					BranchesToBeNotified:      sourceStateData.BranchesToBeNotified,
					PushEvents:                sourceStateData.PushEvents,
					IssuesEvents:              sourceStateData.IssuesEvents,
					ConfidentialIssuesEvents:  sourceStateData.ConfidentialIssuesEvents,
					MergeRequestsEvents:       sourceStateData.MergeRequestsEvents,
					TagPushEvents:             sourceStateData.TagPushEvents,
					NoteEvents:                sourceStateData.NoteEvents,
					ConfidentialNoteEvents:    sourceStateData.ConfidentialNoteEvents,
					PipelineEvents:            sourceStateData.PipelineEvents,
					WikiPageEvents:            sourceStateData.WikiPageEvents,
					PushChannel:               sourceStateData.PushChannel,
					IssueChannel:              sourceStateData.IssueChannel,
					ConfidentialIssueChannel:  sourceStateData.ConfidentialIssueChannel,
					MergeRequestChannel:       sourceStateData.MergeRequestChannel,
					NoteChannel:               sourceStateData.NoteChannel,
					ConfidentialNoteChannel:   sourceStateData.ConfidentialNoteChannel,
					TagPushChannel:            sourceStateData.TagPushChannel,
					PipelineChannel:           sourceStateData.PipelineChannel,
					WikiPageChannel:           sourceStateData.WikiPageChannel,
				}

				tflog.Debug(ctx, "Moving state from gitlab_integration_mattermost to gitlab_project_integration_mattermost")
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}
