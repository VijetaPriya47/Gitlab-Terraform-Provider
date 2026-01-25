package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &gitlabPipelineScheduleResource{}
	_ resource.ResourceWithConfigure    = &gitlabPipelineScheduleResource{}
	_ resource.ResourceWithImportState  = &gitlabPipelineScheduleResource{}
	_ resource.ResourceWithModifyPlan   = &gitlabPipelineScheduleResource{}
	_ resource.ResourceWithUpgradeState = &gitlabPipelineScheduleResource{}
)

func init() {
	registerResource(NewGitLabPipelineScheduleResource)
}

func NewGitLabPipelineScheduleResource() resource.Resource {
	return &gitlabPipelineScheduleResource{}
}

type gitlabPipelineScheduleResource struct {
	client *gitlab.Client
}

type gitlabPipelineScheduleResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	PipelineScheduleID types.Int64  `tfsdk:"pipeline_schedule_id"`
	Project            types.String `tfsdk:"project"`
	Description        types.String `tfsdk:"description"`
	Ref                types.String `tfsdk:"ref"`
	Cron               types.String `tfsdk:"cron"`
	CronTimezone       types.String `tfsdk:"cron_timezone"`
	Active             types.Bool   `tfsdk:"active"`
	TakeOwnership      types.Bool   `tfsdk:"take_ownership"`
	Owner              types.Int64  `tfsdk:"owner"`
}

type gitlabPipelineScheduleResourceModelSchema0 = gitlabPipelineScheduleResourceModel

// Metadata returns the resource name
func (d *gitlabPipelineScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_schedule"
}

func (d *gitlabPipelineScheduleResource) getV1Schema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_pipeline_schedule` " + `resource manages the lifecycle of a scheduled pipeline.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/pipeline_schedules/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id>:<pipeline-schedule-id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pipeline_schedule_id": schema.Int64Attribute{
				MarkdownDescription: "The pipeline schedule id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the schedule to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the pipeline schedule.",
				Required:            true,
			},
			"ref": schema.StringAttribute{
				MarkdownDescription: "The branch/tag name to be triggered. This must be the full branch reference, for example: `refs/heads/main`, not `main`.",
				Required:            true,
			},
			"cron": schema.StringAttribute{
				MarkdownDescription: "The cron (e.g. `0 1 * * *`).",
				Required:            true,
			},
			"cron_timezone": schema.StringAttribute{
				MarkdownDescription: "The timezone.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "The activation of pipeline schedule. If false is set, the pipeline schedule will deactivated initially.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"take_ownership": schema.BoolAttribute{
				MarkdownDescription: "When set to `true`, the user represented by the token running Terraform will take ownership of the scheduled pipeline prior to editing it. This can help when managing scheduled pipeline drift when other users are making changes outside Terraform.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"owner": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user that owns the pipeline schedule.",
				Computed:            true,
			},
		},
		Version: 1,
	}
}

func (r *gitlabPipelineScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getV1Schema()
}

// Configure adds the provider configured client to the resource.
func (r *gitlabPipelineScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabPipelineScheduleResource) pipelineScheduleToStateModel(pid string, pipelineSchedule *gitlab.PipelineSchedule, data *gitlabPipelineScheduleResourceModel) {
	data.Project = types.StringValue(pid)
	data.PipelineScheduleID = types.Int64Value(int64(pipelineSchedule.ID))
	data.Description = types.StringValue(pipelineSchedule.Description)
	data.Ref = types.StringValue(pipelineSchedule.Ref)
	data.Cron = types.StringValue(pipelineSchedule.Cron)
	data.CronTimezone = types.StringValue(pipelineSchedule.CronTimezone)
	data.Active = types.BoolValue(pipelineSchedule.Active)
	data.TakeOwnership = types.BoolValue(data.TakeOwnership.ValueBool())

	ownerId := int64(0)
	if pipelineSchedule.Owner != nil {
		ownerId = pipelineSchedule.Owner.ID
	}
	data.Owner = types.Int64Value(ownerId)
}

