package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	_ datasource.DataSource              = &gitlabClusterAgentsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabClusterAgentsDataSource{}
)

func init() {
	registerDataSource(NewGitlabClusterAgentsDataSource)
}

func NewGitlabClusterAgentsDataSource() datasource.DataSource {
	return &gitlabClusterAgentsDataSource{}
}

type gitlabClusterAgentsDataSource struct {
	client *gitlab.Client
}

type gitlabClusterAgentsDataSourceModel struct {
	ID            types.String                                   `tfsdk:"id"`
	Project       types.String                                   `tfsdk:"project"`
	ClusterAgents []gitlabClusterAgentsIndividualDataSourceModel `tfsdk:"cluster_agents"`
}

type gitlabClusterAgentsIndividualDataSourceModel struct {
	Name            types.String      `tfsdk:"name"`
	AgentID         types.Int64       `tfsdk:"agent_id"`
	CreatedAt       timetypes.RFC3339 `tfsdk:"created_at"`
	CreatedByUserID types.Int64       `tfsdk:"created_by_user_id"`
	Project         types.String      `tfsdk:"project"`
}

func (d *gitlabClusterAgentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_agents"
}

func (d *gitlabClusterAgentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_cluster_agents`" + ` data source retrieves details of GitLab Agents for Kubernetes in a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/cluster_agents/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source. In the format <project>",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or full path of the project maintained by the authenticated user.",
				Required:            true,
			},
			"cluster_agents": schema.ListNestedAttribute{
				MarkdownDescription: "List of the registered agents.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"agent_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the agent.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The Name of the agent.",
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
						"project": schema.StringAttribute{
							MarkdownDescription: "ID or full path of the project maintained by the authenticated user.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabClusterAgentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabClusterAgentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabClusterAgentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	options := &gitlab.ListAgentsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}
	var clusterAgents []*gitlab.Agent
	for options.Page != 0 {
		paginatedClusterAgents, response, err := d.client.ClusterAgents.ListAgents(project, options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("Failed to get cluster agents", err.Error())
			return
		}
		clusterAgents = append(clusterAgents, paginatedClusterAgents...)
		options.Page = response.NextPage
	}

	data.ID = types.StringValue(project)
	data.Project = types.StringValue(project)

	data.ClusterAgents = []gitlabClusterAgentsIndividualDataSourceModel{}
	for _, agent := range clusterAgents {
		agentModel := gitlabClusterAgentsIndividualDataSourceModel{
			Name:            types.StringValue(agent.Name),
			AgentID:         types.Int64Value(int64(agent.ID)),
			CreatedByUserID: types.Int64Value(int64(agent.CreatedByUserID)),
			Project:         types.StringValue(fmt.Sprintf("%d", agent.ConfigProject.ID)),
		}
		createdAt, diags := timetypes.NewRFC3339Value(agent.CreatedAt.Format(time.RFC3339))
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		agentModel.CreatedAt = createdAt
		data.ClusterAgents = append(data.ClusterAgents, agentModel)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
