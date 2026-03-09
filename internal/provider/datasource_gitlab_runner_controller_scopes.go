package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

var (
	_ datasource.DataSource              = &gitlabRunnerControllerScopesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabRunnerControllerScopesDataSource{}
)

func init() {
	registerDataSource(NewGitlabRunnerControllerScopesDataSource)
}

func NewGitlabRunnerControllerScopesDataSource() datasource.DataSource {
	return &gitlabRunnerControllerScopesDataSource{}
}

type gitlabRunnerControllerScopesDataSource struct {
	client *gitlab.Client
}

type gitlabRunnerControllerScopesDataSourceModel struct {
	ID                    types.String                                               `tfsdk:"id"`
	RunnerControllerID    types.Int64                                                `tfsdk:"runner_controller_id"`
	InstanceLevelScopings []gitlabRunnerControllerScopesInstanceLevelDataSourceModel `tfsdk:"instance_level_scopings"`
	RunnerLevelScopings   []gitlabRunnerControllerScopesRunnerLevelDataSourceModel   `tfsdk:"runner_level_scopings"`
}

type gitlabRunnerControllerScopesInstanceLevelDataSourceModel struct {
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

type gitlabRunnerControllerScopesRunnerLevelDataSourceModel struct {
	RunnerID  types.Int64  `tfsdk:"runner_id"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *gitlabRunnerControllerScopesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_controller_scopes"
}

func (d *gitlabRunnerControllerScopesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_runner_controller_scopes`" + ` data source retrieves scopes for a runner controller.

~> This data source is **experimental** and may change or be removed in future versions. Introduced in GitLab 18.10.

-> This data source requires administration privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/runner_controllers/#list-all-scopes-for-a-runner-controller)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source. In the format of `<runner_controller_id>`.",
				Computed:            true,
			},
			"runner_controller_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the runner controller.",
				Required:            true,
			},
			"instance_level_scopings": schema.ListNestedAttribute{
				MarkdownDescription: "The list of instance-level scopings.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"created_at": schema.StringAttribute{
							MarkdownDescription: "The time the scope was created.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "The time the scope was last updated.",
							Computed:            true,
						},
					},
				},
			},
			"runner_level_scopings": schema.ListNestedAttribute{
				MarkdownDescription: "The list of runner-level scopings.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"runner_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the runner.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *gitlabRunnerControllerScopesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds := req.ProviderData.(*GitLabDatasourceData)
	d.client = ds.Client
}

func (d *gitlabRunnerControllerScopesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabRunnerControllerScopesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerID := data.RunnerControllerID.ValueInt64()

	scopes, _, err := d.client.RunnerControllerScopes.ListRunnerControllerScopes(controllerID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read runner controller scopes: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(controllerID, 10))

	data.InstanceLevelScopings = []gitlabRunnerControllerScopesInstanceLevelDataSourceModel{}
	for _, s := range scopes.InstanceLevelScopings {
		model := gitlabRunnerControllerScopesInstanceLevelDataSourceModel{}
		if s.CreatedAt != nil {
			model.CreatedAt = types.StringValue(s.CreatedAt.Format(time.RFC3339))
		}
		if s.UpdatedAt != nil {
			model.UpdatedAt = types.StringValue(s.UpdatedAt.Format(time.RFC3339))
		}
		data.InstanceLevelScopings = append(data.InstanceLevelScopings, model)
	}

	data.RunnerLevelScopings = []gitlabRunnerControllerScopesRunnerLevelDataSourceModel{}
	for _, s := range scopes.RunnerLevelScopings {
		model := gitlabRunnerControllerScopesRunnerLevelDataSourceModel{
			RunnerID: types.Int64Value(s.RunnerID),
		}
		if s.CreatedAt != nil {
			model.CreatedAt = types.StringValue(s.CreatedAt.Format(time.RFC3339))
		}
		if s.UpdatedAt != nil {
			model.UpdatedAt = types.StringValue(s.UpdatedAt.Format(time.RFC3339))
		}
		data.RunnerLevelScopings = append(data.RunnerLevelScopings, model)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
