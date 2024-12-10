package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitLabProjectEnvironmentsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabProjectEnvironmentsDataSource{}
)

func init() {
	registerDataSource(newGitLabProjectEnvironmentsDataSource)
}

// NewGitLabRunnersDataSource is a helper function to simplify the provider implementation.
func newGitLabProjectEnvironmentsDataSource() datasource.DataSource {
	return &gitLabProjectEnvironmentsDataSource{}
}

// gitLabProjectEnvironmentsDataSource is the data source implementation.
type gitLabProjectEnvironmentsDataSource struct {
	client *gitlab.Client
}

// gitLabProjectEnvironmentsDataSourceModel describes the data source data model.
type gitLabProjectEnvironmentsDataSourceModel struct {
	ID           types.String         `tfsdk:"id"`
	Project      types.String         `tfsdk:"project"`
	Name         types.String         `tfsdk:"name"`
	Search       types.String         `tfsdk:"search"`
	States       types.String         `tfsdk:"states"`
	Environments []*gitlabEnvironment `tfsdk:"environments"`
}
type gitlabEnvironment struct {
	ID                  types.Int64  `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Slug                types.String `tfsdk:"slug"`
	Description         types.String `tfsdk:"description"`
	State               types.String `tfsdk:"state"`
	Tier                types.String `tfsdk:"tier"`
	ExternalURL         types.String `tfsdk:"external_url"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	ClusterAgentID      types.Int64  `tfsdk:"cluster_agent_id"`
	KubernetesNamespace types.String `tfsdk:"kubernetes_namespace"`
	FluxResourcePath    types.String `tfsdk:"flux_resource_path"`
}

// Metadata returns the data source type name.
func (d *gitLabProjectEnvironmentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environments"
}

// Schema defines the schema for the data source.
func (d *gitLabProjectEnvironmentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	allowedStatesValues := []string{"available", "stopping", "stopped"}
	environmentTiers := []string{"production", "staging", "testing", "development", "other"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_environments`" + ` data source retrieves information about all environments of the given project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/environments.html#list-environments)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Return the environment with this name. Mutually exclusive with search.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("search")),
				},
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Return list of environments matching the search criteria. Mutually exclusive with name. Must be at least 3 characters long. ",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("name")),
				},
			},
			"states": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("List all environments that match the specified state. Valid values are %s. Returns all environments if not set.", utils.RenderValueListForDocs(allowedStatesValues)),
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf(allowedStatesValues...)},
			},
			"environments": schema.ListNestedAttribute{
				MarkdownDescription: "The list of environments.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the environment.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the environment.",
							Computed:            true,
						},
						// see https://docs.gitlab.com/ee/ci/variables/predefined_variables.html, CI_ENVIRONMENT_SLUG. API also truncates and adds a randum suffix.
						"slug": schema.StringAttribute{
							MarkdownDescription: "The simplified version of the environment name, suitable for inclusion in DNS, URLs, Kubernetes labels, and so on. The slug is truncated to 24 characters. A random suffix is automatically added to uppercase environment names.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "The description of the environment.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("The state of the environment. Value can be one of %s. Returns all environments if not set.", utils.RenderValueListForDocs(allowedStatesValues)),
							Computed:            true,
						},
						"tier": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("The tier of the environment. Value can be one of %s. Returns all environments if not set.", utils.RenderValueListForDocs(environmentTiers)),
							Computed:            true,
						},
						"external_url": schema.StringAttribute{
							MarkdownDescription: "Place to link to for this environment.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Timestamp of the environment creation, RFC3339 format.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "Timestamp of the last environment update, RFC3339 format.",
							Computed:            true,
						},
						"cluster_agent_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the environments cluster agent or `null` if none is assigned.",
							Computed:            true,
						},
						"kubernetes_namespace": schema.StringAttribute{
							MarkdownDescription: "The Kubernetes namespace to associate with this environment.",
							Computed:            true,
						},
						"flux_resource_path": schema.StringAttribute{
							MarkdownDescription: "The Flux resource path to associate with this environment.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabProjectEnvironmentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabProjectEnvironmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitLabProjectEnvironmentsDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.ListEnvironmentsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	if !config.Name.IsNull() && !config.Name.IsUnknown() {
		options.Name = config.Name.ValueStringPointer()
	}

	if !config.Search.IsNull() && !config.Search.IsUnknown() {
		options.Search = config.Search.ValueStringPointer()
	}

	if !config.States.IsNull() && !config.States.IsUnknown() {
		options.States = config.States.ValueStringPointer()
	}

	environments, err := d.getAllEnvironments(ctx, config.Project.ValueString(), options)
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API error occurred",
			err.Error(),
		)
		return
	}

	for _, environment := range environments {
		e := &gitlabEnvironment{
			ID:                  types.Int64Value(int64(environment.ID)),
			Name:                types.StringValue(environment.Name),
			Slug:                types.StringValue(environment.Slug),
			Description:         types.StringValue(environment.Description),
			State:               types.StringValue(environment.State),
			Tier:                types.StringValue(environment.Tier),
			ExternalURL:         types.StringValue(environment.ExternalURL),
			CreatedAt:           types.StringValue(environment.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:           types.StringValue(environment.UpdatedAt.Format(time.RFC3339)),
			KubernetesNamespace: types.StringValue(environment.KubernetesNamespace),
			FluxResourcePath:    types.StringValue(environment.FluxResourcePath),
		}
		if environment.ClusterAgent != nil {
			e.ClusterAgentID = types.Int64Value(int64(environment.ClusterAgent.ID))
		}
		config.Environments = append(config.Environments, e)
	}

	diags := resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

func (d *gitLabProjectEnvironmentsDataSource) getAllEnvironments(ctx context.Context, projectID interface{}, options *gitlab.ListEnvironmentsOptions) ([]*gitlab.Environment, error) {
	var environments []*gitlab.Environment
	for options.Page != 0 {
		// Make API call to read environments
		r, resp, err := d.client.Environments.ListEnvironments(projectID, options, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("unable to read environments page %d: %w", options.Page, err)
		}

		environments = append(environments, r...)

		options.Page = resp.NextPage
	}
	return environments, nil
}
