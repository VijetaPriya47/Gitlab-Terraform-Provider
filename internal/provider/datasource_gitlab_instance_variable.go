package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

var (
	_ datasource.DataSource              = &gitlabInstanceVariableDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabInstanceVariableDataSource{}
)

func init() {
	registerDataSource(NewGitlabInstanceVariableDataSource)
}

func NewGitlabInstanceVariableDataSource() datasource.DataSource {
	return &gitlabInstanceVariableDataSource{}
}

type gitlabInstanceVariableDataSource struct {
	client *gitlab.Client
}

type gitlabInstanceVariableDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Key          types.String `tfsdk:"key"`
	Value        types.String `tfsdk:"value"`
	Description  types.String `tfsdk:"description"`
	VariableType types.String `tfsdk:"variable_type"`
	Protected    types.Bool   `tfsdk:"protected"`
	Masked       types.Bool   `tfsdk:"masked"`
	Raw          types.Bool   `tfsdk:"raw"`
}

func (d *gitlabInstanceVariableDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_variable"
}

func (d *gitlabInstanceVariableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_instance_variable`" + ` data source retrieves details about an instance-level CI/CD variable.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/instance_level_ci_variables/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<key>`.",
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The name of the variable.",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the variable.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the variable. Maximum of 255 characters.",
				Computed:            true,
			},
			"variable_type": schema.StringAttribute{
				MarkdownDescription: "The type of the variable, either `env_var` or `file`.",
				Computed:            true,
			},
			"protected": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the variable will be passed only to pipelines running on protected branches and tags.",
				Computed:            true,
			},
			"masked": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the value of the variable will be hidden in job logs.",
				Computed:            true,
			},
			"raw": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the variable will be treated as a raw string.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabInstanceVariableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabInstanceVariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabInstanceVariableDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := data.Key.ValueString()

	variable, _, err := d.client.InstanceVariables.GetVariable(key, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read instance variable: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(key)
	data.Key = types.StringValue(key)
	data.Value = types.StringValue(variable.Value)
	data.Description = types.StringValue(variable.Description)
	data.VariableType = types.StringValue(string(variable.VariableType))
	data.Protected = types.BoolValue(variable.Protected)
	data.Masked = types.BoolValue(variable.Masked)
	data.Raw = types.BoolValue(variable.Raw)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
