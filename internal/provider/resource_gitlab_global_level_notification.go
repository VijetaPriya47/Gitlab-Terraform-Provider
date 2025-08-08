package provider

import (
	"context"
	"fmt"

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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

const applicationGlobalSettingsID = "gitlab"

var (
	_ resource.Resource                = &gitlabGlobalLevelNotificationsResource{}
	_ resource.ResourceWithConfigure   = &gitlabGlobalLevelNotificationsResource{}
	_ resource.ResourceWithImportState = &gitlabGlobalLevelNotificationsResource{}

	// The allowed notification
	allowedGlobalNotificationLevels = []string{"disabled", "participating", "watch", "mention", "custom"}
)

func init() {
	registerResource(NewgitlabGlobalLevelNotificationsResource)
}

func NewgitlabGlobalLevelNotificationsResource() resource.Resource {
	return &gitlabGlobalLevelNotificationsResource{}
}

type gitlabGlobalLevelNotificationsResource struct {
	client *gitlab.Client
}

type gitlabGlobalLevelNotificationsModel struct {
	ID                        types.String `tfsdk:"id"`
	Level                     types.String `tfsdk:"level"`
	NewNote                   types.Bool   `tfsdk:"new_note"`
	NewIssue                  types.Bool   `tfsdk:"new_issue"`
	ReopenIssue               types.Bool   `tfsdk:"reopen_issue"`
	CloseIssue                types.Bool   `tfsdk:"close_issue"`
	ReassignIssue             types.Bool   `tfsdk:"reassign_issue"`
	IssueDue                  types.Bool   `tfsdk:"issue_due"`
	NewMergeRequest           types.Bool   `tfsdk:"new_merge_request"`
	PushToMergeRequest        types.Bool   `tfsdk:"push_to_merge_request"`
	ReopenMergeRequest        types.Bool   `tfsdk:"reopen_merge_request"`
	CloseMergeRequest         types.Bool   `tfsdk:"close_merge_request"`
	ReassignMergeRequest      types.Bool   `tfsdk:"reassign_merge_request"`
	MergeMergeRequest         types.Bool   `tfsdk:"merge_merge_request"`
	FailedPipeline            types.Bool   `tfsdk:"failed_pipeline"`
	FixedPipeline             types.Bool   `tfsdk:"fixed_pipeline"`
	SuccessPipeline           types.Bool   `tfsdk:"success_pipeline"`
	MovedProject              types.Bool   `tfsdk:"moved_project"`
	MergeWhenPipelineSucceeds types.Bool   `tfsdk:"merge_when_pipeline_succeeds"`
}

// Metadata returns the resource name
func (d *gitlabGlobalLevelNotificationsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_level_notifications"
}

func (d *gitlabGlobalLevelNotificationsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		MarkdownDescription: `The ` + "`" + `gitlab_global_level_notifications` + "`" + ` resource allows to manage global notifications.
		
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/notification_settings/#group--project-level-notification-settings)`,

		// Schema is external because we'll need to re-implement the state migration function.
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the resource.This is a static value named gitlab",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The level of the notification. Valid values are: %s.", utils.RenderValueListForDocs(allowedNotificationLevels)),
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedGlobalNotificationLevels...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"new_note": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for new notes on merge requests. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"new_issue": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for new issues. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"reopen_issue": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for reopened issues. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"close_issue": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for closed issues. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"reassign_issue": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for issue reassignments. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"issue_due": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for due issues. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"new_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for new merge requests. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"push_to_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push to merge request branches. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"reopen_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for reopened merge requests. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"close_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for closed merge requests. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"reassign_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge request reassignments. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"merge_merge_request": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merged merge requests. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"failed_pipeline": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for failed pipelines. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"fixed_pipeline": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for fixed pipelines. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"success_pipeline": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for successful pipelines. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"moved_project": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for moved projects. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
			"merge_when_pipeline_succeeds": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merged merge requests when the pipeline succeeds. Can only be used when `level` is `custom`.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *gitlabGlobalLevelNotificationsResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (d *gitlabGlobalLevelNotificationsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabGlobalLevelNotificationsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGlobalLevelNotificationsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "[DEBUG] Creating gitlab global notification settings", map[string]any{
		"data": data,
	})

	settings, err := d.updateGlobalNotifications(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating gitlab global notification settings", err.Error())
		return
	}

	id := applicationGlobalSettingsID

	data.globalNotificationModelToState(id, settings)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGlobalLevelNotificationsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGlobalLevelNotificationsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	tflog.Debug(ctx, "[DEBUG] Reading gitlab global notification settings", map[string]any{
		"id": id,
	})

	projectNotifications, _, err := d.client.NotificationSettings.GetGlobalSettings()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading gitlab global notification settings",
			err.Error(),
		)
		tflog.Error(ctx, "[ERROR] Error reading gitlab global notification settings", map[string]any{
			"id":    id,
			"error": err,
		})
		return
	}

	data.globalNotificationModelToState(id, projectNotifications)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGlobalLevelNotificationsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGlobalLevelNotificationsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := data.ID.ValueString()
	tflog.Debug(ctx, "[DEBUG] Updating gitlab global notification settings", map[string]any{
		"id": id,
	})

	// Call the update API
	tflog.Debug(ctx, "[DEBUG] Calling Update Project Settings API with project and data", map[string]any{
		"id":   id,
		"data": data,
	})

	settings, err := d.updateGlobalNotifications(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Error updating gitlab global notification settings", err.Error())
		return
	}

	data.globalNotificationModelToState(id, settings)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGlobalLevelNotificationsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabGlobalLevelNotificationsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// There's no "delete" API for notifications, so we just set the notification
	// so we just disable them
	disabled := globalNotificationLevelTypes["disabled"]
	options := &gitlab.NotificationSettingsOptions{
		Level: &disabled,
	}
	_, _, err := d.client.NotificationSettings.UpdateGlobalSettings(options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error encountered when updating settings back to watch for resource ID %s", data.ID), err.Error())
		tflog.Error(ctx, "Error encountered when updating settings back to global", map[string]any{
			"error": err,
		})
	}
}