// Note: In the framework, every state upgrade function must perform all steps necessary to upgrade the state
// in a single step. That means if we ever add another v2 schema, the v0 upgrader will also need to be updated.
func (d *gitlabPipelineScheduleResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	schema := d.getV0Schema()

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data gitlabPipelineScheduleResourceModelSchema0
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				newData, err := resourceGitlabPipelineScheduleStateUpgradeV0ToV1(ctx, &data)
				if err != nil {
					resp.Diagnostics.AddError("Unable to upgrade state", err.Error())
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)
			},
		},
	}
}

// From V0-V1 the `id` attribute value format changed from `<pipeline-schedule-id>` to `<project-id>:<pipeline-schedule-id>:<key>`,
// which means that the actual schema definition was not impacted and we can just return the
// V1 schema as V0 schema.
func (d *gitlabPipelineScheduleResource) getV0Schema() schema.Schema {
	schema := d.getV1Schema()
	schema.Version = 0
	return schema
}

func resourceGitlabPipelineScheduleStateUpgradeV0ToV1(ctx context.Context, data *gitlabPipelineScheduleResourceModelSchema0) (*gitlabPipelineScheduleResourceModel, error) {
	project := data.Project.ValueString()
	oldID := data.ID.ValueString()

	_, err := strconv.Atoi(oldID)
	if err != nil {
		return nil, fmt.Errorf("unable to convert pipeline schedule id %q to integer to migrate to new schema: %s", oldID, err.Error())
	}

	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]any{"project": project, "v0-id": oldID})
	newID := utils.BuildTwoPartID(&project, &oldID)
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]any{"v0-id": oldID, "v1-id": newID})

	newData := &gitlabPipelineScheduleResourceModel{
		ID:                 types.StringValue(newID),
		PipelineScheduleID: data.PipelineScheduleID,
		Project:            data.Project,
		Description:        data.Description,
		Ref:                data.Ref,
		Cron:               data.Cron,
		CronTimezone:       data.CronTimezone,
		Active:             data.Active,
		TakeOwnership:      data.TakeOwnership,
		Owner:              data.Owner,
	}
	return newData, nil
}

