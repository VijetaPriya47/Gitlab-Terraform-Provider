package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	_ datasource.DataSource              = &gitlabInstanceDeployKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabInstanceDeployKeysDataSource{}
)

func init() {
	registerDataSource(NewGitlabInstanceDeployKeysDataSource)
}

func NewGitlabInstanceDeployKeysDataSource() datasource.DataSource {
	return &gitlabInstanceDeployKeysDataSource{}
}

type gitlabInstanceDeployKeysDataSource struct {
	client *gitlab.Client
}

type gitlabInstanceDeployKeysDataSourceModel struct {
	ID         types.String                                    `tfsdk:"id"`
	Public     types.Bool                                      `tfsdk:"public"`
	DeployKeys []gitlabInstanceDeployKeysDataSourceNestedModel `tfsdk:"deploy_keys"`
}

type gitlabInstanceDeployKeysDataSourceNestedModel struct {
	ID                      types.Int64                                           `tfsdk:"id"`
	Title                   types.String                                          `tfsdk:"title"`
	CreatedAt               types.String                                          `tfsdk:"created_at"`
	Key                     types.String                                          `tfsdk:"key"`
	Fingerprint             types.String                                          `tfsdk:"fingerprint"`
	ProjectsWithWriteAccess []gitlabInstanceDeployKeysProjectWithWriteAccessModel `tfsdk:"projects_with_write_access"`
}

type gitlabInstanceDeployKeysProjectWithWriteAccessModel struct {
	ID                types.Int64  `tfsdk:"id"`
	Description       types.String `tfsdk:"description"`
	Name              types.String `tfsdk:"name"`
	NameWithNamespace types.String `tfsdk:"name_with_namespace"`
	Path              types.String `tfsdk:"path"`
	PathWithNamespace types.String `tfsdk:"path_with_namespace"`
	CreatedAt         types.String `tfsdk:"created_at"`
}

func (d *gitlabInstanceDeployKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_deploy_keys"
}

func (d *gitlabInstanceDeployKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_instance_deploy_keys`" + ` data source allows to retrieve a list of deploy keys for a GitLab instance.

-> This data source requires administration privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/deploy_keys/#list-all-deploy-keys)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<public>`.",
				Computed:            true,
			},
			"public": schema.BoolAttribute{
				Description: "Only return deploy keys that are public.",
				Optional:    true,
			},
			"deploy_keys": schema.ListNestedAttribute{
				MarkdownDescription: "The list of all deploy keys across all projects of the GitLab instance.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the deploy key.",
							Computed:            true,
						},
						"title": schema.StringAttribute{
							MarkdownDescription: "The title of the deploy key.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "The creation date of the deploy key. In RFC3339 format.",
							Computed:            true,
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "The deploy key.",
							Computed:            true,
						},
						"fingerprint": schema.StringAttribute{
							MarkdownDescription: "The fingerprint of the deploy key.",
							Computed:            true,
						},
						"projects_with_write_access": schema.ListNestedAttribute{
							MarkdownDescription: "The list of projects that the deploy key has write access to.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										MarkdownDescription: "The ID of the project.",
										Computed:            true,
									},
									"description": schema.StringAttribute{
										MarkdownDescription: "The description of the project.",
										Computed:            true,
									},
									"name": schema.StringAttribute{
										MarkdownDescription: "The name of the project.",
										Computed:            true,
									},
									"name_with_namespace": schema.StringAttribute{
										MarkdownDescription: "The name of the project with namespace.",
										Computed:            true,
									},
									"path": schema.StringAttribute{
										MarkdownDescription: "The path of the project.",
										Computed:            true,
									},
									"path_with_namespace": schema.StringAttribute{
										MarkdownDescription: "The path of the project with namespace.",
										Computed:            true,
									},
									"created_at": schema.StringAttribute{
										MarkdownDescription: "The creation date of the project. In RFC3339 format.",
										Computed:            true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *gitlabInstanceDeployKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabInstanceDeployKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabInstanceDeployKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.ListInstanceDeployKeysOptions{}
	if !data.Public.IsNull() && !data.Public.IsUnknown() {
		options.Public = data.Public.ValueBoolPointer()
	} else {
		options.Public = gitlab.Ptr(false)
	}

	tflog.Info(ctx, fmt.Sprintf("Reading Instance Deploy Keys, with: %v", options))

	instanceDeployKeys, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.InstanceDeployKey, *gitlab.Response, error) {
		return d.client.DeployKeys.ListAllDeployKeys(options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read instance deploy keys: %s", err.Error()))
		return
	}

	// NOTE: this data source doesn't have a "real" id, but the query to the API
	//       should actually return the same response for the same options,
	//       therefore the `Public` field option is used as id.
	data.ID = types.StringValue(fmt.Sprintf("%b", options.Public))
	data.Public = types.BoolValue(data.Public.ValueBool())

	for _, deployKey := range instanceDeployKeys {
		modelKey := gitlabInstanceDeployKeysDataSourceNestedModel{
			ID:          types.Int64Value(int64(deployKey.ID)),
			Title:       types.StringValue(deployKey.Title),
			CreatedAt:   types.StringValue(deployKey.CreatedAt.Format(time.RFC3339)),
			Key:         types.StringValue(deployKey.Key),
			Fingerprint: types.StringValue(deployKey.Fingerprint),
		}

		for _, project := range deployKey.ProjectsWithWriteAccess {
			modelProject := gitlabInstanceDeployKeysProjectWithWriteAccessModel{
				ID:                types.Int64Value(int64(project.ID)),
				Description:       types.StringValue(project.Description),
				Name:              types.StringValue(project.Name),
				NameWithNamespace: types.StringValue(project.NameWithNamespace),
				Path:              types.StringValue(project.Path),
				PathWithNamespace: types.StringValue(project.PathWithNamespace),
				CreatedAt:         types.StringValue(project.CreatedAt.Format(time.RFC3339)),
			}
			modelKey.ProjectsWithWriteAccess = append(modelKey.ProjectsWithWriteAccess, modelProject)
		}

		data.DeployKeys = append(data.DeployKeys, modelKey)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