// ValidateConfig function
func (d *gitlabGlobalLevelNotificationsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// If "level" is custom, other values can be set. If not, they can't be.
	var data gitlabGlobalLevelNotificationsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Skip validation if any custom notification fields are unknown
	// This handles the case where for_each with optional variables causes unknown values during planning
	if data.NewNote.IsUnknown() ||
		data.NewIssue.IsUnknown() ||
		data.ReopenIssue.IsUnknown() ||
		data.CloseIssue.IsUnknown() ||
		data.ReassignIssue.IsUnknown() ||
		data.IssueDue.IsUnknown() ||
		data.NewMergeRequest.IsUnknown() ||
		data.PushToMergeRequest.IsUnknown() ||
		data.ReopenMergeRequest.IsUnknown() ||
		data.CloseMergeRequest.IsUnknown() ||
		data.ReassignMergeRequest.IsUnknown() ||
		data.MergeMergeRequest.IsUnknown() ||
		data.FailedPipeline.IsUnknown() ||
		data.FixedPipeline.IsUnknown() ||
		data.SuccessPipeline.IsUnknown() ||
		data.MovedProject.IsUnknown() ||
		data.MergeWhenPipelineSucceeds.IsUnknown() ||
		data.Level.IsUnknown() {
		tflog.Debug(ctx, "Custom notification fields or level have unknown values, skipping validation")
		return
	}

	// Check if any custom values are set
	customOptionsSet := false
	if !data.NewNote.IsNull() ||
		!data.NewIssue.IsNull() ||
		!data.ReopenIssue.IsNull() ||
		!data.CloseIssue.IsNull() ||
		!data.ReassignIssue.IsNull() ||
		!data.IssueDue.IsNull() ||
		!data.NewMergeRequest.IsNull() ||
		!data.PushToMergeRequest.IsNull() ||
		!data.ReopenMergeRequest.IsNull() ||
		!data.CloseMergeRequest.IsNull() ||
		!data.ReassignMergeRequest.IsNull() ||
		!data.MergeMergeRequest.IsNull() ||
		!data.FailedPipeline.IsNull() ||
		!data.FixedPipeline.IsNull() ||
		!data.SuccessPipeline.IsNull() ||
		!data.MovedProject.IsNull() ||
		!data.MergeWhenPipelineSucceeds.IsNull() {
		customOptionsSet = true
	}

	if customOptionsSet && data.Level.ValueString() != "custom" {
		resp.Diagnostics.AddAttributeError(path.Root("level"),
			`"level" must be set to "custom" to set individual notification levels`,
			`"level" must be set to "custom" to set individual notification levels`)
	}

}

