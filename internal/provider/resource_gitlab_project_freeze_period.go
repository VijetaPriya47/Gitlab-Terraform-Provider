package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                 = &gitlabProjectFreezePeriodResource{}
	_ resource.ResourceWithConfigure    = &gitlabProjectFreezePeriodResource{}
	_ resource.ResourceWithImportState  = &gitlabProjectFreezePeriodResource{}
	_ resource.ResourceWithUpgradeState = &gitlabProjectFreezePeriodResource{}
)

func init() {
	registerResource(NewGitlabProjectFreezePeriodResource)
}

func NewGitlabProjectFreezePeriodResource() resource.Resource {
	return &gitlabProjectFreezePeriodResource{}
}

type gitlabProjectFreezePeriodResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Project      types.String `tfsdk:"project"`
	FreezeStart  types.String `tfsdk:"freeze_start"`
	FreezeEnd    types.String `tfsdk:"freeze_end"`
	CronTimezone types.String `tfsdk:"cron_timezone"`
}

type gitlabProjectFreezePeriodResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectFreezePeriodResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_freeze_period"
}

func (r *gitlabProjectFreezePeriodResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_project_freeze_period` + "`" + ` resource manages the lifecycle of a freeze period for a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/freeze_periods/)`,
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format <project-id:freeze-period-id>.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or path of the project to add the freeze period to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"freeze_start": schema.StringAttribute{
				MarkdownDescription: "Start of the Freeze Period in cron format (for example, `0 1 * * *`).",
				Required:            true,
			},
			"freeze_end": schema.StringAttribute{
				MarkdownDescription: "End of the Freeze Period in cron format (for example, `0 2 * * *`).",
				Required:            true,
			},
			"cron_timezone": schema.StringAttribute{
				MarkdownDescription: "The timezone.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
			},
		},
	}
}

func (r *gitlabProjectFreezePeriodResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectFreezePeriodResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectFreezePeriodResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectFreezePeriodResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	options := &gitlab.CreateFreezePeriodOptions{
		FreezeStart:  data.FreezeStart.ValueStringPointer(),
		FreezeEnd:    data.FreezeEnd.ValueStringPointer(),
		CronTimezone: data.CronTimezone.ValueStringPointer(),
	}

	tflog.Debug(ctx, fmt.Sprintf("Project %s create gitlab project-level freeze period %+v", project, options))
	freezePeriod, _, err := r.client.FreezePeriods.CreateFreezePeriodOptions(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create freeze period: %s", err.Error()))
		return
	}

	freezePeriodID := strconv.FormatInt(freezePeriod.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &freezePeriodID))
	data.modelToStateModel(project, freezePeriod)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectFreezePeriodResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectFreezePeriodResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, freezePeriodID, err := projectAndFreezePeriodIDFromID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read gitlab FreezePeriod %s/%d", project, freezePeriodID))

	freezePeriod, _, err := r.client.FreezePeriods.GetFreezePeriod(project, freezePeriodID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("project freeze period for %s not found so removing it from state", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read freeze period: %s", err.Error()))
		return
	}

	data.modelToStateModel(project, freezePeriod)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectFreezePeriodResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectFreezePeriodResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, freezePeriodID, err := projectAndFreezePeriodIDFromID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	options := &gitlab.UpdateFreezePeriodOptions{}

	if !data.FreezeStart.IsNull() && !data.FreezeStart.IsUnknown() {
		options.FreezeStart = data.FreezeStart.ValueStringPointer()
	}

	if !data.FreezeEnd.IsNull() && !data.FreezeEnd.IsUnknown() {
		options.FreezeEnd = data.FreezeEnd.ValueStringPointer()
	}

	if !data.CronTimezone.IsNull() && !data.CronTimezone.IsUnknown() {
		options.CronTimezone = data.CronTimezone.ValueStringPointer()
	}

	tflog.Debug(ctx, fmt.Sprintf("update gitlab FreezePeriod %s", data.ID.ValueString()))

	freezePeriod, _, err := r.client.FreezePeriods.UpdateFreezePeriodOptions(project, freezePeriodID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update freeze period: %s", err.Error()))
		return
	}

	data.modelToStateModel(project, freezePeriod)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectFreezePeriodResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectFreezePeriodResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, freezePeriodID, err := projectAndFreezePeriodIDFromID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Delete gitlab FreezePeriod %s", data.ID.ValueString()))
	if _, err := r.client.FreezePeriods.DeleteFreezePeriod(project, freezePeriodID, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete freeze period: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectFreezePeriodResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: r.getV0Schema(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data *gitlabProjectFreezePeriodResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}
				newData := &gitlabProjectFreezePeriodResourceModel{
					ID:           data.ID,
					Project:      data.ProjectID,
					FreezeStart:  data.FreezeStart,
					FreezeEnd:    data.FreezeEnd,
					CronTimezone: data.CronTimezone,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)
			},
		},
	}
}

type gitlabProjectFreezePeriodResourceModelV0 struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	FreezeStart  types.String `tfsdk:"freeze_start"`
	FreezeEnd    types.String `tfsdk:"freeze_end"`
	CronTimezone types.String `tfsdk:"cron_timezone"`
}

func (r *gitlabProjectFreezePeriodResource) getV0Schema() *schema.Schema {
	return &schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_project_freeze_period` + "`" + ` resource manages the lifecycle of a freeze period for a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/freeze_periods/)`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format <project-id:freeze-period-id>.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project to add the freeze period to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"freeze_start": schema.StringAttribute{
				MarkdownDescription: "Start of the Freeze Period in cron format (for example, `0 1 * * *`).",
				Required:            true,
			},
			"freeze_end": schema.StringAttribute{
				MarkdownDescription: "End of the Freeze Period in cron format (for example, `0 2 * * *`).",
				Required:            true,
			},
			"cron_timezone": schema.StringAttribute{
				MarkdownDescription: "The timezone.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
			},
		},
	}
}

func (d *gitlabProjectFreezePeriodResourceModel) modelToStateModel(project string, freezePeriod *gitlab.FreezePeriod) {
	d.Project = types.StringValue(project)
	d.FreezeStart = types.StringValue(freezePeriod.FreezeStart)
	d.FreezeEnd = types.StringValue(freezePeriod.FreezeEnd)
	d.CronTimezone = types.StringValue(freezePeriod.CronTimezone)
}

func projectAndFreezePeriodIDFromID(id string) (string, int64, error) {
	project, freezePeriodIDString, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	freezePeriodID, err := strconv.ParseInt(freezePeriodIDString, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get freezePeriodId: %v", err)
	}

	return project, freezePeriodID, nil
}
