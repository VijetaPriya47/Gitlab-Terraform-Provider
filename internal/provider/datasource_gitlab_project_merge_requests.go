package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func init() {
	registerDataSource(NewGitlabProjectMergeRequestsDataSource)
}

// NewGitlabProjectMergeRequestsDataSource creates a default data source instance for project merge requests.
func NewGitlabProjectMergeRequestsDataSource() datasource.DataSource {
	return &gitlabProjectMergeRequestsDataSource{}
}

type gitlabProjectMergeRequestsDataSourceModel struct {
	Project          types.String                                   `tfsdk:"project"`
	IIDs             []types.Int64                                  `tfsdk:"iids"`
	State            types.String                                   `tfsdk:"state"`
	OrderBy          types.String                                   `tfsdk:"order_by"`
	Sort             types.String                                   `tfsdk:"sort"`
	Milestone        types.String                                   `tfsdk:"milestone"`
	CreatedAfter     types.String                                   `tfsdk:"created_after"`
	CreatedBefore    types.String                                   `tfsdk:"created_before"`
	UpdatedAfter     types.String                                   `tfsdk:"updated_after"`
	UpdatedBefore    types.String                                   `tfsdk:"updated_before"`
	Scope            types.String                                   `tfsdk:"scope"`
	AuthorID         types.Int64                                    `tfsdk:"author_id"`
	AuthorUsername   types.String                                   `tfsdk:"author_username"`
	ReviewerUsername types.String                                   `tfsdk:"reviewer_username"`
	MyReactionEmoji  types.String                                   `tfsdk:"my_reaction_emoji"`
	SourceBranch     types.String                                   `tfsdk:"source_branch"`
	TargetBranch     types.String                                   `tfsdk:"target_branch"`
	Search           types.String                                   `tfsdk:"search"`
	WIP              types.String                                   `tfsdk:"wip"`
	MergeRequests    []*gitlabProjectMergeRequestsMRDataSourceModel `tfsdk:"merge_requests"`
}

type gitlabProjectMergeRequestsMRDataSourceModel struct {
	ID                          types.Int32                                    `tfsdk:"id"`
	IID                         types.Int32                                    `tfsdk:"iid"`
	Assignee                    *gitlabProjectMergeRequestUserDataSourceModel  `tfsdk:"assignee"`
	Assignees                   []gitlabProjectMergeRequestUserDataSourceModel `tfsdk:"assignees"`
	Author                      *gitlabProjectMergeRequestUserDataSourceModel  `tfsdk:"author"`
	BlockingDiscussionsResolved types.Bool                                     `tfsdk:"blocking_discussions_resolved"`
	ClosedAt                    types.String                                   `tfsdk:"closed_at"`
	ClosedBy                    *gitlabProjectMergeRequestUserDataSourceModel  `tfsdk:"closed_by"`
	CreatedAt                   types.String                                   `tfsdk:"created_at"`
}

type gitlabProjectMergeRequestsDataSource struct {
	client *gitlab.Client
}

func (d *gitlabProjectMergeRequestsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_merge_requests"
}

func (d *gitlabProjectMergeRequestsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

	mergeRequestAttributes := map[string]schema.Attribute{
		"id": schema.Int32Attribute{
			MarkdownDescription: "The unique instance level ID of the merge request.",
			Computed:            true,
		},
		"iid": schema.Int32Attribute{
			MarkdownDescription: "The unique project level ID of the merge request.",
			Computed:            true,
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
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `
The ` + "`gitlab_project_merge_requests`" + ` data source retrieves
information about a list of merge requests related to a specific project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/merge_requests/#list-project-merge-requests)`,
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or path of the project.",
				Required:            true,
			},
			"iids": schema.ListAttribute{
				MarkdownDescription: "The unique internal IDs of the merge requests.",
				ElementType:         types.Int64Type,
				Optional:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Return all merge requests (all) or just those that are opened, closed, locked, or merged.",
				Optional:            true,
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Return requests ordered by `created_at`, `title` or `updated_at`. Default is `created_at`.",
				Optional:            true,
			},
			"sort": schema.StringAttribute{
				MarkdownDescription: "Return requests sorted in `asc` or `desc` order. Default is `desc`.",
				Optional:            true,
			},
			"milestone": schema.StringAttribute{
				MarkdownDescription: "Return only merge requests for a specific milestone. `None` returns merge requests with no milestone. `Any` returns merge requests that have an assigned milestone.",
				Optional:            true,
			},
			"created_after": schema.StringAttribute{
				MarkdownDescription: "Return merge requests created after the given time. Expected in RFC3339 format (2006-01-02T15:04:05Z).",
				Optional:            true,
			},
			"created_before": schema.StringAttribute{
				MarkdownDescription: "Return merge requests created before the given time. Expected in RFC3339 format (2006-01-02T15:04:05Z).",
				Optional:            true,
			},
			"updated_after": schema.StringAttribute{
				MarkdownDescription: "Return merge requests updated after the given time. Expected in RFC3339 format (2006-01-02T15:04:05Z).",
				Optional:            true,
			},
			"updated_before": schema.StringAttribute{
				MarkdownDescription: "Return merge requests updated before the given time. Expected in RFC3339 format (2006-01-02T15:04:05Z).",
				Optional:            true,
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Return merge requests for the given scope: `created_by_me`, `assigned_to_me`, or `all`.",
				Optional:            true,
			},
			"author_id": schema.Int64Attribute{
				MarkdownDescription: "Return merge requests created by the given user ID.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.ConflictsWith(path.MatchRoot("author_username"))},
			},
			"author_username": schema.StringAttribute{
				MarkdownDescription: "Return merge requests created by the given username.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("author_id"))},
			},
			"reviewer_username": schema.StringAttribute{
				MarkdownDescription: "Return merge requests reviewed by the given username. `None` returns merge requests with no reviews. `Any` returns merge requests with any reviewer.",
				Optional:            true,
			},
			"my_reaction_emoji": schema.StringAttribute{
				MarkdownDescription: "Return merge requests reacted to by the authenticated user with the given emoji. `None` returns issues not given a reaction. `Any` returns issues given at least one reaction.",
				Optional:            true,
			},
			"source_branch": schema.StringAttribute{
				MarkdownDescription: "Return merge requests with the given source branch.",
				Optional:            true,
			},
			"target_branch": schema.StringAttribute{
				MarkdownDescription: "Return merge requests with the given target branch.",
				Optional:            true,
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Search merge requests against their `title` or `description`.",
				Optional:            true,
			},
			"wip": schema.StringAttribute{
				MarkdownDescription: "Filter merge requests against their wip status. `yes` to return only draft merge requests, `no` to return non-draft merge requests.",
				Optional:            true,
			},
			"merge_requests": schema.ListNestedAttribute{
				MarkdownDescription: "The list of merge requests.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: mergeRequestAttributes,
				},
			},
		},
	}
}

func (d *gitlabProjectMergeRequestsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectMergeRequestsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitlabProjectMergeRequestsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.ListProjectMergeRequestsOptions{}
	if len(config.IIDs) > 0 {
		var iids []int
		for _, iid := range config.IIDs {
			iids = append(iids, int(iid.ValueInt64()))
		}
		options.IIDs = &iids
	}
	if !config.State.IsNull() {
		options.State = config.State.ValueStringPointer()
	}
	if !config.OrderBy.IsNull() {
		options.OrderBy = config.OrderBy.ValueStringPointer()
	}
	if !config.Sort.IsNull() {
		options.Sort = config.Sort.ValueStringPointer()
	}
	if !config.Milestone.IsNull() {
		options.Milestone = config.Milestone.ValueStringPointer()
	}
	if !config.CreatedAfter.IsNull() {
		t, err := time.Parse(time.RFC3339, config.CreatedAfter.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid created_after timestamp", err.Error())
			return
		}
		options.CreatedAfter = &t
	}
	if !config.CreatedBefore.IsNull() {
		t, err := time.Parse(time.RFC3339, config.CreatedBefore.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid created_before timestamp", err.Error())
			return
		}
		options.CreatedBefore = &t
	}
	if !config.UpdatedAfter.IsNull() {
		t, err := time.Parse(time.RFC3339, config.UpdatedAfter.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid updated_after timestamp", err.Error())
			return
		}
		options.UpdatedAfter = &t
	}
	if !config.UpdatedBefore.IsNull() {
		t, err := time.Parse(time.RFC3339, config.UpdatedBefore.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid updated_before timestamp", err.Error())
			return
		}
		options.UpdatedBefore = &t
	}
	if !config.Scope.IsNull() {
		options.Scope = config.Scope.ValueStringPointer()
	}
	if !config.AuthorID.IsNull() {
		options.AuthorID = gitlab.Ptr(int(config.AuthorID.ValueInt64()))
	}
	if !config.AuthorUsername.IsNull() {
		options.AuthorUsername = config.AuthorUsername.ValueStringPointer()
	}
	if !config.ReviewerUsername.IsNull() {
		options.ReviewerUsername = config.ReviewerUsername.ValueStringPointer()
	}
	if !config.MyReactionEmoji.IsNull() {
		options.MyReactionEmoji = config.MyReactionEmoji.ValueStringPointer()
	}
	if !config.SourceBranch.IsNull() {
		options.SourceBranch = config.SourceBranch.ValueStringPointer()
	}
	if !config.TargetBranch.IsNull() {
		options.TargetBranch = config.TargetBranch.ValueStringPointer()
	}
	if !config.Search.IsNull() {
		options.Search = config.Search.ValueStringPointer()
	}
	if !config.WIP.IsNull() {
		options.WIP = config.WIP.ValueStringPointer()
	}

	mrs, _, err := d.client.MergeRequests.ListProjectMergeRequests(
		config.Project.ValueString(),
		options,
		gitlab.WithContext(ctx),
	)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf(
			"Unable to read project merge request details: %v", err,
		))
		return
	}

	for _, mr := range mrs {
		config.MergeRequests = append(config.MergeRequests, config.mergeRequestToModel(mr))
	}

	diags := resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

func (d *gitlabProjectMergeRequestsDataSourceModel) mergeRequestToModel(mr *gitlab.BasicMergeRequest) *gitlabProjectMergeRequestsMRDataSourceModel {
	var config gitlabProjectMergeRequestsMRDataSourceModel
	config.ID = types.Int32Value(int32(mr.ID))
	config.IID = types.Int32Value(int32(mr.IID))
	config.Author = generateUserModel(mr.Author)
	config.BlockingDiscussionsResolved = types.BoolValue(mr.BlockingDiscussionsResolved)

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
	return &config
}
