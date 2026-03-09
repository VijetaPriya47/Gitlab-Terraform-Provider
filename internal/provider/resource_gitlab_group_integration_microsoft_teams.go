package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabGroupIntegrationMicrosoftTeamsResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupIntegrationMicrosoftTeamsResource{}
	_ resource.ResourceWithImportState = &gitlabGroupIntegrationMicrosoftTeamsResource{}

	allowedBranchesToBeNotified = []string{"all", "default", "protected", "default_and_protected"}
)

func init() {
	registerResource(NewGitlabGroupIntegrationMicrosoftTeamsResource)
}

func NewGitlabGroupIntegrationMicrosoftTeamsResource() resource.Resource {
	return &gitlabGroupIntegrationMicrosoftTeamsResource{}
}

type gitlabGroupIntegrationMicrosoftTeamsResource struct {
	client *gitlab.Client
}

type gitlabGroupIntegrationMicrosoftTeamsResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Group                     types.String `tfsdk:"group"`
	CreatedAt                 types.String `tfsdk:"created_at"`
	UpdatedAt                 types.String `tfsdk:"updated_at"`
	Active                    types.Bool   `tfsdk:"active"`
	Webhook                   types.String `tfsdk:"webhook"`
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
	UseInheritedSettings      types.Bool   `tfsdk:"use_inherited_settings"`
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_integration_microsoft_teams"
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_integration_microsoft_teams`" + ` resource manages the lifecycle of a group integration with Microsoft Teams.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_integrations/#set-up-microsoft-teams-notifications)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "ID of the group you want to activate integration on.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the integration was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the integration was last updated.",
				Computed:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the integration is active.",
				Computed:            true,
			},
			"webhook": schema.StringAttribute{
				MarkdownDescription: "The Microsoft Teams webhook (for example, https://outlook.office.com/webhook/...).",
				Required:            true,
			},
			"notify_only_broken_pipelines": schema.BoolAttribute{
				MarkdownDescription: "Send notifications for broken pipelines.",
				Optional:            true,
			},
			"branches_to_be_notified": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf(`Branches to send notifications for. Valid options are: %s. The default value is "default"`, utils.RenderValueListForDocs(allowedBranchesToBeNotified)),
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedBranchesToBeNotified...),
				},
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push events.",
				Optional:            true,
			},
			"issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for issues events.",
				Optional:            true,
			},
			"confidential_issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential issues events.",
				Optional:            true,
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge requests events.",
				Optional:            true,
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for tag push events.",
				Optional:            true,
			},
			"note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for note events.",
				Optional:            true,
			},
			"confidential_note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential note events.",
				Optional:            true,
			},
			"pipeline_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for pipeline events.",
				Optional:            true,
			},
			"wiki_page_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for wiki page events.",
				Optional:            true,
			},
			"use_inherited_settings": schema.BoolAttribute{
				MarkdownDescription: `Indicates whether to inherit the default settings. Defaults to "false".`,
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.setIntegration(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupIntegrationMicrosoftTeamsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupId := data.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Read GitLab group integration Microsoft Teams for %s", groupId))

	integration, _, err := r.client.Integrations.GetGroupMicrosoftTeamsNotifications(groupId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Microsoft Teams integration doesn't exist, removing from state", map[string]any{
				"group": groupId,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"GitLab API error occurred",
			fmt.Sprintf("Error reading Microsoft Teams integration for group %s: %s", groupId, err.Error()),
		)
		return
	}

	data.groupIntegrationMicrosoftTeamsToStateModel(integration, groupId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.setIntegration(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupIntegrationMicrosoftTeamsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupId := data.Group.ValueString()

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Delete GitLab group integration Microsoft Teams for %s", groupId))

	_, err := r.client.Integrations.DisableGroupMicrosoftTeamsNotifications(groupId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Microsoft Teams integration doesn't exist, removing from state", map[string]any{
				"group": data.Group,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting Microsoft Teams integration for group %s: %s", groupId, err.Error()))
		return
	}
}

func (r *gitlabGroupIntegrationMicrosoftTeamsResource) setIntegration(ctx context.Context, plan *tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var data *gitlabGroupIntegrationMicrosoftTeamsResourceModel
	diags.Append(plan.Get(ctx, &data)...)
	if diags.HasError() {
		return
	}
	groupId := data.Group.ValueString()

	options := &gitlab.SetMicrosoftTeamsNotificationsOptions{
		Webhook:                   data.Webhook.ValueStringPointer(),
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
		UseInheritedSettings:      data.UseInheritedSettings.ValueBoolPointer(),
	}

	integration, _, err := r.client.Integrations.SetGroupMicrosoftTeamsNotifications(groupId, options, gitlab.WithContext(ctx))
	if err != nil {
		diags.AddError(
			"Failed to configure Microsoft Teams group integration",
			fmt.Sprintf("Unable to set GitLab group integration Microsoft Teams: %s", err.Error()),
		)
		return
	}

	data.groupIntegrationMicrosoftTeamsToStateModel(integration, groupId)
	diags.Append(state.Set(ctx, &data)...)
}

func (data *gitlabGroupIntegrationMicrosoftTeamsResourceModel) groupIntegrationMicrosoftTeamsToStateModel(integration *gitlab.MicrosoftTeamsIntegration, groupId string) {
	data.ID = types.StringValue(groupId)
	data.Group = types.StringValue(groupId)

	if integration.CreatedAt != nil {
		data.CreatedAt = types.StringValue(integration.CreatedAt.String())
	}
	if integration.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(integration.UpdatedAt.String())
	}

	// Note: notify_only_broken_pipelines and branches_to_be_notified are not returned by the API.
	// The user's configured values for these fields are preserved in state instead of being overwritten.

	data.PushEvents = types.BoolValue(integration.PushEvents)
	data.IssuesEvents = types.BoolValue(integration.IssuesEvents)
	data.ConfidentialIssuesEvents = types.BoolValue(integration.ConfidentialIssuesEvents)
	data.MergeRequestsEvents = types.BoolValue(integration.MergeRequestsEvents)
	data.TagPushEvents = types.BoolValue(integration.TagPushEvents)
	data.NoteEvents = types.BoolValue(integration.NoteEvents)
	data.ConfidentialNoteEvents = types.BoolValue(integration.ConfidentialNoteEvents)
	data.PipelineEvents = types.BoolValue(integration.PipelineEvents)
	data.WikiPageEvents = types.BoolValue(integration.WikiPageEvents)
	data.Active = types.BoolValue(integration.Active)
	data.UseInheritedSettings = types.BoolValue(integration.Inherited)
}
