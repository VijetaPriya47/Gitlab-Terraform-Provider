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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIntegrationMatrixResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationMatrixResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationMatrixResource{}
)

func init() {
	registerResource(NewGitlabProjectIntegrationMatrixResource)
}

func NewGitlabProjectIntegrationMatrixResource() resource.Resource {
	return &gitlabProjectIntegrationMatrixResource{}
}

type gitlabProjectIntegrationMatrixResource struct {
	client *gitlab.Client
}

type gitlabProjectIntegrationMatrixResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Project                   types.String `tfsdk:"project"`
	Hostname                  types.String `tfsdk:"hostname"`
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
	UseInheritedSettings      types.Bool   `tfsdk:"use_inherited_settings"`
}

func (r *gitlabProjectIntegrationMatrixResourceModel) matrixModelToState(service *gitlab.MatrixService, projectId string) {
	r.Project = types.StringValue(projectId)
	r.Hostname = types.StringValue(service.Properties.Hostname)
	// Note: Token is not set here as the API doesn't return it.
	// It's preserved from plan/state by not being overwritten.
	r.Room = types.StringValue(service.Properties.Room)
	r.NotifyOnlyBrokenPipelines = types.BoolValue(bool(service.Properties.NotifyOnlyBrokenPipelines))
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
	r.UseInheritedSettings = types.BoolValue(service.Inherited)
}

func (r *gitlabProjectIntegrationMatrixResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_integration_matrix"
}

func (r *gitlabProjectIntegrationMatrixResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_project_integration_matrix`" + ` resource manages the lifecycle of a project integration with Matrix.
		
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#matrix-notifications)`),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project to integrate with Matrix.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Custom hostname of the Matrix server. The default value is \"https://matrix.org\".",
				Optional:            true,
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The Matrix access token (for example, syt-zyx57W2v1u123ew11).",
				Required:            true,
				Sensitive:           true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"room": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the target room (in the format `!qPKKM111FFKKsfoCVy:matrix.org`).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"notify_only_broken_pipelines": schema.BoolAttribute{
				MarkdownDescription: "Send notifications for broken pipelines.",
				Optional:            true,
				Computed:            true,
			},
			"branches_to_be_notified": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Branches to send notifications for. Valid options are: %s. The default value is \"default\"", utils.RenderValueListForDocs(api.ValidBranchesToBeNotified)),
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(api.ValidBranchesToBeNotified...),
				},
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push events.",
				Optional:            true,
				Computed:            true,
			},
			"issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for issue events.",
				Optional:            true,
				Computed:            true,
			},
			"confidential_issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential issue events.",
				Optional:            true,
				Computed:            true,
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge request events.",
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
			"use_inherited_settings": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether to inherit the default settings. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectIntegrationMatrixResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationMatrixResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectIntegrationMatrixResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	err := r.setMatrixIntegration(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Matrix integration", err.Error())
	}
}

func (r *gitlabProjectIntegrationMatrixResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabProjectIntegrationMatrixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.ID.ValueString()

	service, _, err := r.client.Services.GetMatrixService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Matrix integration does not exist, removing from state", map[string]any{
				"project_id": projectId,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Matrix integration", err.Error())
		return
	}

	data.matrixModelToState(service, projectId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationMatrixResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	err := r.setMatrixIntegration(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Matrix integration", err.Error())
	}
}

func (r *gitlabProjectIntegrationMatrixResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationMatrixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.ID.ValueString()

	if _, err := r.client.Services.DeleteMatrixService(projectId, gitlab.WithContext(ctx)); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Matrix integration does not exist, removing from state", map[string]any{
				"project_id": projectId,
			})
			return
		}
		resp.Diagnostics.AddError("Failed to delete Matrix integration", err.Error())
		return
	}
}

func (r *gitlabProjectIntegrationMatrixResource) setMatrixIntegration(ctx context.Context, plan *tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) error {
	var data gitlabProjectIntegrationMatrixResourceModel
	diags.Append(plan.Get(ctx, &data)...)
	if diags.HasError() {
		return nil
	}
	projectId := data.Project.ValueString()

	options := &gitlab.SetMatrixServiceOptions{
		Hostname:                  data.Hostname.ValueStringPointer(),
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
		UseInheritedSettings:      data.UseInheritedSettings.ValueBoolPointer(),
	}

	if _, _, err := r.client.Services.SetMatrixService(projectId, options, gitlab.WithContext(ctx)); err != nil {
		return err
	}

	service, _, err := r.client.Services.GetMatrixService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		return err
	}

	data.ID = types.StringValue(projectId)
	data.matrixModelToState(service, projectId)

	diags.Append(state.Set(ctx, &data)...)

	return nil
}
