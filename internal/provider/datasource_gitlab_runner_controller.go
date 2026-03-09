package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabRunnerControllerDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabRunnerControllerDataSource{}
)

func init() {
	registerDataSource(NewGitlabRunnerControllerDataSource)
}

func NewGitlabRunnerControllerDataSource() datasource.DataSource {
	return &gitlabRunnerControllerDataSource{}
}

type gitlabRunnerControllerDataSource struct {
	client *gitlab.Client
}

type gitlabRunnerControllerDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *gitlabRunnerControllerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_controller"
}

func (d *gitlabRunnerControllerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	validRunnerControllerStates := []string{"disabled", "enabled", "dry_run"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_runner_controller`" + ` data source retrieves details about a single runner controller.

~> This data source is **experimental** and may change or be removed in future versions. Introduced in GitLab 18.9.

-> This data source requires administration privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/runner_controllers/#retrieve-a-single-runner-controller)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the runner controller.",
				Required:            true,
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
	}
}

func (d *gitlabRunnerControllerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds := req.ProviderData.(*GitLabDatasourceData)
	d.client = ds.Client
}

func (d *gitlabRunnerControllerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabRunnerControllerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueInt64()

	controller, _, err := d.client.RunnerControllers.GetRunnerController(id, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read runner controller: %s", err.Error()))
		return
	}

	data.ID = types.Int64Value(controller.ID)
	data.Description = types.StringValue(controller.Description)
	data.State = types.StringValue(string(controller.State))
	if controller.CreatedAt != nil {
		data.CreatedAt = types.StringValue(controller.CreatedAt.Format(time.RFC3339))
	}
	if controller.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(controller.UpdatedAt.Format(time.RFC3339))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
