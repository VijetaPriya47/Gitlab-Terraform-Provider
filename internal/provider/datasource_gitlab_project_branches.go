package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	_ datasource.DataSource              = &gitlabProjectBranchesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectBranchesDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectBranchesDataSource)
}

func NewGitlabProjectBranchesDataSource() datasource.DataSource {
	return &gitlabProjectBranchesDataSource{}
}

type gitlabProjectBranchesDataSource struct {
	client *gitlab.Client
}

type gitlabProjectBranchesDataSourceModel struct {
	ID       types.String                         `tfsdk:"id"`
	Project  types.String                         `tfsdk:"project"`
	Branches []gitlabProjectBranchDataSourceModel `tfsdk:"branches"`
}

type gitlabProjectBranchDataSourceModel struct {
	Name               types.String                        `tfsdk:"name"`
	Merged             types.Bool                          `tfsdk:"merged"`
	Protected          types.Bool                          `tfsdk:"protected"`
	Default            types.Bool                          `tfsdk:"default"`
	DevelopersCanPush  types.Bool                          `tfsdk:"developers_can_push"`
	DevelopersCanMerge types.Bool                          `tfsdk:"developers_can_merge"`
	CanPush            types.Bool                          `tfsdk:"can_push"`
	WebURL             types.String                        `tfsdk:"web_url"`
	Commit             []gitlabBranchCommitDataSourceModel `tfsdk:"commit"`
}

func (d *gitlabProjectBranchesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_branches"
}

func (d *gitlabProjectBranchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_branches`" + ` data source allows details of the branches of a given project to be retrieved.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/branches/#list-repository-branches)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or URL-encoded path of the project owned by the authenticated user.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"branches": schema.ListNestedAttribute{
				MarkdownDescription: "The list of branches of the project, as defined below.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the branch.",
							Computed:            true,
						},
						"merged": schema.BoolAttribute{
							MarkdownDescription: "Bool, true if the branch has been merged into its parent.",
							Computed:            true,
						},
						"protected": schema.BoolAttribute{
							MarkdownDescription: "Bool, true if branch has branch protection.",
							Computed:            true,
						},
						"default": schema.BoolAttribute{
							MarkdownDescription: "Bool, true if branch is the default branch for the project.",
							Computed:            true,
						},
						"developers_can_push": schema.BoolAttribute{
							MarkdownDescription: "Bool, true if developer level access allows git push.",
							Computed:            true,
						},
						"developers_can_merge": schema.BoolAttribute{
							MarkdownDescription: "Bool, true if developer level access allows to merge branch.",
							Computed:            true,
						},
						"can_push": schema.BoolAttribute{
							MarkdownDescription: "Bool, true if you can push to the branch.",
							Computed:            true,
						},
						"web_url": schema.StringAttribute{
							MarkdownDescription: "URL that can be used to find the branch in a browser.",
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
				},
			},
		},
	}
}

func (d *gitlabProjectBranchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectBranchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabProjectBranchesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Gitlab branches")

	project := data.Project.ValueString()

	options := &gitlab.ListBranchesOptions{}

	allBranches, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Branch, *gitlab.Response, error) {
		return d.client.Branches.ListBranches(project, options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project branches: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(project)

	data.Branches = []gitlabProjectBranchDataSourceModel{}
	for _, branch := range allBranches {
		commits := []gitlabBranchCommitDataSourceModel{}
		if branch.Commit != nil {
			parentIDs, diags := types.SetValueFrom(ctx, types.StringType, branch.Commit.ParentIDs)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			commits = append(commits, gitlabBranchCommitDataSourceModel{
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
			})
		}
		modelBranch := gitlabProjectBranchDataSourceModel{
			Name:               types.StringValue(branch.Name),
			Merged:             types.BoolValue(branch.Merged),
			Protected:          types.BoolValue(branch.Protected),
			Default:            types.BoolValue(branch.Default),
			DevelopersCanPush:  types.BoolValue(branch.DevelopersCanPush),
			DevelopersCanMerge: types.BoolValue(branch.DevelopersCanMerge),
			CanPush:            types.BoolValue(branch.CanPush),
			WebURL:             types.StringValue(branch.WebURL),
			Commit:             commits,
		}
		data.Branches = append(data.Branches, modelBranch)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