func (r *gitlabPipelineScheduleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var rawTakeOwnership types.Bool
	diags := req.Plan.GetAttribute(ctx, path.Root("take_ownership"), &rawTakeOwnership)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the current user is different than the owner of the pipeline,
	// modify the plan to update ownership, which will force an apply to
	// take ownership of the pipeline.
	takeOwnership := rawTakeOwnership.ValueBool()
	if takeOwnership {
		var rawOwnerID types.Int64
		diags := req.Plan.GetAttribute(ctx, path.Root("owner"), &rawOwnerID)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		ownerID := rawOwnerID.ValueInt64()

		currentUser, _, err := r.client.Users.CurrentUser(gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"GitLab API error occured",
				fmt.Sprintf("unable to get current user: %s", err.Error()),
			)
			return
		}

		if ownerID != int64(currentUser.ID) {
			rawNewOwnerID := types.Int64Value(int64(currentUser.ID))
			diags = resp.Plan.SetAttribute(ctx, path.Root("owner"), rawNewOwnerID)
			resp.Diagnostics.Append(diags...)
		}
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabPipelineScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabPipelineScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabPipelineScheduleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	projectID, rawPipelineScheduleID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<project-id>:<pipeline-schedule-id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	pipelineScheduleID, err := strconv.ParseInt(rawPipelineScheduleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid pipeline schedule ID provided, pipeline schedule ID should be an Int",
			fmt.Sprintf("Unable to convert pipeline schedule id to int: %s", err.Error()),
		)
		return
	}

	// Read pipeline schedule
	pipelineSchedule, _, err := r.client.PipelineSchedules.GetPipelineSchedule(projectID, pipelineScheduleID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "pipeline schedule does not exist, removing from state", map[string]any{
				"project": projectID, "pipelineSchedule": pipelineScheduleID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read gitlab pipeline schedule details: %s", err.Error()))
		return
	}

	// persist API response in state model
	rawPipelineScheduleID = strconv.FormatInt(pipelineSchedule.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&projectID, &rawPipelineScheduleID))
	r.pipelineScheduleToStateModel(projectID, pipelineSchedule, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPipelineScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabPipelineScheduleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, rawPipelineScheduleID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Update. It should be '<project-id>:<pipeline-schedule-id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	pipelineScheduleID, err := strconv.ParseInt(rawPipelineScheduleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid pipeline schedule ID provided, pipeline schedule ID should be an Int",
			fmt.Sprintf("Unable to convert pipeline schedule id to int: %s", err.Error()),
		)
		return
	}

	optionsEdit := &gitlab.EditPipelineScheduleOptions{}

	optionsEdit.Description = gitlab.Ptr(data.Description.ValueString())
	optionsEdit.Ref = gitlab.Ptr(data.Ref.ValueString())
	optionsEdit.Cron = gitlab.Ptr(data.Cron.ValueString())
	optionsEdit.CronTimezone = gitlab.Ptr(data.CronTimezone.ValueString())
	if !data.Active.IsNull() && !data.Active.IsUnknown() {
		optionsEdit.Active = gitlab.Ptr(data.Active.ValueBool())
	}

	if data.TakeOwnership.ValueBool() {
		tflog.Debug(ctx, "[DEBUG] Taking ownership of gitlab PipelineSchedule.", map[string]any{"pipelineSchedule": data.ID})

		_, _, err := r.client.PipelineSchedules.TakeOwnershipOfPipelineSchedule(projectID, pipelineScheduleID, nil, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"GitLab API error occurred",
				fmt.Sprintf("Unable to take ownership on pipeline schedule: %s", err.Error()),
			)
			return
		}
	}

	pipelineSchedule, _, err := r.client.PipelineSchedules.EditPipelineSchedule(projectID, pipelineScheduleID, optionsEdit, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update pipeline schedule: %s", err.Error()))
		return
	}

	// persist API response in state model
	rawPipelineScheduleID = strconv.FormatInt(pipelineSchedule.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&projectID, &rawPipelineScheduleID))
	r.pipelineScheduleToStateModel(projectID, pipelineSchedule, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "updated a pipeline schedule", map[string]any{
		"project": projectID, "pipelineSchedule": pipelineScheduleID,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPipelineScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabPipelineScheduleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, rawPipelineScheduleID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Delete. It should be '<project-id>:<pipeline-schedule-id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	pipelineScheduleID, err := strconv.ParseInt(rawPipelineScheduleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid pipeline schedule ID provided, pipeline schedule ID should be an Int",
			fmt.Sprintf("Unable to convert pipeline schedule id to int: %s", err.Error()),
		)
		return
	}

	if _, err = r.client.PipelineSchedules.DeletePipelineSchedule(projectID, pipelineScheduleID, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete pipeline schedule: %s", err.Error()),
		)
	}
}

func (r *gitlabPipelineScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabPipelineScheduleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	// read all information for refresh from resource id
	projectID := data.Project.ValueString()
	pipelineScheduleDescription := data.Description.ValueString()
	pipelineScheduleRef := data.Ref.ValueString()
	pipelineScheduleCron := data.Cron.ValueString()
	pipelineScheduleCronTimezone := data.CronTimezone.ValueString()
	pipelineScheduleActive := data.Active.ValueBool()

	optionsCreate := &gitlab.CreatePipelineScheduleOptions{
		Description:  gitlab.Ptr(pipelineScheduleDescription),
		Ref:          gitlab.Ptr(pipelineScheduleRef),
		Cron:         gitlab.Ptr(pipelineScheduleCron),
		CronTimezone: gitlab.Ptr(pipelineScheduleCronTimezone),
		Active:       gitlab.Ptr(pipelineScheduleActive),
	}

	pipelineSchedule, _, err := r.client.PipelineSchedules.CreatePipelineSchedule(projectID, optionsCreate, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create pipeline schedule: %s", err.Error()))
		return
	}

	// persist API response in state model
	rawPipelineScheduleID := strconv.FormatInt(pipelineSchedule.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&projectID, &rawPipelineScheduleID))
	r.pipelineScheduleToStateModel(projectID, pipelineSchedule, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "created a pipeline schedule", map[string]any{
		"project": projectID, "pipelineSchedule": rawPipelineScheduleID,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
