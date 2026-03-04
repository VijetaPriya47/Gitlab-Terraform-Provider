package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabRunnerControllersDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabRunnerControllersDataSource{}
)

func init() {
	registerDataSource(NewGitlabRunnerControllersDataSource)
}

func NewGitlabRunnerControllersDataSource() datasource.DataSource {
	return &gitlabRunnerControllersDataSource{}
}

type gitlabRunnerControllersDataSource struct {
	client *gitlab.Client
}

type gitlabRunnerControllersDataSourceModel struct {
	ID                types.String                                       `tfsdk:"id"`
	RunnerControllers []gitlabRunnerControllersIndividualDataSourceModel `tfsdk:"runner_controllers"`
}

type gitlabRunnerControllersIndividualDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *gitlabRunnerControllersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_controllers"
}

func (d *gitlabRunnerControllersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	validRunnerControllerStates := []string{"disabled", "enabled", "dry_run"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_runner_controllers`" + ` data source retrieves all runner controllers.

~> This data source is **experimental** and may change or be removed in future versions. Introduced in GitLab 18.9.

-> This data source requires administration privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/runner_controllers/#list-all-runner-controllers)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source.",
				Computed:            true,
			},
			"runner_controllers": schema.ListNestedAttribute{
				MarkdownDescription: "The list of runner controllers.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the runner controller.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "The description of the runner controller.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("The state of the runner controller. Valid values are: %s.", utils.RenderValueListForDocs(validRunnerControllerStates)),
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *gitlabRunnerControllersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds := req.ProviderData.(*GitLabDatasourceData)
	d.client = ds.Client
}

func (d *gitlabRunnerControllersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabRunnerControllersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := gitlab.ListRunnerControllersOptions{}
	controllers, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.RunnerController, *gitlab.Response, error) {
		return d.client.RunnerControllers.ListRunnerControllers(&options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to list runner controllers: %s", err.Error()))
		return
	}

	data.ID = types.StringValue("runner_controllers")
	data.RunnerControllers = []gitlabRunnerControllersIndividualDataSourceModel{}
	for _, c := range controllers {
		model := gitlabRunnerControllersIndividualDataSourceModel{
			ID:          types.Int64Value(c.ID),
			Description: types.StringValue(c.Description),
			State:       types.StringValue(string(c.State)),
		}
		if c.CreatedAt != nil {
			model.CreatedAt = types.StringValue(c.CreatedAt.Format(time.RFC3339))
		}
		if c.UpdatedAt != nil {
			model.UpdatedAt = types.StringValue(c.UpdatedAt.Format(time.RFC3339))
		}
		data.RunnerControllers = append(data.RunnerControllers, model)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
