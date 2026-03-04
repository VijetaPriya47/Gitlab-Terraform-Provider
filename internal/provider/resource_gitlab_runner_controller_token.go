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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabRunnerControllerTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabRunnerControllerTokenResource{}
	_ resource.ResourceWithImportState = &gitlabRunnerControllerTokenResource{}
)

func init() {
	registerResource(NewGitlabRunnerControllerTokenResource)
}

func NewGitlabRunnerControllerTokenResource() resource.Resource {
	return &gitlabRunnerControllerTokenResource{}
}

type gitlabRunnerControllerTokenResource struct {
	client *gitlab.Client
}

type gitlabRunnerControllerTokenResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	RunnerControllerID types.Int64  `tfsdk:"runner_controller_id"`
	Description        types.String `tfsdk:"description"`
	Token              types.String `tfsdk:"token"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *gitlabRunnerControllerTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_controller_token"
}

func (r *gitlabRunnerControllerTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_runner_controller_token`" + ` resource manages the lifecycle of a runner controller token.

~> This resource is **experimental** and may change or be removed in future versions. Introduced in GitLab 18.9.

-> This resource requires administration privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/runner_controller_tokens/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<runner_controller_id>:<token_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"runner_controller_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the runner controller.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the runner controller token.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token value. **Note**: the token is not available for imported resources.",
				Computed:            true,
				Sensitive:           true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time the token was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The time the token was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabRunnerControllerTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabRunnerControllerTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabRunnerControllerTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID := data.RunnerControllerID.ValueInt64()
	options := &gitlab.CreateRunnerControllerTokenOptions{}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}

	tflog.Debug(ctx, "Creating runner controller token", map[string]any{
		"controller_id": controllerID,
	})

	token, _, err := r.client.RunnerControllerTokens.CreateRunnerControllerToken(controllerID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create runner controller token: %s", err.Error()))
		return
	}

	controllerIDStr := strconv.FormatInt(controllerID, 10)
	tokenIDStr := strconv.FormatInt(token.ID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&controllerIDStr, &tokenIDStr))
	data.runnerControllerTokenToStateModel(token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabRunnerControllerTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID, tokenID, err := resourceGitlabRunnerControllerTokenParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	tflog.Debug(ctx, "Reading runner controller token", map[string]any{
		"controller_id": controllerID,
		"token_id":      tokenID,
	})

	token, _, err := r.client.RunnerControllerTokens.GetRunnerControllerToken(controllerID, tokenID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Runner controller token not found, removing from state", map[string]any{
				"controller_id": controllerID,
				"token_id":      tokenID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read runner controller token: %s", err.Error()))
		return
	}

	data.RunnerControllerID = types.Int64Value(token.RunnerControllerID)
	data.runnerControllerTokenToStateModel(token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabRunnerControllerTokenResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

func (r *gitlabRunnerControllerTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabRunnerControllerTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID, tokenID, err := resourceGitlabRunnerControllerTokenParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	tflog.Debug(ctx, "Revoking runner controller token", map[string]any{
		"controller_id": controllerID,
		"token_id":      tokenID,
	})

	_, err = r.client.RunnerControllerTokens.RevokeRunnerControllerToken(controllerID, tokenID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Runner controller token already revoked", map[string]any{
				"controller_id": controllerID,
				"token_id":      tokenID,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to revoke runner controller token: %s", err.Error()))
		return
	}
}

func (r *gitlabRunnerControllerTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabRunnerControllerTokenResourceModel) runnerControllerTokenToStateModel(token *gitlab.RunnerControllerToken) {
	d.Description = types.StringValue(token.Description)
	if token.Token != "" {
		d.Token = types.StringValue(token.Token)
	}
	if token.CreatedAt != nil {
		d.CreatedAt = types.StringValue(token.CreatedAt.Format(time.RFC3339))
	}
	if token.UpdatedAt != nil {
		d.UpdatedAt = types.StringValue(token.UpdatedAt.Format(time.RFC3339))
	}
}

func resourceGitlabRunnerControllerTokenParseID(id string) (int64, int64, error) {
	controllerIDStr, tokenIDStr, err := utils.ParseTwoPartID(id)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse runner controller token ID %q: %w", id, err)
	}

	controllerID, err := strconv.ParseInt(controllerIDStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse runner controller ID %q: %w", controllerIDStr, err)
	}

	tokenID, err := strconv.ParseInt(tokenIDStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse token ID %q: %w", tokenIDStr, err)
	}

	return controllerID, tokenID, nil
}
