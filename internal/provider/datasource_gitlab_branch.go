package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabBranchDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabBranchDataSource{}
)

func init() {
	registerDataSource(NewGitlabBranchDataSource)
}

func NewGitlabBranchDataSource() datasource.DataSource {
	return &gitlabBranchDataSource{}
}

type gitlabBranchDataSource struct {
	client *gitlab.Client
}

type gitlabBranchDataSourceModel struct {
	ID                types.String                        `tfsdk:"id"`
	Name              types.String                        `tfsdk:"name"`
	Project           types.String                        `tfsdk:"project"`
	WebURL            types.String                        `tfsdk:"web_url"`
	Default           types.Bool                          `tfsdk:"default"`
	CanPush           types.Bool                          `tfsdk:"can_push"`
	Protected         types.Bool                          `tfsdk:"protected"`
	Merged            types.Bool                          `tfsdk:"merged"`
	DeveloperCanMerge types.Bool                          `tfsdk:"developer_can_merge"`
	DeveloperCanPush  types.Bool                          `tfsdk:"developer_can_push"`
	Commit            []gitlabBranchCommitDataSourceModel `tfsdk:"commit"`
}

type gitlabBranchCommitDataSourceModel struct {
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

func (d *gitlabBranchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (d *gitlabBranchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_branch`" + ` data source allows details of a repository branch to be retrieved by its name and project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/branches/#get-single-repository-branch)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:name>`.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the branch.",
				Required:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The full path or id of the project.",
				Required:            true,
			},
			"web_url": schema.StringAttribute{
				MarkdownDescription: "The url of the created branch (https.)",
				Computed:            true,
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if branch is the default branch for the project.",
				Computed:            true,
			},
			"can_push": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if you can push to the branch.",
				Computed:            true,
			},
			"protected": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if branch has branch protection.",
				Computed:            true,
			},
			"merged": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if the branch has been merged into its parent.",
				Computed:            true,
			},
			"developer_can_merge": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if developer level access allows to merge branch.",
				Computed:            true,
			},
			"developer_can_push": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if developer level access allows git push.",
				Computed:            true,
			},
			"commit": schema.SetNestedAttribute{
				MarkdownDescription: "The commit associated with the branch ref.",
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

func (d *gitlabBranchDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabBranchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabBranchDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := data.Name.ValueString()
	project := data.Project.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("read gitlab branch %s", name))
	branch, _, err := d.client.Branches.GetBranch(project, name, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read branch: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &name))
	data.Name = types.StringValue(branch.Name)
	data.Project = types.StringValue(project)
	data.WebURL = types.StringValue(branch.WebURL)
	data.Default = types.BoolValue(branch.Default)
	data.CanPush = types.BoolValue(branch.CanPush)
	data.Protected = types.BoolValue(branch.Protected)
	data.Merged = types.BoolValue(branch.Merged)
	data.DeveloperCanMerge = types.BoolValue(branch.DevelopersCanMerge)
	data.DeveloperCanPush = types.BoolValue(branch.DevelopersCanPush)
	parentIDs, diags := types.SetValueFrom(ctx, types.StringType, branch.Commit.ParentIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Commit = []gitlabBranchCommitDataSourceModel{{
		ID:             types.StringValue(branch.Commit.ID),
		AuthorEmail:    types.StringValue(branch.Commit.AuthorEmail),
		AuthorName:     types.StringValue(branch.Commit.AuthorName),
		AuthoredDate:   types.StringValue(branch.Commit.AuthoredDate.Format(time.RFC3339)),
		CommittedDate:  types.StringValue(branch.Commit.CommittedDate.Format(time.RFC3339)),
		CommitterEmail: types.StringValue(branch.Commit.CommitterEmail),
		CommitterName:  types.StringValue(branch.Commit.CommitterName),
		ShortID:        types.StringValue(branch.Commit.ShortID),
		Title:          types.StringValue(branch.Commit.Title),
		Message:        types.StringValue(branch.Commit.Message),
		ParentIDs:      parentIDs,
	}}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
