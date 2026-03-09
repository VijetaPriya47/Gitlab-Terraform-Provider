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
	_ datasource.DataSource              = &gitlabProjectVariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectVariablesDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectVariablesDataSource)
}

func NewGitlabProjectVariablesDataSource() datasource.DataSource {
	return &gitlabProjectVariablesDataSource{}
}

type gitlabProjectVariablesDataSource struct {
	client *gitlab.Client
}

type gitlabProjectVariablesDataSourceModel struct {
	ID               types.String                                      `tfsdk:"id"`
	Project          types.String                                      `tfsdk:"project"`
	EnvironmentScope types.String                                      `tfsdk:"environment_scope"`
	Variables        []gitlabProjectVariablesIndividualDataSourceModel `tfsdk:"variables"`
}

type gitlabProjectVariablesIndividualDataSourceModel struct {
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

func (d *gitlabProjectVariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_variables"
}

func (d *gitlabProjectVariablesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_variables`" + ` data source allows to retrieve all project-level CI/CD variables.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_level_variables/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:environment-scope>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or path of the project.",
				Required:            true,
			},
			"environment_scope": schema.StringAttribute{
				MarkdownDescription: "The environment scope of the variable. Defaults to all environment (`*`).",
				Optional:            true,
			},
			"variables": schema.ListNestedAttribute{
				MarkdownDescription: "The list of variables returned by the search",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project": schema.StringAttribute{
							MarkdownDescription: "The name or path of the project.",
							Computed:            true,
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "The name of the variable.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *gitlabProjectVariablesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectVariablesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()

	environmentScope := "*"
	if !data.EnvironmentScope.IsNull() && !data.EnvironmentScope.IsUnknown() {
		environmentScope = data.EnvironmentScope.ValueString()
	}

	options := &gitlab.ListProjectVariablesOptions{}
	variables, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error) {
		return d.client.ProjectVariables.ListVariables(project, options, p, gitlab.WithContext(ctx), utils.WithEnvironmentScopeFilter(ctx, environmentScope))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project variables: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", project, environmentScope))

	data.Variables = []gitlabProjectVariablesIndividualDataSourceModel{}
	for _, variable := range variables {
		modelVariable := gitlabProjectVariablesIndividualDataSourceModel{
			Project:          types.StringValue(project),
			Key:              types.StringValue(variable.Key),
			Value:            types.StringValue(variable.Value),
			VariableType:     types.StringValue(string(variable.VariableType)),
			Protected:        types.BoolValue(variable.Protected),
			Masked:           types.BoolValue(variable.Masked),
			EnvironmentScope: types.StringValue(variable.EnvironmentScope),
			Raw:              types.BoolValue(variable.Raw),
			Description:      types.StringValue(variable.Description),
		}
		data.Variables = append(data.Variables, modelVariable)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
