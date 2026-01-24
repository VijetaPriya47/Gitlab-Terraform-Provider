package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabGroupVariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupVariablesDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupVariablesDataSource)
}

// NewGitlabGroupVariablesDataSource is a helper function to simplify the provider implementation.
func NewGitlabGroupVariablesDataSource() datasource.DataSource {
	return &gitlabGroupVariablesDataSource{}
}

// gitlabGroupVariablesDataSource is the data source implementation.
type gitlabGroupVariablesDataSource struct {
	client *gitlab.Client
}

// gitlabGroupVariablesDataSourceModel describes the data source data model.
type gitlabGroupVariablesDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Group            types.String `tfsdk:"group"`
	EnvironmentScope types.String `tfsdk:"environment_scope"`
	Variables        types.List   `tfsdk:"variables"`
}

func variableDataAttibutes() map[string]attr.Type {
	return map[string]attr.Type{
		"group":             types.StringType,
		"key":               types.StringType,
		"value":             types.StringType,
		"variable_type":     types.StringType,
		"protected":         types.BoolType,
		"masked":            types.BoolType,
		"environment_scope": types.StringType,
		"raw":               types.BoolType,
		"description":       types.StringType,
	}
}

// Metadata returns the data source type name.
func (d *gitlabGroupVariablesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_variables"
}

// Schema defines the schema for the data source.
func (d *gitlabGroupVariablesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_variables`" + ` data source allows to retrieve all group-level CI/CD variables.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_level_variables/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>:<service_account_id>`.",
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The name or id of the group.",
				Required:            true,
			},
			"environment_scope": schema.StringAttribute{
				MarkdownDescription: "The environment scope of the variable. Defaults to all environment (`*`).",
				Optional:            true,
			},
			"variables": schema.ListAttribute{
				MarkdownDescription: "The list of variables returned by the search",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: variableDataAttibutes(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabGroupVariablesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabGroupVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitlabGroupVariablesDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	group := state.Group.ValueString()
	environmentScope := state.EnvironmentScope.ValueString()
	options := &gitlab.ListGroupVariablesOptions{}
	variables, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error) {
		return d.client.GroupVariables.ListVariables(group, options, p, gitlab.WithContext(ctx), utils.WithEnvironmentScopeFilter(ctx, environmentScope))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read group variables: %s", err.Error()))
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s:%s", group, environmentScope))
	var diags diag.Diagnostics
	state.Variables, diags = types.ListValue(types.ObjectType{AttrTypes: variableDataAttibutes()}, flattenGitlabGroupVariables(group, variables))
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func flattenGitlabGroupVariables(group string, variables []*gitlab.GroupVariable) (values []attr.Value) {
	for _, variable := range variables {
		values = append(values, gitlabGroupVariableToObjectValue(group, variable))
	}
	return values
}

func gitlabGroupVariableToObjectValue(group string, variable *gitlab.GroupVariable) basetypes.ObjectValue {
	return types.ObjectValueMust(variableDataAttibutes(), map[string]attr.Value{
		"group":             types.StringValue(group),
		"key":               types.StringValue(variable.Key),
		"value":             types.StringValue(variable.Value),
		"variable_type":     types.StringValue(string(variable.VariableType)),
		"protected":         types.BoolValue(variable.Protected),
		"masked":            types.BoolValue(variable.Masked),
		"environment_scope": types.StringValue(variable.EnvironmentScope),
		"raw":               types.BoolValue(variable.Raw),
		"description":       types.StringValue(variable.Description),
	})
}
