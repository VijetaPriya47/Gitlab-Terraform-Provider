package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabClusterAgentTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabClusterAgentTokenResource{}
	_ resource.ResourceWithImportState = &gitlabClusterAgentTokenResource{}
)

func init() {
	registerResource(NewGitlabClusterAgentTokenResource)
}

func NewGitlabClusterAgentTokenResource() resource.Resource {
	return &gitlabClusterAgentTokenResource{}
}

type gitlabClusterAgentTokenResource struct {
	client *gitlab.Client
}

func (r *gitlabClusterAgentTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_agent_token"
}

type gitlabClusterAgentTokenResourceModel struct {
	ID              types.String      `tfsdk:"id"`
	Project         types.String      `tfsdk:"project"`
	AgentID         types.Int64       `tfsdk:"agent_id"`
	TokenID         types.Int64       `tfsdk:"token_id"`
	Name            types.String      `tfsdk:"name"`
	Description     types.String      `tfsdk:"description"`
	Status          types.String      `tfsdk:"status"`
	CreatedAt       timetypes.RFC3339 `tfsdk:"created_at"`
	CreatedByUserID types.Int64       `tfsdk:"created_by_user_id"`
	LastUsedAt      timetypes.RFC3339 `tfsdk:"last_used_at"`
	Token           types.String      `tfsdk:"token"`
}

func (r *gitlabClusterAgentTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	tokenStatuses := []string{"active", "revoked"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_cluster_agent_token` + "`" + ` resource manages the lifecycle of a token for a GitLab Agent for Kubernetes.

-> Requires at least maintainer permissions on the project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/cluster_agents/#create-an-agent-token)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource. In the format <project-id:agent-id:token_id>",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or full path of the project maintained by the authenticated user.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"agent_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the agent.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"token_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the token.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The Name of the agent.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The Description for the agent.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The status of the token. Valid values are %s.", utils.RenderValueListForDocs(tokenStatuses)),
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(tokenStatuses...)},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 datetime when the agent was created.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"created_by_user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user who created the agent.",
				Computed:            true,
			},
			"last_used_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 datetime when the token was last used.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The secret token for the agent. The `token` is not available in imported resources.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *gitlabClusterAgentTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabClusterAgentTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabClusterAgentTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabClusterAgentTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	agentID := data.AgentID.ValueInt64()
	options := &gitlab.CreateAgentTokenOptions{
		Name: data.Name.ValueStringPointer(),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}

	tflog.Debug(ctx, fmt.Sprintf("create token for GitLab Agent for Kubernetes %d in project %s with name '%v'", agentID, project, options.Name))
	clusterAgentToken, _, err := r.client.ClusterAgents.CreateAgentToken(project, agentID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create token: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d:%d", project, agentID, clusterAgentToken.ID))
	data.Token = types.StringValue(clusterAgentToken.Token)
	data.modelToStateModel(project, clusterAgentToken)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabClusterAgentTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabClusterAgentTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, agentID, tokenID, err := resourceGitlabClusterAgentTokenParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Read. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read token for GitLab Agent for Kubernetes %d in project %s with id %d", agentID, project, tokenID))
	clusterAgentToken, _, err := r.client.ClusterAgents.GetAgentToken(project, agentID, tokenID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("read token for GitLab Agent for Kubernetes %d in project %s with id %d not found, removing from state", agentID, project, tokenID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read token: %s", err.Error()))
		return
	}

	data.modelToStateModel(project, clusterAgentToken)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabClusterAgentTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Provider Error, report upstream",
		"Somehow the resource was requested to perform an in-place upgrade which is not possible.",
	)
}

func (r *gitlabClusterAgentTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabClusterAgentTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, agentID, tokenID, err := resourceGitlabClusterAgentTokenParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Delete. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("delete token for GitLab Agent for Kubernetes %d in project %s with id %d", agentID, project, tokenID))
	if _, err := r.client.ClusterAgents.RevokeAgentToken(project, agentID, tokenID, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to revoke token: %s", err.Error()))
		return
	}
}

func (d *gitlabClusterAgentTokenResourceModel) modelToStateModel(project string, clusterAgentToken *gitlab.AgentToken) *diag.Diagnostics {
	d.Project = types.StringValue(project)
	d.AgentID = types.Int64Value(int64(clusterAgentToken.AgentID))
	d.TokenID = types.Int64Value(int64(clusterAgentToken.ID))
	d.Name = types.StringValue(clusterAgentToken.Name)
	d.Description = types.StringValue(clusterAgentToken.Description)
	d.Status = types.StringValue(clusterAgentToken.Status)
	createdAt, diags := timetypes.NewRFC3339Value(clusterAgentToken.CreatedAt.Format(time.RFC3339))
	if diags != nil && diags.HasError() {
		return &diags
	}
	d.CreatedAt = createdAt
	d.CreatedByUserID = types.Int64Value(int64(clusterAgentToken.CreatedByUserID))
	if clusterAgentToken.LastUsedAt == nil {
		d.LastUsedAt = timetypes.NewRFC3339Null()
	} else {
		lastUsedAt, diags := timetypes.NewRFC3339Value(clusterAgentToken.LastUsedAt.Format(time.RFC3339))
		if diags != nil && diags.HasError() {
			return &diags
		}
		d.LastUsedAt = lastUsedAt
	}

	return nil
}

func resourceGitlabClusterAgentTokenParseID(id string) (string, int64, int64, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 3 {
		return "", 0, 0, fmt.Errorf("invalid cluster agent token id %q, expected format '{project}:{agent_id}:{token_id}", id)
	}
	project, rawAgentID, rawTokenID := parts[0], parts[1], parts[2]
	agentID, err := strconv.ParseInt(rawAgentID, 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid cluster agent token id %q with 'agent_id' %q, expected integer", id, rawAgentID)
	}
	tokenID, err := strconv.ParseInt(rawTokenID, 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid cluster agent token id %q with 'token_id' %q, expected integer", id, rawTokenID)
	}

	return project, agentID, tokenID, nil
}
