package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &gitlabRunnerControllerResource{}
	_ resource.ResourceWithConfigure   = &gitlabRunnerControllerResource{}
	_ resource.ResourceWithImportState = &gitlabRunnerControllerResource{}
)

func init() {
	registerResource(NewGitlabRunnerControllerResource)
}

func NewGitlabRunnerControllerResource() resource.Resource {
	return &gitlabRunnerControllerResource{}
}

type gitlabRunnerControllerResource struct {
	client *gitlab.Client
}

type gitlabRunnerControllerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *gitlabRunnerControllerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_controller"
}

func (r *gitlabRunnerControllerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	validRunnerControllerStates := []string{"disabled", "enabled", "dry_run"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_runner_controller`" + ` resource manages the lifecycle of a runner controller.

~> This resource is **experimental** and may change or be removed in future versions. Introduced in GitLab 18.9.

-> This resource requires administration privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/runner_controllers/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<controller_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the runner controller.",
				Optional:            true,
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The state of the runner controller. Valid values are: %s.", utils.RenderValueListForDocs(validRunnerControllerStates)),
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(validRunnerControllerStates...),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time the runner controller was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The time the runner controller was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabRunnerControllerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabRunnerControllerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabRunnerControllerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.CreateRunnerControllerOptions{}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}
	if !data.State.IsNull() && !data.State.IsUnknown() {
		state := gitlab.RunnerControllerStateValue(data.State.ValueString())
		options.State = &state
	}

	tflog.Debug(ctx, "Creating runner controller")

	controller, _, err := r.client.RunnerControllers.CreateRunnerController(options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create runner controller: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(controller.ID, 10))
	data.runnerControllerToStateModelFromCreate(controller)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabRunnerControllerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", fmt.Sprintf("Could not parse runner controller ID %q: %s", data.ID.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "Reading runner controller", map[string]any{
		"id": id,
	})

	controller, _, err := r.client.RunnerControllers.GetRunnerController(id, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Runner controller not found, removing from state", map[string]any{
				"id": id,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read runner controller: %s", err.Error()))
		return
	}

	data.runnerControllerToStateModel(controller)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabRunnerControllerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", fmt.Sprintf("Could not parse runner controller ID %q: %s", data.ID.ValueString(), err))
		return
	}

	options := &gitlab.UpdateRunnerControllerOptions{}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}
	if !data.State.IsNull() && !data.State.IsUnknown() {
		state := gitlab.RunnerControllerStateValue(data.State.ValueString())
		options.State = &state
	}

	tflog.Debug(ctx, "Updating runner controller", map[string]any{
		"id": id,
	})

	controller, _, err := r.client.RunnerControllers.UpdateRunnerController(id, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update runner controller: %s", err.Error()))
		return
	}

	data.runnerControllerToStateModelFromUpdate(controller)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabRunnerControllerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", fmt.Sprintf("Could not parse runner controller ID %q: %s", data.ID.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "Deleting runner controller", map[string]any{
		"id": id,
	})

	_, err = r.client.RunnerControllers.DeleteRunnerController(id, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Runner controller already deleted", map[string]any{
				"id": id,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete runner controller: %s", err.Error()))
		return
	}
}

func (r *gitlabRunnerControllerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabRunnerControllerResourceModel) runnerControllerToStateModel(controller *gitlab.RunnerControllerDetails) {
	d.Description = types.StringValue(controller.Description)
	d.State = types.StringValue(string(controller.State))
	if controller.CreatedAt != nil {
		d.CreatedAt = types.StringValue(controller.CreatedAt.Format(time.RFC3339))
	}
	if controller.UpdatedAt != nil {
		d.UpdatedAt = types.StringValue(controller.UpdatedAt.Format(time.RFC3339))
	}
}

func (d *gitlabRunnerControllerResourceModel) runnerControllerToStateModelFromCreate(controller *gitlab.RunnerController) {
	d.Description = types.StringValue(controller.Description)
	d.State = types.StringValue(string(controller.State))
	if controller.CreatedAt != nil {
		d.CreatedAt = types.StringValue(controller.CreatedAt.Format(time.RFC3339))
	}
	if controller.UpdatedAt != nil {
		d.UpdatedAt = types.StringValue(controller.UpdatedAt.Format(time.RFC3339))
	}
}

func (d *gitlabRunnerControllerResourceModel) runnerControllerToStateModelFromUpdate(controller *gitlab.RunnerController) {
	d.Description = types.StringValue(controller.Description)
	d.State = types.StringValue(string(controller.State))
	if controller.CreatedAt != nil {
		d.CreatedAt = types.StringValue(controller.CreatedAt.Format(time.RFC3339))
	}
	if controller.UpdatedAt != nil {
		d.UpdatedAt = types.StringValue(controller.UpdatedAt.Format(time.RFC3339))
	}
}
