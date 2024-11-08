package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabIntegrationTelegramResource{}
	_ resource.ResourceWithConfigure   = &gitlabIntegrationTelegramResource{}
	_ resource.ResourceWithImportState = &gitlabIntegrationTelegramResource{}
)

func init() {
	registerResource(NewGitlabIntegrationTelegramResource)
}

func NewGitlabIntegrationTelegramResource() resource.Resource {
	return &gitlabIntegrationTelegramResource{}
}

type gitlabIntegrationTelegramResourceModel struct {
	Id                        types.String `tfsdk:"id"`
	Project                   types.String `tfsdk:"project"`
	Token                     types.String `tfsdk:"token"`
	Room                      types.String `tfsdk:"room"`
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
}

func (r *gitlabIntegrationTelegramResourceModel) TelegramServiceToStateModel(service *gitlab.TelegramService, projectId string) {
	r.Id = types.StringValue(projectId)
	r.Project = types.StringValue(projectId)
	r.Room = types.StringValue(service.Properties.Room)
	r.NotifyOnlyBrokenPipelines = types.BoolValue(service.Properties.NotifyOnlyBrokenPipelines)
	r.BranchesToBeNotified = types.StringValue(service.Properties.BranchesToBeNotified)
	r.PushEvents = types.BoolValue(service.PushEvents)
	r.IssuesEvents = types.BoolValue(service.IssuesEvents)
	r.ConfidentialIssuesEvents = types.BoolValue(service.ConfidentialIssuesEvents)
	r.MergeRequestsEvents = types.BoolValue(service.MergeRequestsEvents)
	r.TagPushEvents = types.BoolValue(service.TagPushEvents)
	r.NoteEvents = types.BoolValue(service.NoteEvents)
	r.ConfidentialNoteEvents = types.BoolValue(service.ConfidentialNoteEvents)
	r.PipelineEvents = types.BoolValue(service.PipelineEvents)
	r.WikiPageEvents = types.BoolValue(service.WikiPageEvents)
}

type gitlabIntegrationTelegramResource struct {
	client *gitlab.Client
}

func (r *gitlabIntegrationTelegramResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_telegram"
}

func (r *gitlabIntegrationTelegramResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_integration_telegram`" + ` resource allows to manage the lifecycle of a project integration with Telegram.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/integrations.html#telegram)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project to integrate with Telegram.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The Telegram bot token.",
				Required:            true,
				Sensitive:           true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"room": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the target chat or the username of the target channel (in the format `@channelusername`)",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"notify_only_broken_pipelines": schema.BoolAttribute{
				MarkdownDescription: "Send notifications for broken pipelines.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"branches_to_be_notified": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Branches to send notifications for (introduced in GitLab 16.5). Update of this attribute was not supported before Gitlab 16.11 due to API bug. Valid options are %s.", utils.RenderValueListForDocs(api.ValidBranchesToBeNotified)),
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidBranchesToBeNotified...)},
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push events.",
				Required:            true,
			},
			"issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for issues events.",
				Required:            true,
			},
			"confidential_issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential issues events.",
				Required:            true,
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge requests events.",
				Required:            true,
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for tag push events.",
				Required:            true,
			},
			"note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for note events.",
				Required:            true,
			},
			"confidential_note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential note events.",
				Required:            true,
			},
			"pipeline_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for pipeline events.",
				Required:            true,
			},
			"wiki_page_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for wiki page events.",
				Required:            true,
			},
		},
	}
}

func (r *gitlabIntegrationTelegramResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabIntegrationTelegramResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create telegram integration", err.Error())
	}
}

func (r *gitlabIntegrationTelegramResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabIntegrationTelegramResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.Id.ValueString()

	service, _, err := r.client.Services.GetTelegramService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "telegram integration doesn't exist, removing from state", map[string]interface{}{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading telegram integration for project %s", err.Error()))
		return
	}

	data.TelegramServiceToStateModel(service, projectId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabIntegrationTelegramResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update telegram integration", err.Error())
	}
}

func (r *gitlabIntegrationTelegramResource) update(ctx context.Context, plan *tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) error {
	var data gitlabIntegrationTelegramResourceModel
	diags.Append(plan.Get(ctx, &data)...)
	if diags.HasError() {
		return nil
	}
	projectId := data.Project.ValueString()

	options := &gitlab.SetTelegramServiceOptions{
		Token:                     data.Token.ValueStringPointer(),
		Room:                      data.Room.ValueStringPointer(),
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
	}

	if _, err := r.client.Services.SetTelegramService(projectId, options, gitlab.WithContext(ctx)); err != nil {
		return err
	}

	service, _, err := r.client.Services.GetTelegramService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		return err
	}

	data.TelegramServiceToStateModel(service, projectId)

	diags.Append(state.Set(ctx, &data)...)

	return nil
}

func (r *gitlabIntegrationTelegramResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabIntegrationTelegramResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.Id.ValueString()

	if _, err := r.client.Services.DeleteTelegramService(projectId, gitlab.WithContext(ctx)); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "telegram integration doesn't exist, removing from state", map[string]interface{}{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting telegram integration for project %s", err.Error()))
		return
	}
}

func (r *gitlabIntegrationTelegramResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
