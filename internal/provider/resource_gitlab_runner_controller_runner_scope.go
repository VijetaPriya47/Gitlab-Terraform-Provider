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
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabRunnerControllerRunnerScopeResource{}
	_ resource.ResourceWithConfigure   = &gitlabRunnerControllerRunnerScopeResource{}
	_ resource.ResourceWithImportState = &gitlabRunnerControllerRunnerScopeResource{}
)

func init() {
	registerResource(NewGitlabRunnerControllerRunnerScopeResource)
}

func NewGitlabRunnerControllerRunnerScopeResource() resource.Resource {
	return &gitlabRunnerControllerRunnerScopeResource{}
}

type gitlabRunnerControllerRunnerScopeResource struct {
	client *gitlab.Client
}

type gitlabRunnerControllerRunnerScopeResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	RunnerControllerID types.Int64  `tfsdk:"runner_controller_id"`
	RunnerID           types.Int64  `tfsdk:"runner_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *gitlabRunnerControllerRunnerScopeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_controller_runner_scope"
}

func (r *gitlabRunnerControllerRunnerScopeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_runner_controller_runner_scope`" + ` resource manages a runner-level scope of a runner controller. The runner must be an instance-level runner.

~> This resource is **experimental** and may change or be removed in future versions. Introduced in GitLab 18.10.

-> This resource requires administration privileges.

~> Instance scopes and runner scopes are mutually exclusive on a runner controller.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/runner_controllers/#runner-controller-scopes)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<runner_controller_id>:<runner_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"runner_controller_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the runner controller.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"runner_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the instance-level runner to scope the controller to.",
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

func (r *gitlabRunnerControllerRunnerScopeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabRunnerControllerRunnerScopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabRunnerControllerRunnerScopeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID := data.RunnerControllerID.ValueInt64()
	runnerID := data.RunnerID.ValueInt64()

	tflog.Debug(ctx, "Adding runner scope to runner controller", map[string]any{
		"controller_id": controllerID,
		"runner_id":     runnerID,
	})

	scope, _, err := r.client.RunnerControllerScopes.AddRunnerControllerRunnerScope(controllerID, runnerID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to add runner scope to runner controller: %s", err.Error()))
		return
	}

	controllerIDStr := strconv.FormatInt(controllerID, 10)
	runnerIDStr := strconv.FormatInt(runnerID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&controllerIDStr, &runnerIDStr))
	if scope.CreatedAt != nil {
		data.CreatedAt = types.StringValue(scope.CreatedAt.Format(time.RFC3339))
	}
	if scope.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(scope.UpdatedAt.Format(time.RFC3339))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerRunnerScopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabRunnerControllerRunnerScopeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID, runnerID, err := resourceGitlabRunnerControllerRunnerScopeParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	tflog.Debug(ctx, "Reading runner scope for runner controller", map[string]any{
		"controller_id": controllerID,
		"runner_id":     runnerID,
	})

	scopes, _, err := r.client.RunnerControllerScopes.ListRunnerControllerScopes(controllerID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Runner controller not found, removing runner scope from state", map[string]any{
				"controller_id": controllerID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read runner controller scopes: %s", err.Error()))
		return
	}

	var found *gitlab.RunnerControllerRunnerLevelScoping
	for _, s := range scopes.RunnerLevelScopings {
		if s.RunnerID == runnerID {
			found = s
			break
		}
	}

	if found == nil {
		tflog.Debug(ctx, "Runner scope not found for runner controller, removing from state", map[string]any{
			"controller_id": controllerID,
			"runner_id":     runnerID,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.RunnerControllerID = types.Int64Value(controllerID)
	data.RunnerID = types.Int64Value(runnerID)
	if found.CreatedAt != nil {
		data.CreatedAt = types.StringValue(found.CreatedAt.Format(time.RFC3339))
	}
	if found.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(found.UpdatedAt.Format(time.RFC3339))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerRunnerScopeResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

func (r *gitlabRunnerControllerRunnerScopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabRunnerControllerRunnerScopeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID, runnerID, err := resourceGitlabRunnerControllerRunnerScopeParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	tflog.Debug(ctx, "Removing runner scope from runner controller", map[string]any{
		"controller_id": controllerID,
		"runner_id":     runnerID,
	})

	_, err = r.client.RunnerControllerScopes.RemoveRunnerControllerRunnerScope(controllerID, runnerID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Runner scope already removed from runner controller", map[string]any{
				"controller_id": controllerID,
				"runner_id":     runnerID,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to remove runner scope from runner controller: %s", err.Error()))
		return
	}
}

func (r *gitlabRunnerControllerRunnerScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func resourceGitlabRunnerControllerRunnerScopeParseID(id string) (int64, int64, error) {
	controllerIDStr, runnerIDStr, err := utils.ParseTwoPartID(id)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse runner controller runner scope ID %q: %w", id, err)
	}

	controllerID, err := strconv.ParseInt(controllerIDStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse runner controller ID %q: %w", controllerIDStr, err)
	}

	runnerID, err := strconv.ParseInt(runnerIDStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse runner ID %q: %w", runnerIDStr, err)
	}

	return controllerID, runnerID, nil
}
