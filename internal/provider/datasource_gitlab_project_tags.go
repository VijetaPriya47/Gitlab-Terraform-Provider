package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mitchellh/hashstructure/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	_ datasource.DataSource              = &gitlabProjectTagsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectTagsDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectTagsDataSource)
}

func NewGitlabProjectTagsDataSource() datasource.DataSource {
	return &gitlabProjectTagsDataSource{}
}

type gitlabProjectTagsDataSource struct {
	client *gitlab.Client
}

type gitlabProjectTagsDataSourceModel struct {
	ID      types.String                                    `tfsdk:"id"`
	Project types.String                                    `tfsdk:"project"`
	OrderBy types.String                                    `tfsdk:"order_by"`
	Sort    types.String                                    `tfsdk:"sort"`
	Search  types.String                                    `tfsdk:"search"`
	Tags    []gitlabProjectTagsIndividualTagDataSourceModel `tfsdk:"tags"`
}

type gitlabProjectTagsIndividualTagDataSourceModel struct {
	Name      types.String                             `tfsdk:"name"`
	Message   types.String                             `tfsdk:"message"`
	Protected types.Bool                               `tfsdk:"protected"`
	Target    types.String                             `tfsdk:"target"`
	Release   []gitlabProjectTagReleaseDataSourceModel `tfsdk:"release"`
	Commit    []gitlabProjectTagCommitDataSourceModel  `tfsdk:"commit"`
}

func (d *gitlabProjectTagsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_tags"
}

func (d *gitlabProjectTagsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_tags`" + ` data source allows details of project tags to be retrieved by some search criteria.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/tags/#list-project-repository-tags)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:hash-of-other-options>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project owned by the authenticated user.",
				Required:            true,
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Return tags ordered by `name` or `updated` fields. Default is `updated`.",
				Optional:            true,
			},
			"sort": schema.StringAttribute{
				MarkdownDescription: "Return tags sorted in `asc` or `desc` order. Default is `desc`.",
				Optional:            true,
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Return list of tags matching the search criteria. You can use `^term` and `term$` to find tags that begin and end with `term` respectively. No other regular expressions are supported.",
				Optional:            true,
			},
			"tags": schema.ListNestedAttribute{
				MarkdownDescription: "List of repository tags from a project.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of a tag.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *gitlabProjectTagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectTagsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	options := gitlab.ListTagsOptions{}

	if !data.OrderBy.IsNull() && !data.OrderBy.IsUnknown() {
		options.OrderBy = data.OrderBy.ValueStringPointer()
	}

	if !data.Sort.IsNull() && !data.Sort.IsUnknown() {
		options.Sort = data.Sort.ValueStringPointer()
	}

	if !data.Search.IsNull() && !data.Search.IsUnknown() {
		options.Search = data.Search.ValueStringPointer()
	}

	optionsHash, err := hashstructure.Hash(&options, hashstructure.FormatV1, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate options hash", err.Error())
		return
	}

	tags, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Tag, *gitlab.Response, error) {
		return d.client.Tags.ListTags(project, &options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project tags: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d", project, optionsHash))
	diags := data.modelToStateModel(ctx, project, tags)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabProjectTagsDataSourceModel) modelToStateModel(ctx context.Context, project string, tags []*gitlab.Tag) diag.Diagnostics {
	d.Project = types.StringValue(project)
	d.Tags = []gitlabProjectTagsIndividualTagDataSourceModel{}

	for _, tag := range tags {
		modelTag := gitlabProjectTagsIndividualTagDataSourceModel{
			Name:      types.StringValue(tag.Name),
			Message:   types.StringValue(tag.Message),
			Protected: types.BoolValue(tag.Protected),
			Target:    types.StringValue(tag.Target),
			Release:   []gitlabProjectTagReleaseDataSourceModel{},
			Commit:    []gitlabProjectTagCommitDataSourceModel{},
		}

		if tag.Release != nil {
			modelTag.Release = append(modelTag.Release, gitlabProjectTagReleaseDataSourceModel{
				TagName:     types.StringValue(tag.Release.TagName),
				Description: types.StringValue(tag.Release.Description),
			})
		}

		if tag.Commit != nil {
			parentIDs, diags := types.SetValueFrom(ctx, types.StringType, tag.Commit.ParentIDs)
			if diags.HasError() {
				return diags
			}
			modelTag.Commit = append(modelTag.Commit, gitlabProjectTagCommitDataSourceModel{
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
			})
		}

		d.Tags = append(d.Tags, modelTag)
	}

	return nil
}
