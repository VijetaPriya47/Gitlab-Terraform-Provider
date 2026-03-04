package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ resource.Resource                = &gitlabRunnerControllerInstanceScopeResource{}
	_ resource.ResourceWithConfigure   = &gitlabRunnerControllerInstanceScopeResource{}
	_ resource.ResourceWithImportState = &gitlabRunnerControllerInstanceScopeResource{}
)

func init() {
	registerResource(NewGitlabRunnerControllerInstanceScopeResource)
}

func NewGitlabRunnerControllerInstanceScopeResource() resource.Resource {
	return &gitlabRunnerControllerInstanceScopeResource{}
}

type gitlabRunnerControllerInstanceScopeResource struct {
	client *gitlab.Client
}

type gitlabRunnerControllerInstanceScopeResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	RunnerControllerID types.Int64  `tfsdk:"runner_controller_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *gitlabRunnerControllerInstanceScopeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_controller_instance_scope"
}

func (r *gitlabRunnerControllerInstanceScopeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_runner_controller_instance_scope`" + ` resource manages the instance-level scope of a runner controller.

~> This resource is **experimental** and may change or be removed in future versions. Introduced in GitLab 18.10.

-> This resource requires administration privileges.

~> Instance scopes and runner scopes are mutually exclusive on a runner controller.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/runner_controllers/#runner-controller-scopes)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<runner_controller_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"runner_controller_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the runner controller.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time the scope was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The time the scope was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabRunnerControllerInstanceScopeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabRunnerControllerInstanceScopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabRunnerControllerInstanceScopeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID := data.RunnerControllerID.ValueInt64()

	tflog.Debug(ctx, "Adding instance scope to runner controller", map[string]any{
		"controller_id": controllerID,
	})

	scope, _, err := r.client.RunnerControllerScopes.AddRunnerControllerInstanceScope(controllerID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to add instance scope to runner controller: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(controllerID, 10))
	if scope.CreatedAt != nil {
		data.CreatedAt = types.StringValue(scope.CreatedAt.Format(time.RFC3339))
	}
	if scope.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(scope.UpdatedAt.Format(time.RFC3339))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerInstanceScopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabRunnerControllerInstanceScopeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", fmt.Sprintf("Could not parse runner controller ID %q: %s", data.ID.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "Reading instance scope for runner controller", map[string]any{
		"controller_id": controllerID,
	})

	scopes, _, err := r.client.RunnerControllerScopes.ListRunnerControllerScopes(controllerID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Runner controller not found, removing instance scope from state", map[string]any{
				"controller_id": controllerID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read runner controller scopes: %s", err.Error()))
		return
	}

	if len(scopes.InstanceLevelScopings) == 0 {
		tflog.Debug(ctx, "No instance scope found for runner controller, removing from state", map[string]any{
			"controller_id": controllerID,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.RunnerControllerID = types.Int64Value(controllerID)
	scope := scopes.InstanceLevelScopings[0]
	if scope.CreatedAt != nil {
		data.CreatedAt = types.StringValue(scope.CreatedAt.Format(time.RFC3339))
	}
	if scope.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(scope.UpdatedAt.Format(time.RFC3339))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerInstanceScopeResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

func (r *gitlabRunnerControllerInstanceScopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabRunnerControllerInstanceScopeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", fmt.Sprintf("Could not parse runner controller ID %q: %s", data.ID.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "Removing instance scope from runner controller", map[string]any{
		"controller_id": controllerID,
	})

	_, err = r.client.RunnerControllerScopes.RemoveRunnerControllerInstanceScope(controllerID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Instance scope already removed from runner controller", map[string]any{
				"controller_id": controllerID,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to remove instance scope from runner controller: %s", err.Error()))
		return
	}
}

func (r *gitlabRunnerControllerInstanceScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
