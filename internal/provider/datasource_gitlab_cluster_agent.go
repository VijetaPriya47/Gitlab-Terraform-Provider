package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabClusterAgentDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabClusterAgentDataSource{}
)

func init() {
	registerDataSource(NewGitlabClusterAgentDataSource)
}

func NewGitlabClusterAgentDataSource() datasource.DataSource {
	return &gitlabClusterAgentDataSource{}
}

type gitlabClusterAgentDataSource struct {
	client *gitlab.Client
}

type gitlabClusterAgentDataSourceModel struct {
	ID              types.String      `tfsdk:"id"`
	Project         types.String      `tfsdk:"project"`
	Name            types.String      `tfsdk:"name"`
	AgentID         types.Int64       `tfsdk:"agent_id"`
	CreatedAt       timetypes.RFC3339 `tfsdk:"created_at"`
	CreatedByUserID types.Int64       `tfsdk:"created_by_user_id"`
}

func (d *gitlabClusterAgentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_agent"
}

func (d *gitlabClusterAgentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_cluster_agent`" + ` data source retrieves details about a GitLab Agent for Kubernetes.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/cluster_agents/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source. In the format <project:agent_id>",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or full path of the project maintained by the authenticated user.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The Name of the agent.",
				Computed:            true,
			},
			"agent_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the agent.",
				Required:            true,
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

func (d *gitlabClusterAgentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabClusterAgentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabClusterAgentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	agentID := data.AgentID.ValueInt64()

	agent, _, err := d.client.ClusterAgents.GetAgent(project, agentID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get cluster agent", err.Error())
		return
	}

	agentIDStr := strconv.FormatInt(agentID, 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &agentIDStr))
	data.Project = types.StringValue(project)
	data.Name = types.StringValue(agent.Name)
	data.AgentID = types.Int64Value(int64(agent.ID))
	createdAt, diags := timetypes.NewRFC3339Value(agent.CreatedAt.Format(time.RFC3339))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.CreatedAt = createdAt
	data.CreatedByUserID = types.Int64Value(int64(agent.CreatedByUserID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
