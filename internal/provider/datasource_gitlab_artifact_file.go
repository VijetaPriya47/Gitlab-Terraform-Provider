package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

var (
	_ datasource.DataSource              = &gitlabArtifactFileDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabArtifactFileDataSource{}
)

func init() {
	registerDataSource(NewGitlabArtifactFileDataSource)
}

func NewGitlabArtifactFileDataSource() datasource.DataSource {
	return &gitlabArtifactFileDataSource{}
}

type gitlabArtifactFileDataSource struct {
	client *gitlab.Client
}

type gitlabArtifactFileDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Project       types.String `tfsdk:"project"`
	Job           types.String `tfsdk:"job"`
	Ref           types.String `tfsdk:"ref"`
	ArtifactPath  types.String `tfsdk:"artifact_path"`
	MaxSizeBytes  types.Int64  `tfsdk:"max_size_bytes"`
	Content       types.String `tfsdk:"content"`
	ContentBase64 types.String `tfsdk:"content_base64"`
}

func (d *gitlabArtifactFileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_artifact_file"
}

func (d *gitlabArtifactFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_artifact_file`" + ` data source allows downloading a single artifact file from a specific job in the latest successful pipeline for a given reference (branch, tag, or commit).
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/job_artifacts/#download-a-single-artifact-file-by-reference-name)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project>:<ref>:<job>:<artifact_path>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
			},
			"job": schema.StringAttribute{
				MarkdownDescription: "The name of the job.",
				Required:            true,
			},
			"ref": schema.StringAttribute{
				MarkdownDescription: "The name of the branch, tag, or commit SHA.",
				Required:            true,
			},
			"artifact_path": schema.StringAttribute{
				MarkdownDescription: "Path to the artifact file within the job's artifacts archive. This path is relative to the archive contents (not the local filesystem). Ensure each `gitlab_artifact_file` data source in your configuration uses a unique artifact_path to avoid ambiguity.",
				Required:            true,
			},
			"max_size_bytes": schema.Int64Attribute{
				MarkdownDescription: "Maximum bytes to read from the artifact. Defaults to 10MB (10485760 bytes).",
				Optional:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The content of the artifact file as a UTF-8 string. Use `content_base64` for binary files.",
				Computed:            true,
			},
			"content_base64": schema.StringAttribute{
				MarkdownDescription: "The content of the artifact file as a base64-encoded string. Useful for binary files.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabArtifactFileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabArtifactFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabArtifactFileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	job := data.Job.ValueString()
	ref := data.Ref.ValueString()
	artifactPath := data.ArtifactPath.ValueString()

	options := &gitlab.DownloadArtifactsFileOptions{
		Job: &job,
	}

	artifactReader, _, err := d.client.Jobs.DownloadSingleArtifactsFileByTagOrBranch(
		project,
		ref,
		artifactPath,
		options,
		gitlab.WithContext(ctx),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API error occurred",
			fmt.Sprintf("Unable to download artifact file: %s", err.Error()),
		)
		return
	}

	// Read the artifact content with a size limit to prevent OOM errors
	// Default to 10MB, but allow user override
	maxArtifactSize := int64(10 * 1024 * 1024) // 10MB default
	if !data.MaxSizeBytes.IsNull() && data.MaxSizeBytes.ValueInt64() > 0 {
		maxArtifactSize = data.MaxSizeBytes.ValueInt64()
	}
	limitedReader := io.LimitReader(artifactReader, maxArtifactSize+1)

	artifactBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read artifact content",
			fmt.Sprintf("Unable to read artifact file content: %s", err.Error()),
		)
		return
	}

	// Check if the artifact exceeded the size limit
	if int64(len(artifactBytes)) > maxArtifactSize {
		resp.Diagnostics.AddError(
			"Artifact file too large",
			fmt.Sprintf("Artifact file exceeds maximum size of %d bytes. Consider downloading the file using other means or reducing the artifact size.", maxArtifactSize),
		)
		return
	}

	// Set the ID
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s:%s", project, ref, job, artifactPath))

	// Store both UTF-8 string and base64 versions
	if utf8.Valid(artifactBytes) {
		data.Content = types.StringValue(string(artifactBytes))
	} else {
		data.Content = types.StringNull()
	}
	data.ContentBase64 = types.StringValue(base64.StdEncoding.EncodeToString(artifactBytes))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
