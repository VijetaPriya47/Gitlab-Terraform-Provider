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
	_ datasource.DataSource              = &gitlabRepositoryFileDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabRepositoryFileDataSource{}
)

func init() {
	registerDataSource(NewGitlabRepositoryFileDataSource)
}

func NewGitlabRepositoryFileDataSource() datasource.DataSource {
	return &gitlabRepositoryFileDataSource{}
}

type gitlabRepositoryFileDataSource struct {
	client *gitlab.Client
}

type gitlabRepositoryFileDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Project         types.String `tfsdk:"project"`
	FilePath        types.String `tfsdk:"file_path"`
	Content         types.String `tfsdk:"content"`
	Ref             types.String `tfsdk:"ref"`
	FileName        types.String `tfsdk:"file_name"`
	Size            types.Int64  `tfsdk:"size"`
	Encoding        types.String `tfsdk:"encoding"`
	ContentSHA256   types.String `tfsdk:"content_sha256"`
	ExecuteFilemode types.Bool   `tfsdk:"execute_filemode"`
	BlobID          types.String `tfsdk:"blob_id"`
	CommitID        types.String `tfsdk:"commit_id"`
	LastCommitID    types.String `tfsdk:"last_commit_id"`
}

func (d *gitlabRepositoryFileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_file"
}

func (d *gitlabRepositoryFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_repository_file`" + ` data source allows details of a file in a repository to be retrieved.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/repository_files/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:ref:file_path>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or ID of the project.",
				Required:            true,
			},
			"file_path": schema.StringAttribute{
				MarkdownDescription: "The full path of the file. It must be relative to the root of the project without a leading slash `/` or `./`.",
				Required:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "File content.",
				Computed:            true,
			},
			"ref": schema.StringAttribute{
				MarkdownDescription: "The name of branch, tag or commit.",
				Required:            true,
			},
			"file_name": schema.StringAttribute{
				MarkdownDescription: "The filename.",
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "The file size.",
				Computed:            true,
			},
			"encoding": schema.StringAttribute{
				MarkdownDescription: "The file content encoding.",
				Computed:            true,
			},
			"content_sha256": schema.StringAttribute{
				MarkdownDescription: "File content sha256 digest.",
				Computed:            true,
			},
			"execute_filemode": schema.BoolAttribute{
				MarkdownDescription: "Enables or disables the execute flag on the file.",
				Computed:            true,
			},
			"blob_id": schema.StringAttribute{
				MarkdownDescription: "The blob id.",
				Computed:            true,
			},
			"commit_id": schema.StringAttribute{
				MarkdownDescription: "The commit id.",
				Computed:            true,
			},
			"last_commit_id": schema.StringAttribute{
				MarkdownDescription: "The last known commit id.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabRepositoryFileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabRepositoryFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabRepositoryFileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()
	filePath := data.FilePath.ValueString()

	options := &gitlab.GetFileOptions{
		Ref: data.Ref.ValueStringPointer(),
	}

	repositoryFile, _, err := d.client.RepositoryFiles.GetFile(project, filePath, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project repository file: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", project, repositoryFile.Ref, repositoryFile.FilePath))
	data.Project = types.StringValue(project)
	data.FileName = types.StringValue(repositoryFile.FileName)
	data.FilePath = types.StringValue(repositoryFile.FilePath)
	data.Size = types.Int64Value(int64(repositoryFile.Size))
	data.Encoding = types.StringValue(repositoryFile.Encoding)
	data.Content = types.StringValue(repositoryFile.Content)
	data.ContentSHA256 = types.StringValue(repositoryFile.SHA256)
	data.ExecuteFilemode = types.BoolValue(repositoryFile.ExecuteFilemode)
	data.Ref = types.StringValue(repositoryFile.Ref)
	data.BlobID = types.StringValue(repositoryFile.BlobID)
	data.CommitID = types.StringValue(repositoryFile.CommitID)
	data.LastCommitID = types.StringValue(repositoryFile.LastCommitID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