func (d *gitlabGlobalLevelNotificationsModel) globalNotificationModelToState(id string, notifications *gitlab.NotificationSettings) {
	d.ID = types.StringValue(id)

	d.Level = types.StringValue(notifications.Level.String())
	// The "events" object will only be set when the level is "custom", otherwise
	// all the events should be null
	if notifications.Events != nil {
		d.NewNote = types.BoolValue(notifications.Events.NewNote)
		d.NewIssue = types.BoolValue(notifications.Events.NewIssue)
		d.ReopenIssue = types.BoolValue(notifications.Events.ReopenIssue)
		d.CloseIssue = types.BoolValue(notifications.Events.CloseIssue)
		d.ReassignIssue = types.BoolValue(notifications.Events.ReassignIssue)
		d.IssueDue = types.BoolValue(notifications.Events.IssueDue)
		d.NewMergeRequest = types.BoolValue(notifications.Events.NewMergeRequest)
		d.PushToMergeRequest = types.BoolValue(notifications.Events.PushToMergeRequest)
		d.ReopenMergeRequest = types.BoolValue(notifications.Events.ReopenMergeRequest)
		d.CloseMergeRequest = types.BoolValue(notifications.Events.CloseMergeRequest)
		d.ReassignMergeRequest = types.BoolValue(notifications.Events.ReassignMergeRequest)
		d.MergeMergeRequest = types.BoolValue(notifications.Events.MergeMergeRequest)
		d.FailedPipeline = types.BoolValue(notifications.Events.FailedPipeline)
		d.FixedPipeline = types.BoolValue(notifications.Events.FixedPipeline)
		d.SuccessPipeline = types.BoolValue(notifications.Events.SuccessPipeline)
		d.MovedProject = types.BoolValue(notifications.Events.MovedProject)
		d.MergeWhenPipelineSucceeds = types.BoolValue(notifications.Events.MergeWhenPipelineSucceeds)
	} else {
		d.NewNote = types.BoolNull()
		d.NewIssue = types.BoolNull()
		d.ReopenIssue = types.BoolNull()
		d.CloseIssue = types.BoolNull()
		d.ReassignIssue = types.BoolNull()
		d.IssueDue = types.BoolNull()
		d.NewMergeRequest = types.BoolNull()
		d.PushToMergeRequest = types.BoolNull()
		d.ReopenMergeRequest = types.BoolNull()
		d.CloseMergeRequest = types.BoolNull()
		d.ReassignMergeRequest = types.BoolNull()
		d.MergeMergeRequest = types.BoolNull()
		d.FailedPipeline = types.BoolNull()
		d.FixedPipeline = types.BoolNull()
		d.SuccessPipeline = types.BoolNull()
		d.MovedProject = types.BoolNull()
		d.MergeWhenPipelineSucceeds = types.BoolNull()
	}
}

// Both update and create essentially do the same thing; there is no resource to create in GitLab,
// we just update the global notification settings either way.
func (d *gitlabGlobalLevelNotificationsResource) updateGlobalNotifications(ctx context.Context, data *gitlabGlobalLevelNotificationsModel) (*gitlab.NotificationSettings, error) {

	opts := &gitlab.NotificationSettingsOptions{}
	if !data.Level.IsNull() {
		val := globalNotificationLevelTypes[data.Level.ValueString()]
		opts.Level = &val
	}
	if !data.NewNote.IsNull() {
		opts.NewNote = data.NewNote.ValueBoolPointer()
	}
	if !data.NewIssue.IsNull() {
		opts.NewIssue = data.NewIssue.ValueBoolPointer()
	}
	if !data.ReopenIssue.IsNull() {
		opts.ReopenIssue = data.ReopenIssue.ValueBoolPointer()
	}
	if !data.CloseIssue.IsNull() {
		opts.CloseIssue = data.CloseIssue.ValueBoolPointer()
	}
	if !data.ReassignIssue.IsNull() {
		opts.ReassignIssue = data.ReassignIssue.ValueBoolPointer()
	}
	if !data.IssueDue.IsNull() {
		opts.IssueDue = data.IssueDue.ValueBoolPointer()
	}
	if !data.NewMergeRequest.IsNull() {
		opts.NewMergeRequest = data.NewMergeRequest.ValueBoolPointer()
	}
	if !data.PushToMergeRequest.IsNull() {
		opts.PushToMergeRequest = data.PushToMergeRequest.ValueBoolPointer()
	}
	if !data.ReopenMergeRequest.IsNull() {
		opts.ReopenMergeRequest = data.ReopenMergeRequest.ValueBoolPointer()
	}
	if !data.CloseMergeRequest.IsNull() {
		opts.CloseMergeRequest = data.CloseMergeRequest.ValueBoolPointer()
	}
	if !data.ReassignMergeRequest.IsNull() {
		opts.ReassignMergeRequest = data.ReassignMergeRequest.ValueBoolPointer()
	}
	if !data.MergeMergeRequest.IsNull() {
		opts.MergeMergeRequest = data.MergeMergeRequest.ValueBoolPointer()
	}
	if !data.FailedPipeline.IsNull() {
		opts.FailedPipeline = data.FailedPipeline.ValueBoolPointer()
	}
	if !data.FixedPipeline.IsNull() {
		opts.FixedPipeline = data.FixedPipeline.ValueBoolPointer()
	}
	if !data.SuccessPipeline.IsNull() {
		opts.SuccessPipeline = data.SuccessPipeline.ValueBoolPointer()
	}
	if !data.MovedProject.IsNull() {
		opts.MovedProject = data.MovedProject.ValueBoolPointer()
	}
	if !data.MergeWhenPipelineSucceeds.IsNull() {
		opts.MergeWhenPipelineSucceeds = data.MergeWhenPipelineSucceeds.ValueBoolPointer()
	}

	settings, _, err := d.client.NotificationSettings.UpdateGlobalSettings(opts, gitlab.WithContext(ctx))
	if err != nil {
		tflog.Error(ctx, "Error setting global notifications", map[string]any{
			"data":  data,
			"error": err,
		})
		return nil, err
	}

	return settings, nil
}

var globalNotificationLevelTypes = map[string]gitlab.NotificationLevelValue{
	"disabled":      DisabledNotificationLevel,
	"participating": ParticipatingNotificationLevel,
	"watch":         WatchNotificationLevel,
	"mention":       MentionNotificationLevel,
	"custom":        CustomNotificationLevel,
}
