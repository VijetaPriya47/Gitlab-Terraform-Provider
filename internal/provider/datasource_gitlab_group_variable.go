package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabGroupVariableDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupVariableDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupVariableDataSource)
}

// NewGitlabGroupVariableDataSource is a helper function to simplify the provider implementation.
func NewGitlabGroupVariableDataSource() datasource.DataSource {
	return &gitlabGroupVariableDataSource{}
}

// gitlabGroupVariableDataSource is the data source implementation.
type gitlabGroupVariableDataSource struct {
	client *gitlab.Client
}

// gitlabGroupVariableDataSourceModel describes the data source data model.
type gitlabGroupVariableDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Group            types.String `tfsdk:"group"`
	Key              types.String `tfsdk:"key"`
	Value            types.String `tfsdk:"value"`
	VariableType     types.String `tfsdk:"variable_type"`
	Protected        types.Bool   `tfsdk:"protected"`
	Masked           types.Bool   `tfsdk:"masked"`
	EnvironmentScope types.String `tfsdk:"environment_scope"`
	Raw              types.Bool   `tfsdk:"raw"`
	Description      types.String `tfsdk:"description"`
}

// Metadata implements datasource.DataSource.
func (d *gitlabGroupVariableDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_variable"
}

// Schema defines the schema for the data source.
func (d *gitlabGroupVariableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_variable`" + ` data source allows to retrieve details about a group-level CI/CD variable.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_level_variables/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>:<key>:<environment_scope>`.",
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The name or id of the group.",
				Required:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The name of the variable.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(regexpGitlabVariableName, "is an invalid, only A-Z, a-z, 0-9, and _ are allowed"),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the variable.",
				Computed:            true,
			},
			"variable_type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The type of a variable. Valid values are: %s.", utils.RenderValueListForDocs(gitlabVariableTypeValues)),
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(gitlabVariableTypeValues...)},
			},
			"protected": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the variable will be passed only to pipelines running on protected branches and tags",
				Computed:            true,
			},
			"masked": schema.BoolAttribute{
				MarkdownDescription: "If set to `true`, the value of the variable will be hidden in job logs. The value must meet the [masking requirements](https://docs.gitlab.com/ci/variables/#mask-a-cicd-variable).",
				Computed:            true,
			},
			"environment_scope": schema.StringAttribute{
				MarkdownDescription: "The environment scope of the variable. Defaults to all environment (`*`). Note that in Community Editions of Gitlab, values other than `*` will cause inconsistent plans.",
				Optional:            true,
				Computed:            true,
			},
			"raw": schema.BoolAttribute{
				MarkdownDescription: "Whether the variable is treated as a raw string. Default: false. When true, variables in the value are not expanded.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the variable.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabGroupVariableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabGroupVariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitlabGroupVariableDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	group := state.Group.ValueString()
	key := state.Key.ValueString()
	environmentScope := state.EnvironmentScope.ValueString()

	variable, _, err := d.client.GroupVariables.GetVariable(group, key, nil, gitlab.WithContext(ctx), utils.WithEnvironmentScopeFilter(ctx, environmentScope))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read group variable: %s", err.Error()))
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", group, key, environmentScope))
	state.Group = types.StringValue(group)
	state.Key = types.StringValue(variable.Key)
	state.Value = types.StringValue(variable.Value)
	state.VariableType = types.StringValue(string(variable.VariableType))
	state.Protected = types.BoolValue(variable.Protected)
	state.Masked = types.BoolValue(variable.Masked)
	state.EnvironmentScope = types.StringValue(variable.EnvironmentScope)
	state.Raw = types.BoolValue(variable.Raw)
	state.Description = types.StringValue(variable.Description)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
