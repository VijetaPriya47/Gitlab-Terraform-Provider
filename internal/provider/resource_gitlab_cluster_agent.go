package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ resource.Resource                = &gitlabClusterAgentResource{}
	_ resource.ResourceWithConfigure   = &gitlabClusterAgentResource{}
	_ resource.ResourceWithImportState = &gitlabClusterAgentResource{}
)

func init() {
	registerResource(NewGitlabClusterAgentResource)
}

func NewGitlabClusterAgentResource() resource.Resource {
	return &gitlabClusterAgentResource{}
}

type gitlabClusterAgentResource struct {
	client *gitlab.Client
}

func (r *gitlabClusterAgentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_agent"
}

type gitlabClusterAgentResourceModel struct {
	ID              types.String      `tfsdk:"id"`
	Project         types.String      `tfsdk:"project"`
	Name            types.String      `tfsdk:"name"`
	AgentID         types.Int64       `tfsdk:"agent_id"`
	CreatedAt       timetypes.RFC3339 `tfsdk:"created_at"`
	CreatedByUserID types.Int64       `tfsdk:"created_by_user_id"`
}

func (r *gitlabClusterAgentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_cluster_agent` + "`" + ` resource allows to manage the lifecycle of a GitLab Agent for Kubernetes.

-> Note that this resource only registers the agent, but doesn't configure it.
   The configuration needs to be manually added as described in
   [the docs](https://docs.gitlab.com/user/clusters/agent/install/index/#create-an-agent-configuration-file).
   However, a ` + "`gitlab_repository_file`" + ` resource may be used to achieve that.

-> Requires at least maintainer permissions on the project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/cluster_agents/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource. In the format <project:agent_id>",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or full path of the project maintained by the authenticated user.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The Name of the agent.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"agent_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the agent.",
				Computed:            true,
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
		},
	}
}

func (r *gitlabClusterAgentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabClusterAgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabClusterAgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabClusterAgentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	options := &gitlab.RegisterAgentOptions{
		Name: data.Name.ValueStringPointer(),
	}

	tflog.Debug(ctx, fmt.Sprintf("create GitLab Agent for Kubernetes in project %s with name '%v'", project, options.Name))
	clusterAgent, _, err := r.client.ClusterAgents.RegisterAgent(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create agent: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d", project, clusterAgent.ID))
	resp.Diagnostics.Append(data.modelToStateModel(project, clusterAgent)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabClusterAgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabClusterAgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, agentID, err := resourceGitlabClusterAgentParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Read. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read GitLab Agent for Kubernetes in project %s with id %d", project, agentID))
	clusterAgent, _, err := r.client.ClusterAgents.GetAgent(project, agentID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("read GitLab Agent for Kubernetes in project %s with id %d not found, removing from state", project, agentID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read agent: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(project, clusterAgent)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabClusterAgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Provider Error, report upstream",
		"Somehow the resource was requested to perform an in-place upgrade which is not possible.",
	)
}

func (r *gitlabClusterAgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabClusterAgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, agentID, err := resourceGitlabClusterAgentParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID format", fmt.Sprintf("The resource ID '%s' has an invalid format in Delete. Error: %s", data.ID.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("delete GitLab Agent for Kubernetes in project %s with id %d", project, agentID))
	if _, err := r.client.ClusterAgents.DeleteAgent(project, agentID, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete agent: %s", err.Error()))
		return
	}
}

func (d *gitlabClusterAgentResourceModel) modelToStateModel(project string, clusterAgent *gitlab.Agent) diag.Diagnostics {
	d.Project = types.StringValue(project)
	d.Name = types.StringValue(clusterAgent.Name)
	d.AgentID = types.Int64Value(int64(clusterAgent.ID))
	createdAt, diags := timetypes.NewRFC3339Value(clusterAgent.CreatedAt.Format(time.RFC3339))
	if diags != nil && diags.HasError() {
		return diags
	}
	d.CreatedAt = createdAt
	d.CreatedByUserID = types.Int64Value(int64(clusterAgent.CreatedByUserID))
	return nil
}

func resourceGitlabClusterAgentParseID(id string) (string, int64, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid cluster agent id %q, expected format '{project}:{agent_id}'", id)
	}
	project, rawAgentID := parts[0], parts[1]
	agentID, err := strconv.ParseInt(rawAgentID, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid cluster agent id %q with 'agent_id' %q, expected integer", id, rawAgentID)
	}

	return project, agentID, nil
}
