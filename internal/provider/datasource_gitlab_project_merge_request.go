package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/gitlab-org/api/client-go"
)

func init() {
	registerDataSource(NewGitlabProjectMergeRequestDataSource)
}

// NewGitlabProjectMergeRequestDataSource creates a default data source instance for project merge requests.
func NewGitlabProjectMergeRequestDataSource() datasource.DataSource {
	return &gitlabProjectMergeRequestDataSource{}
}

type gitlabProjectMergeRequestUserDataSourceModel struct {
	ID        types.Int32  `tfsdk:"id"`
	AvatarURL types.String `tfsdk:"avatar_url"`
	Name      types.String `tfsdk:"name"`
	State     types.String `tfsdk:"state"`
	Username  types.String `tfsdk:"username"`
	WebURL    types.String `tfsdk:"web_url"`
}

type gitlabProjectMergeRequestDataSourceModel struct {
	ID                          types.Int32                                    `tfsdk:"id"`
	IID                         types.Int32                                    `tfsdk:"iid"`
	Project                     types.String                                   `tfsdk:"project"`
	Assignee                    *gitlabProjectMergeRequestUserDataSourceModel  `tfsdk:"assignee"`
	Assignees                   []gitlabProjectMergeRequestUserDataSourceModel `tfsdk:"assignees"`
	Author                      *gitlabProjectMergeRequestUserDataSourceModel  `tfsdk:"author"`
	BlockingDiscussionsResolved types.Bool                                     `tfsdk:"blocking_discussions_resolved"`
	ChangesCount                types.String                                   `tfsdk:"changes_count"`
	ClosedAt                    types.String                                   `tfsdk:"closed_at"`
	ClosedBy                    *gitlabProjectMergeRequestUserDataSourceModel  `tfsdk:"closed_by"`
	CreatedAt                   types.String                                   `tfsdk:"created_at"`
}

type gitlabProjectMergeRequestDataSource struct {
	client *gitlab.Client
}

func (d *gitlabProjectMergeRequestDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_merge_request"
}

func (d *gitlabProjectMergeRequestDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	userAttributes := map[string]schema.Attribute{
		"id": schema.Int32Attribute{
			MarkdownDescription: "The internal ID number of the user.",
			Computed:            true,
		},
		"avatar_url": schema.StringAttribute{
			MarkdownDescription: "A link to the user's avatar image.",
			Computed:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the user.",
			Computed:            true,
		},
		"state": schema.StringAttribute{
			MarkdownDescription: "The state of the user account.",
			Computed:            true,
		},
		"username": schema.StringAttribute{
			MarkdownDescription: "The username of the user.",
			Computed:            true,
		},
		"web_url": schema.StringAttribute{
			MarkdownDescription: "A link to the user's profile page.",
			Computed:            true,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `
The ` + "`gitlab_project_merge_request`" + ` data source retrieves
information about a single merge request related to a specific project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/merge_requests.html#get-single-mr)
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				MarkdownDescription: "The unique instance level ID of the merge request.",
				Computed:            true,
			},
			"iid": schema.Int32Attribute{
				MarkdownDescription: "The unique project level ID of the merge request.",
				Required:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or path of the project.",
				Required:            true,
			},
			"assignee": schema.SingleNestedAttribute{
				MarkdownDescription: "First assignee of the merge request.",
				Computed:            true,
				Attributes:          userAttributes,
			},
			"assignees": schema.ListNestedAttribute{
				MarkdownDescription: "Assignees of the merge request.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: userAttributes,
				},
			},
			"author": schema.SingleNestedAttribute{
				MarkdownDescription: "User who created this merge request.",
				Computed:            true,
				Attributes:          userAttributes,
			},
			"blocking_discussions_resolved": schema.BoolAttribute{
				MarkdownDescription: `
Indicates if all discussions are resolved only if all are
required before merge request can be merged.
				`,
				Computed: true,
			},
			"changes_count": schema.StringAttribute{
				MarkdownDescription: `
Number of changes made on the merge request. Empty when the
merge request is created, and populates asynchronously.
				`,
				Computed: true,
			},
			"closed_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the merge request was closed.",
				Computed:            true,
			},
			"closed_by": schema.SingleNestedAttribute{
				MarkdownDescription: "User who closed this merge request.",
				Computed:            true,
				Attributes:          userAttributes,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the merge request was created.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabProjectMergeRequestDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectMergeRequestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitlabProjectMergeRequestDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mr, _, err := d.client.MergeRequests.GetMergeRequest(
		config.Project.ValueString(),
		int(config.IID.ValueInt32()),
		&gitlab.GetMergeRequestsOptions{},
	)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf(
			"Unable to read project merge request details: %v", err,
		))
		return
	}

	config.ID = types.Int32Value(int32(mr.ID))
	config.Author = generateUserModel(mr.Author)
	config.BlockingDiscussionsResolved = types.BoolValue(mr.BlockingDiscussionsResolved)
	config.ChangesCount = types.StringValue(mr.ChangesCount)

	if mr.Assignee != nil {
		config.Assignee = generateUserModel(mr.Assignee)
	}
	if mr.Assignees != nil {
		config.Assignees = make([]gitlabProjectMergeRequestUserDataSourceModel, 0)
		for _, assignee := range mr.Assignees {
			config.Assignees = append(config.Assignees, *generateUserModel(assignee))
		}
	}
	if mr.ClosedAt != nil {
		config.ClosedAt = types.StringValue(mr.ClosedAt.Format(time.RFC3339))
	}
	if mr.ClosedBy != nil {
		config.ClosedBy = generateUserModel(mr.ClosedBy)
	}
	if mr.CreatedAt != nil {
		config.CreatedAt = types.StringValue(mr.CreatedAt.Format(time.RFC3339))
	}

	diags := resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

func generateUserModel(user *gitlab.BasicUser) *gitlabProjectMergeRequestUserDataSourceModel {
	return &gitlabProjectMergeRequestUserDataSourceModel{
		ID:        types.Int32Value(int32(user.ID)),
		AvatarURL: types.StringValue(user.AvatarURL),
		Name:      types.StringValue(user.Name),
		State:     types.StringValue(user.State),
		Username:  types.StringValue(user.Username),
		WebURL:    types.StringValue(user.WebURL),
	}
}
