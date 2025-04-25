package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitLabComplianceFrameworkDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabComplianceFrameworkDataSource{}
)

func init() {
	registerDataSource(NewGitLabComplianceFrameworkDataSource)
}

// NewGitLabComplianceFrameworkDataSource is a helper function to simplify the provider implementation.
func NewGitLabComplianceFrameworkDataSource() datasource.DataSource {
	return &gitLabComplianceFrameworkDataSource{}
}

// gitLabComplianceFrameworkDataSource is the data source implementation.
type gitLabComplianceFrameworkDataSource struct {
	client *gitlab.Client
}

// gitLabComplianceFrameworkDataSourceModel describes the data source data model.
type gitLabComplianceFrameworkDataSourceModel struct {
	Id                            types.String `tfsdk:"id"`
	FrameworkId                   types.String `tfsdk:"framework_id"`
	NamespacePath                 types.String `tfsdk:"namespace_path"`
	Name                          types.String `tfsdk:"name"`
	Description                   types.String `tfsdk:"description"`
	Color                         types.String `tfsdk:"color"`
	DefaultFramework              types.Bool   `tfsdk:"default"`
	PipelineConfigurationFullPath types.String `tfsdk:"pipeline_configuration_full_path"`
}

// Metadata returns the data source type name.
func (d *gitLabComplianceFrameworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_framework"
}

// Schema defines the schema for the data source.
func (d *gitLabComplianceFrameworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_compliance_framework`" + ` data source allows details of a compliance framework to be retrieved by its name and the namespace it belongs to.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#querynamespace)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<namespace_path>:<framework_id>`.",
				Computed:            true,
			},
			"framework_id": schema.StringAttribute{
				MarkdownDescription: "Globally unique ID of the compliance framework.",
				Computed:            true,
			},
			"namespace_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the namespace to where the compliance framework is.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name for the compliance framework.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description for the compliance framework.",
				Computed:            true,
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "Color representation of the compliance framework in hex format. e.g. #FCA121.",
				Computed:            true,
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Is the compliance framework the default framework for the group.",
				Computed:            true,
			},
			"pipeline_configuration_full_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the compliance pipeline configuration stored in a project repository, such as `.gitlab/.compliance-gitlab-ci.yml@compliance/hipaa`. Format: `path/file.y[a]ml@group-name/project-name` **Note**: Ultimate license required.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabComplianceFrameworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabComplianceFrameworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabComplianceFrameworkDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			query {
				namespace(fullPath: "%s") {
					fullPath,
					complianceFrameworks(search: "%s", first: 10) {
						nodes {
							id,
							name,
							description,
							color,
							default,
							pipelineConfigurationFullPath
						}
					}
				}
			}`, state.NamespacePath.ValueString(), state.Name.ValueString()),
	}

	var response ComplianceFrameworkResponse
	if _, err := d.client.GraphQL.Do(ctx, query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read compliance framework details: %s", err.Error()))
		return
	}

	// error if 0 compliance frameworks were returned
	if len(response.Data.Namespace.ComplianceFrameworks.Nodes) == 0 {
		resp.Diagnostics.AddError("Compliance Framework not found", fmt.Sprintf("Unable to find Compliance Framework: %s in namespace: %s", state.Name, state.NamespacePath))
		return
	}

	// loop over results looking for exact match
	var matchingFramework api.GraphQLComplianceFramework
	tflog.Debug(ctx, "Found Compliance Frameworks in Namespace", map[string]any{
		"number found": len(response.Data.Namespace.ComplianceFrameworks.Nodes),
		"namespace":    state.NamespacePath.ValueString(),
	})
	for _, v := range response.Data.Namespace.ComplianceFrameworks.Nodes {
		if v.Name == state.Name.ValueString() {
			matchingFramework = v
			break
		}
	}

	if matchingFramework == (api.GraphQLComplianceFramework{}) {
		resp.Diagnostics.AddError("Compliance Framework match not found", fmt.Sprintf("Unable to find exact Compliance Framework match: %s in namespace: %s", state.Name, state.NamespacePath))
		return
	}

	state.Id = types.StringValue(utils.BuildTwoPartID(state.NamespacePath.ValueStringPointer(), &matchingFramework.ID))
	state.FrameworkId = types.StringValue(matchingFramework.ID)
	state.Name = types.StringValue(matchingFramework.Name)
	state.Description = types.StringValue(matchingFramework.Description)
	state.Color = types.StringValue(matchingFramework.Color)
	state.DefaultFramework = types.BoolValue(matchingFramework.DefaultFramework)
	if matchingFramework.PipelineConfigurationFullPath != "" {
		state.PipelineConfigurationFullPath = types.StringValue(matchingFramework.PipelineConfigurationFullPath)
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
