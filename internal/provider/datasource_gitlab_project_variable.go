package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabProjectVariableDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectVariableDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectVariableDataSource)
}

func NewGitlabProjectVariableDataSource() datasource.DataSource {
	return &gitlabProjectVariableDataSource{}
}

type gitlabProjectVariableDataSource struct {
	client *gitlab.Client
}

type gitlabProjectVariableDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Project          types.String `tfsdk:"project"`
	Key              types.String `tfsdk:"key"`
	Value            types.String `tfsdk:"value"`
	VariableType     types.String `tfsdk:"variable_type"`
	Protected        types.Bool   `tfsdk:"protected"`
	Masked           types.Bool   `tfsdk:"masked"`
	EnvironmentScope types.String `tfsdk:"environment_scope"`
	Raw              types.Bool   `tfsdk:"raw"`
	Description      types.String `tfsdk:"description"`
}

func (d *gitlabProjectVariableDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_variable"
}

func (d *gitlabProjectVariableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_variable`" + ` data source allows to retrieve details about a project-level CI/CD variable.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_level_variables/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:key:environment-scope>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or path of the project.",
				Required:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The name of the variable.",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the variable.",
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
			"environment_scope": schema.StringAttribute{
				MarkdownDescription: "The environment scope of the variable. Defaults to all environment (`*`).",
				Optional:            true,
				Computed:            true,
			},
			"raw": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the variable will be treated as a raw string.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the variable. Maximum of 255 characters.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabProjectVariableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectVariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectVariableDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()
	key := data.Key.ValueString()
	environmentScope := "*"
	if !data.EnvironmentScope.IsNull() && !data.EnvironmentScope.IsUnknown() {
		environmentScope = data.EnvironmentScope.ValueString()
	}

	variable, _, err := d.client.ProjectVariables.GetVariable(project, key, nil, gitlab.WithContext(ctx), utils.WithEnvironmentScopeFilter(ctx, environmentScope))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project variable: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", project, key, environmentScope))
	data.Project = types.StringValue(project)
	data.Key = types.StringValue(key)
	data.Value = types.StringValue(variable.Value)
	data.VariableType = types.StringValue(string(variable.VariableType))
	data.Protected = types.BoolValue(variable.Protected)
	data.Masked = types.BoolValue(variable.Masked)
	data.EnvironmentScope = types.StringValue(variable.EnvironmentScope)
	data.Raw = types.BoolValue(variable.Raw)
	data.Description = types.StringValue(variable.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
