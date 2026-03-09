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
	_ datasource.DataSource              = &gitlabInstanceVariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabInstanceVariablesDataSource{}
)

func init() {
	registerDataSource(NewGitlabInstanceVariablesDataSource)
}

func NewGitlabInstanceVariablesDataSource() datasource.DataSource {
	return &gitlabInstanceVariablesDataSource{}
}

type gitlabInstanceVariablesDataSource struct {
	client *gitlab.Client
}

type gitlabInstanceVariablesDataSourceModel struct {
	ID        types.String                                       `tfsdk:"id"`
	Variables []gitlabInstanceVariablesIndividualDataSourceModel `tfsdk:"variables"`
}

type gitlabInstanceVariablesIndividualDataSourceModel struct {
	Key          types.String `tfsdk:"key"`
	Value        types.String `tfsdk:"value"`
	Description  types.String `tfsdk:"description"`
	VariableType types.String `tfsdk:"variable_type"`
	Protected    types.Bool   `tfsdk:"protected"`
	Masked       types.Bool   `tfsdk:"masked"`
	Raw          types.Bool   `tfsdk:"raw"`
}

func (d *gitlabInstanceVariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_variables"
}

func (d *gitlabInstanceVariablesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_instance_variables`" + ` data source retrieves all instance-level CI/CD variables.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/instance_level_ci_variables/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the hardcoded format `instance_variables`.",
				Computed:            true,
			},
			"variables": schema.ListNestedAttribute{
				MarkdownDescription: "The list of variables returned by the search.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							MarkdownDescription: "The name of the variable.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *gitlabInstanceVariablesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabInstanceVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabInstanceVariablesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.ListInstanceVariablesOptions{}
	variables, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.InstanceVariable, *gitlab.Response, error) {
		return d.client.InstanceVariables.ListVariables(options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read instance variables: %s", err.Error()))
		return
	}

	data.ID = types.StringValue("instance_variables")

	data.Variables = []gitlabInstanceVariablesIndividualDataSourceModel{}
	for _, variable := range variables {
		modelVariable := gitlabInstanceVariablesIndividualDataSourceModel{
			Key:          types.StringValue(variable.Key),
			Value:        types.StringValue(variable.Value),
			Description:  types.StringValue(variable.Description),
			VariableType: types.StringValue(string(variable.VariableType)),
			Protected:    types.BoolValue(variable.Protected),
			Masked:       types.BoolValue(variable.Masked),
			Raw:          types.BoolValue(variable.Raw),
		}
		data.Variables = append(data.Variables, modelVariable)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
