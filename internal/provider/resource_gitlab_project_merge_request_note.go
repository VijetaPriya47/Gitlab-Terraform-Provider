package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	_ resource.Resource                = &gitlabProjectMergeRequestNoteResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectMergeRequestNoteResource{}
	_ resource.ResourceWithImportState = &gitlabProjectMergeRequestNoteResource{}
	_ resource.ResourceWithModifyPlan  = &gitlabProjectMergeRequestNoteResource{}
)

func init() {
	registerResource(NewGitlabProjectMergeRequestNoteResource)
}

func NewGitlabProjectMergeRequestNoteResource() resource.Resource {
	return &gitlabProjectMergeRequestNoteResource{}
}

type gitlabProjectMergeRequestNoteResourceModel struct {
	ID                      types.String      `tfsdk:"id"`
	Project                 types.String      `tfsdk:"project"`
	MergeRequestIID         types.Int64       `tfsdk:"merge_request_iid"`
	NoteID                  types.Int64       `tfsdk:"note_id"`
	Body                    types.String      `tfsdk:"body"`
	CreatedAt               timetypes.RFC3339 `tfsdk:"created_at"`
	UpdatedAt               timetypes.RFC3339 `tfsdk:"updated_at"`
	System                  types.Bool        `tfsdk:"system"`
	Internal                types.Bool        `tfsdk:"internal"`
	Resolvable              types.Bool        `tfsdk:"resolvable"`
	MergeRequestDiffHeadSHA types.String      `tfsdk:"merge_request_diff_head_sha"`
}

type gitlabProjectMergeRequestNoteResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectMergeRequestNoteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_merge_request_note"
}

func (r *gitlabProjectMergeRequestNoteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectMergeRequestNoteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectMergeRequestNoteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_project_merge_request_note` + "`" + ` resource manages the lifecycle of a project merge request note.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/notes/#merge-requests)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the merge request note. In the format of `<project>:<merge_request_iid>:<note_id>`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or path of the project to add the note to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"merge_request_iid": schema.Int64Attribute{
				MarkdownDescription: "The IID of the merge request to add the note to.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"note_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the merge request note.",
				Computed:            true,
			},
			"body": schema.StringAttribute{
				MarkdownDescription: "The body of the merge request note.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 1000000)},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The creation date of the merge request note. Using this field requires the token used with the provider to either be an Admin, or hava a Project or Group Owner role.",
				CustomType:          timetypes.RFC3339Type{},
				Computed:            true,
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The last updated date of the merge request note.",
				CustomType:          timetypes.RFC3339Type{},
				Computed:            true,
			},
			"system": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the merge request note is a system note.",
				Computed:            true,
			},
			"internal": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the merge request note is internal.",
				Computed:            true,
				Optional:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"resolvable": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the merge request note is resolvable.",
				Computed:            true,
			},
			"merge_request_diff_head_sha": schema.StringAttribute{
				MarkdownDescription: "The diff head SHA of the merge request when the note was created.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *gitlabProjectMergeRequestNoteResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var data *gitlabProjectMergeRequestNoteResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no check for token permissions is needed")
		return
	}

	if !data.CreatedAt.IsUnknown() && !data.CreatedAt.IsNull() {
		user, _, err := r.client.Users.CurrentUser(gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to check if current user is admin or project owner: %s", err.Error()))
			return
		}
		if user.IsAdmin {
			return
		}
		membership, _, err := r.client.ProjectMembers.GetInheritedProjectMember(data.Project.ValueString(), user.ID, gitlab.WithContext(ctx))
		if err != nil && !api.Is404(err) {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to check if current user is admin or project owner: %s", err.Error()))
			return
		}
		if err != nil || membership.AccessLevel != gitlab.OwnerPermissions {
			resp.Diagnostics.AddAttributeError(path.Root("created_at"), "Attribute Not Permitted", "Using this field requires the token used with the provider to either be an Admin, or have a Project or Group Owner role.")
		}
	}
}

func (r *gitlabProjectMergeRequestNoteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectMergeRequestNoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	options := &gitlab.CreateMergeRequestNoteOptions{
		Body: data.Body.ValueStringPointer(),
	}

	if !data.CreatedAt.IsNull() && !data.CreatedAt.IsUnknown() {
		parsedCreatedAt, err := time.Parse(time.RFC3339, data.CreatedAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid created_at value", fmt.Sprintf("Unable to parse created_at value: %s", err.Error()))
			return
		}
		options.CreatedAt = &parsedCreatedAt
	}

	if !data.Internal.IsNull() && !data.Internal.IsUnknown() {
		options.Internal = data.Internal.ValueBoolPointer()
	}

	if !data.MergeRequestDiffHeadSHA.IsNull() && !data.MergeRequestDiffHeadSHA.IsUnknown() {
		options.MergeRequestDiffHeadSHA = data.MergeRequestDiffHeadSHA.ValueStringPointer()
	}

	project := data.Project.ValueString()
	mergeRequestIID := data.MergeRequestIID.ValueInt64()

	note, _, err := r.client.Notes.CreateMergeRequestNote(project, mergeRequestIID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create project merge request note: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildThreePartID(&project, gitlab.Ptr(strconv.FormatInt(mergeRequestIID, 10)), gitlab.Ptr(strconv.FormatInt(note.ID, 10))))
	resp.Diagnostics.Append(data.modelToStateModel(note, project, mergeRequestIID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectMergeRequestNoteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectMergeRequestNoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project, mergeRequestIID, noteID, err := resourceGitlabProjectMergeRequestNoteParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}
	note, _, err := r.client.Notes.GetMergeRequestNote(project, mergeRequestIID, noteID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occurred", fmt.Sprintf("Project merge request note doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project merge request note: %s", err.Error()))
		return
	}
	resp.Diagnostics.Append(data.modelToStateModel(note, project, mergeRequestIID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectMergeRequestNoteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectMergeRequestNoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()
	mergeRequestIID := data.MergeRequestIID.ValueInt64()
	noteID := data.NoteID.ValueInt64()
	options := &gitlab.UpdateMergeRequestNoteOptions{
		Body: data.Body.ValueStringPointer(),
	}
	note, _, err := r.client.Notes.UpdateMergeRequestNote(project, mergeRequestIID, noteID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update project merge request note: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildThreePartID(&project, gitlab.Ptr(strconv.FormatInt(mergeRequestIID, 10)), gitlab.Ptr(strconv.FormatInt(note.ID, 10))))
	resp.Diagnostics.Append(data.modelToStateModel(note, project, mergeRequestIID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectMergeRequestNoteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectMergeRequestNoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project, mergeRequestIID, noteID, err := resourceGitlabProjectMergeRequestNoteParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}
	_, err = r.client.Notes.DeleteMergeRequestNote(project, mergeRequestIID, noteID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete project merge request note: %s", err.Error()))
		return
	}
	resp.State.RemoveResource(ctx)
}

func (data *gitlabProjectMergeRequestNoteResourceModel) modelToStateModel(note *gitlab.Note, project string, mergeRequestIID int64) diag.Diagnostics {
	data.Project = types.StringValue(project)
	data.MergeRequestIID = types.Int64Value(mergeRequestIID)
	data.NoteID = types.Int64Value(int64(note.ID))
	data.Body = types.StringValue(note.Body)
	createdAt, diags := timetypes.NewRFC3339Value(note.CreatedAt.Format(time.RFC3339))
	if diags.HasError() {
		return diags
	}
	data.CreatedAt = createdAt
	updatedAt, diags := timetypes.NewRFC3339Value(note.UpdatedAt.Format(time.RFC3339))
	if diags.HasError() {
		return diags
	}
	data.UpdatedAt = updatedAt
	data.System = types.BoolValue(note.System)
	data.Internal = types.BoolValue(note.Internal)
	data.Resolvable = types.BoolValue(note.Resolvable)
	return diags
}

func resourceGitlabProjectMergeRequestNoteParseID(id string) (string, int64, int64, error) {
	project, mergeRequestIID, noteID, err := utils.ParseThreePartID(id)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to parse ID: %w", err)
	}
	// convert mergeRequestIID into an int64
	mergeRequestIIDInt, err := strconv.ParseInt(mergeRequestIID, 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to convert merge request IID to int64: %w", err)
	}

	// convert noteID into an int64
	noteIDInt, err := strconv.ParseInt(noteID, 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to convert note ID to int64: %w", err)
	}

	return project, mergeRequestIIDInt, noteIDInt, nil
}
