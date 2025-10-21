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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabProjectTagDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectTagDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectTagDataSource)
}

func NewGitlabProjectTagDataSource() datasource.DataSource {
	return &gitlabProjectTagDataSource{}
}

type gitlabProjectTagDataSource struct {
	client *gitlab.Client
}

type gitlabProjectTagDataSourceModel struct {
	ID        types.String                             `tfsdk:"id"`
	Project   types.String                             `tfsdk:"project"`
	Name      types.String                             `tfsdk:"name"`
	Message   types.String                             `tfsdk:"message"`
	Protected types.Bool                               `tfsdk:"protected"`
	Target    types.String                             `tfsdk:"target"`
	Release   []gitlabProjectTagReleaseDataSourceModel `tfsdk:"release"`
	Commit    []gitlabProjectTagCommitDataSourceModel  `tfsdk:"commit"`
}

type gitlabProjectTagReleaseDataSourceModel struct {
	TagName     types.String `tfsdk:"tag_name"`
	Description types.String `tfsdk:"description"`
}

type gitlabProjectTagCommitDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	AuthorEmail    types.String `tfsdk:"author_email"`
	AuthorName     types.String `tfsdk:"author_name"`
	AuthoredDate   types.String `tfsdk:"authored_date"`
	CommittedDate  types.String `tfsdk:"committed_date"`
	CommitterEmail types.String `tfsdk:"committer_email"`
	CommitterName  types.String `tfsdk:"committer_name"`
	ShortID        types.String `tfsdk:"short_id"`
	Title          types.String `tfsdk:"title"`
	Message        types.String `tfsdk:"message"`
	ParentIDs      types.Set    `tfsdk:"parent_ids"`
}

func (d *gitlabProjectTagDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_tag"
}

func (d *gitlabProjectTagDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_tag`" + ` data source allows details of a project tag to be retrieved by its name.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/tags/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:name>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project owned by the authenticated user.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of a tag.",
				Required:            true,
			},
			"message": schema.StringAttribute{
				MarkdownDescription: "The message of the annotated tag.",
				Computed:            true,
			},
			"protected": schema.BoolAttribute{
				MarkdownDescription: "True if tag has tag protection.",
				Computed:            true,
			},
			"target": schema.StringAttribute{
				MarkdownDescription: "The unique id assigned to the commit by Gitlab.",
				Computed:            true,
			},
			"release": schema.SetNestedAttribute{
				MarkdownDescription: "The release associated with the tag.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"tag_name": schema.StringAttribute{
							MarkdownDescription: "The name of the tag.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "The description of the release.",
							Computed:            true,
						},
					},
				},
			},
			"commit": schema.SetNestedAttribute{
				MarkdownDescription: "The commit associated with the tag.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique id assigned to the commit by Gitlab.",
							Computed:            true,
						},
						"author_email": schema.StringAttribute{
							MarkdownDescription: "The email of the author.",
							Computed:            true,
						},
						"author_name": schema.StringAttribute{
							MarkdownDescription: "The name of the author.",
							Computed:            true,
						},
						"authored_date": schema.StringAttribute{
							MarkdownDescription: "The date which the commit was authored (format: yyyy-MM-ddTHH:mm:ssZ).",
							Computed:            true,
						},
						"committed_date": schema.StringAttribute{
							MarkdownDescription: "The date at which the commit was pushed (format: yyyy-MM-ddTHH:mm:ssZ).",
							Computed:            true,
						},
						"committer_email": schema.StringAttribute{
							MarkdownDescription: "The email of the user that committed.",
							Computed:            true,
						},
						"committer_name": schema.StringAttribute{
							MarkdownDescription: "The name of the user that committed.",
							Computed:            true,
						},
						"short_id": schema.StringAttribute{
							MarkdownDescription: "The short id assigned to the commit by Gitlab.",
							Computed:            true,
						},
						"title": schema.StringAttribute{
							MarkdownDescription: "The title of the commit",
							Computed:            true,
						},
						"message": schema.StringAttribute{
							MarkdownDescription: "The commit message",
							Computed:            true,
						},
						"parent_ids": schema.SetAttribute{
							MarkdownDescription: "The id of the parents of the commit",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabProjectTagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectTagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectTagDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	project := data.Project.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("read gitlab tag %s/%s", project, name))
	tag, _, err := d.client.Tags.GetTag(project, name, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project tag: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &name))
	data.Name = types.StringValue(tag.Name)
	data.Project = types.StringValue(project)
	data.Message = types.StringValue(tag.Message)
	data.Protected = types.BoolValue(tag.Protected)
	data.Target = types.StringValue(tag.Target)

	if tag.Release == nil {
		data.Release = []gitlabProjectTagReleaseDataSourceModel{}
	} else {
		data.Release = []gitlabProjectTagReleaseDataSourceModel{{
			TagName:     types.StringValue(tag.Release.TagName),
			Description: types.StringValue(tag.Release.Description),
		}}
	}

	if tag.Commit == nil {
		data.Commit = []gitlabProjectTagCommitDataSourceModel{}
	} else {
		parentIDs, diags := types.SetValueFrom(ctx, types.StringType, tag.Commit.ParentIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Commit = []gitlabProjectTagCommitDataSourceModel{{
			ID:             types.StringValue(tag.Commit.ID),
			ShortID:        types.StringValue(tag.Commit.ShortID),
			Title:          types.StringValue(tag.Commit.Title),
			AuthorName:     types.StringValue(tag.Commit.AuthorName),
			AuthorEmail:    types.StringValue(tag.Commit.AuthorEmail),
			AuthoredDate:   types.StringValue(tag.Commit.AuthoredDate.Format(time.RFC3339)),
			CommittedDate:  types.StringValue(tag.Commit.CommittedDate.Format(time.RFC3339)),
			CommitterEmail: types.StringValue(tag.Commit.CommitterEmail),
			CommitterName:  types.StringValue(tag.Commit.CommitterName),
			Message:        types.StringValue(tag.Commit.Message),
			ParentIDs:      parentIDs,
		}}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
