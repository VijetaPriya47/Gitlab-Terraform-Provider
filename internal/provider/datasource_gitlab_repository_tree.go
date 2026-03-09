package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

var (
	_ datasource.DataSource              = &gitlabRepositoryTreeDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabRepositoryTreeDataSource{}
)

func init() {
	registerDataSource(NewGitlabRepositoryTreeDataSource)
}

func NewGitlabRepositoryTreeDataSource() datasource.DataSource {
	return &gitlabRepositoryTreeDataSource{}
}

type gitlabRepositoryTreeDataSource struct {
	client *gitlab.Client
}

type gitlabRepositoryTreeDataSourceModel struct {
	ID        types.String                                    `tfsdk:"id"`
	Project   types.String                                    `tfsdk:"project"`
	Ref       types.String                                    `tfsdk:"ref"`
	Path      types.String                                    `tfsdk:"path"`
	Recursive types.Bool                                      `tfsdk:"recursive"`
	Tree      []gitlabRepositoryTreeIndividualDataSourceModel `tfsdk:"tree"`
}

type gitlabRepositoryTreeIndividualDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	NodeID types.String `tfsdk:"node_id"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	Path   types.String `tfsdk:"path"`
	Mode   types.String `tfsdk:"mode"`
}

func (d *gitlabRepositoryTreeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_tree"
}

func (d *gitlabRepositoryTreeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_repository_tree`" + ` data source allows details of directories and files in a repository to be retrieved.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/repositories/#list-repository-tree)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. A hash of project and ref, with path and recursive if set.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project owned by the authenticated user.",
				Required:            true,
			},
			"ref": schema.StringAttribute{
				MarkdownDescription: "The name of a repository branch or tag.",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The path inside repository. Used to get content of subdirectories.",
				Optional:            true,
			},
			"recursive": schema.BoolAttribute{
				MarkdownDescription: "Boolean value used to get a recursive tree (false by default).",
				Optional:            true,
			},
			"tree": schema.ListNestedAttribute{
				MarkdownDescription: "The list of files/directories returned by the search",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The project ID. Use `node_id` instead. To be removed in 19.0.",
							Computed:            true,
							DeprecationMessage:  "Use `node_id` instead. To be removed in 19.0.",
						},
						"node_id": schema.StringAttribute{
							MarkdownDescription: "The SHA-1 hash of the tree or blob in the repository.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the blob or tree in the repository",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Type of object in the repository. Can be either type tree or of type blob",
							Computed:            true,
						},
						"path": schema.StringAttribute{
							MarkdownDescription: "Path of the object inside of the repository.",
							Computed:            true,
						},
						"mode": schema.StringAttribute{
							MarkdownDescription: "Unix access mode of the file in the repository.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabRepositoryTreeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabRepositoryTreeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabRepositoryTreeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()

	options := &gitlab.ListTreeOptions{
		Path:      data.Path.ValueStringPointer(),
		Ref:       data.Ref.ValueStringPointer(),
		Recursive: data.Recursive.ValueBoolPointer(),
	}

	nodes, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
		return d.client.Repositories.ListTree(project, options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read repository tree: %s", err.Error()))
		return
	}

	var optionsHash strings.Builder
	optionsHash.WriteString(project)
	optionsHash.WriteString(data.Path.ValueString())
	if !data.Ref.IsNull() && !data.Ref.IsUnknown() {
		optionsHash.WriteString(data.Ref.ValueString())
	}
	if !data.Recursive.IsNull() && !data.Recursive.IsUnknown() {
		optionsHash.WriteString(strconv.FormatBool(data.Recursive.ValueBool()))
	}

	data.ID = types.StringValue(optionsHash.String())
	data.Tree = []gitlabRepositoryTreeIndividualDataSourceModel{}
	for _, node := range nodes {
		modelNode := gitlabRepositoryTreeIndividualDataSourceModel{
			ID:     types.StringValue(project),
			NodeID: types.StringValue(node.ID),
			Name:   types.StringValue(node.Name),
			Type:   types.StringValue(node.Type),
			Path:   types.StringValue(node.Path),
			Mode:   types.StringValue(node.Mode),
		}
		data.Tree = append(data.Tree, modelNode)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
