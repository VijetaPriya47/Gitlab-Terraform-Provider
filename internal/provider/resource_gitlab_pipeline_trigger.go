package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                 = &gitlabPipelineTriggerResource{}
	_ resource.ResourceWithConfigure    = &gitlabPipelineTriggerResource{}
	_ resource.ResourceWithImportState  = &gitlabPipelineTriggerResource{}
	_ resource.ResourceWithUpgradeState = &gitlabPipelineTriggerResource{}
)

func init() {
	registerResource(NewGitLabPipelineTriggerResource)
}

func NewGitLabPipelineTriggerResource() resource.Resource {
	return &gitlabPipelineTriggerResource{}
}

type gitlabPipelineTriggerResource struct {
	client *gitlab.Client
}

type gitlabPipelineTriggerResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Project           types.String `tfsdk:"project"`
	Description       types.String `tfsdk:"description"`
	PipelineTriggerId types.Int64  `tfsdk:"pipeline_trigger_id"`
	Token             types.String `tfsdk:"token"`
}

func (r *gitlabPipelineTriggerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_trigger"
}

func (r *gitlabPipelineTriggerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getSchema()
}

func (r *gitlabPipelineTriggerResource) getSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_pipeline_trigger` + "`" + ` resource manages the lifecycle of a pipeline trigger.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/pipeline_triggers/)`,
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<pipeline_trigger_id>`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the trigger to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the pipeline trigger.",
				Required:            true,
			},
			"pipeline_trigger_id": schema.Int64Attribute{
				MarkdownDescription: "The pipeline trigger id.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The pipeline trigger token. This value is not available during import.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *gitlabPipelineTriggerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabPipelineTriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabPipelineTriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	description := data.Description.ValueString()

	tflog.Debug(ctx, "creating pipeline trigger", map[string]any{
		"project":     project,
		"description": description,
	})

	options := &gitlab.AddPipelineTriggerOptions{
		Description: &description,
	}

	pipelineTrigger, _, err := r.client.PipelineTriggers.AddPipelineTrigger(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create pipeline trigger",
			fmt.Sprintf("Could not create pipeline trigger for project %s: %s", project, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(resourceGitlabPipelineTriggerBuildId(project, pipelineTrigger.ID))
	data.PipelineTriggerId = types.Int64Value(int64(pipelineTrigger.ID))

	// While the token does come back on the "GET" api on every request, attempting to retrieve the token using a
	// different user than the one who created the token results in a truncated value coming back from the API,
	// which can corrupt the state. See https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/1410 for
	// more information
	data.Token = types.StringValue(pipelineTrigger.Token)

	tflog.Debug(ctx, "created pipeline trigger", map[string]any{
		"id":                  data.ID.ValueString(),
		"pipeline_trigger_id": pipelineTrigger.ID,
	})

	data.modelToStateModel(project, pipelineTrigger)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPipelineTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabPipelineTriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, pipelineTriggerID, err := resourceGitlabPipelineTriggerParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse pipeline trigger resource ID %s: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "reading pipeline trigger", map[string]any{
		"project":             project,
		"pipeline_trigger_id": pipelineTriggerID,
	})

	pipelineTrigger, _, err := r.client.PipelineTriggers.GetPipelineTrigger(project, pipelineTriggerID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "pipeline trigger not found, removing from state", map[string]any{
				"project":             project,
				"pipeline_trigger_id": pipelineTriggerID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Failed to read pipeline trigger",
			fmt.Sprintf("Could not read pipeline trigger %d for project %s: %s", pipelineTriggerID, project, err.Error()),
		)
		return
	}

	data.modelToStateModel(project, pipelineTrigger)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPipelineTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabPipelineTriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, pipelineTriggerID, err := resourceGitlabPipelineTriggerParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse pipeline trigger resource ID %s: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	description := data.Description.ValueString()

	tflog.Debug(ctx, "updating pipeline trigger", map[string]any{
		"project":             project,
		"pipeline_trigger_id": pipelineTriggerID,
		"description":         description,
	})

	options := &gitlab.EditPipelineTriggerOptions{
		Description: &description,
	}

	pipelineTrigger, _, err := r.client.PipelineTriggers.EditPipelineTrigger(project, pipelineTriggerID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update pipeline trigger",
			fmt.Sprintf("Could not update pipeline trigger %d for project %s: %s", pipelineTriggerID, project, err.Error()),
		)
		return
	}

	data.modelToStateModel(project, pipelineTrigger)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPipelineTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabPipelineTriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, pipelineTriggerID, err := resourceGitlabPipelineTriggerParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse pipeline trigger resource ID %s: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleting pipeline trigger", map[string]any{
		"project":             project,
		"pipeline_trigger_id": pipelineTriggerID,
	})

	_, err = r.client.PipelineTriggers.DeletePipelineTrigger(project, pipelineTriggerID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete pipeline trigger",
			fmt.Sprintf("Could not delete pipeline trigger %d for project %s: %s", pipelineTriggerID, project, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted pipeline trigger", map[string]any{
		"project":             project,
		"pipeline_trigger_id": pipelineTriggerID,
	})
}

func (r *gitlabPipelineTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabPipelineTriggerResourceModel) modelToStateModel(project string, pipelineTrigger *gitlab.PipelineTrigger) {
	d.Project = types.StringValue(project)
	d.Description = types.StringValue(pipelineTrigger.Description)
	d.PipelineTriggerId = types.Int64Value(int64(pipelineTrigger.ID))
}

func (r *gitlabPipelineTriggerResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	schema := r.getSchema()

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data *gitlabPipelineTriggerResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				oldID := data.ID.ValueString()
				pipelineTriggerID, err := strconv.ParseInt(oldID, 10, 64)
				if err != nil {
					resp.Diagnostics.AddError(
						"State Migration Failed",
						fmt.Sprintf("Unable to convert pipeline trigger id %q to integer during migration from V0 to V1: %s", oldID, err.Error()),
					)
					return
				}
				project := data.Project.ValueString()

				tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]any{"project": project, "v0-id": oldID})
				data.ID = types.StringValue(resourceGitlabPipelineTriggerBuildId(project, pipelineTriggerID))
				tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]any{"v0-id": oldID, "v1-id": data.ID.ValueString()})
			},
		},
	}
}

func resourceGitlabPipelineTriggerBuildId(project string, pipelineTriggerId int64) string {
	id := fmt.Sprintf("%d", pipelineTriggerId)
	return utils.BuildTwoPartID(&project, &id)
}

func resourceGitlabPipelineTriggerParseID(id string) (string, int64, error) {
	project, rawPipelineTriggerID, err := utils.ParseTwoPartID(id)
	e := fmt.Errorf("unable to parse id %q. Expected format <project>:<pipeline-trigger-id>", id)
	if err != nil {
		return "", 0, e
	}

	pipelineTriggerId, err := strconv.ParseInt(rawPipelineTriggerID, 10, 64)
	if err != nil {
		return "", 0, e
	}

	return project, pipelineTriggerId, nil
}
