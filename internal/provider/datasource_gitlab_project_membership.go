package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ datasource.DataSource              = &gitlabProjectMembershipDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectMembershipDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectMembershipDataSource)
}

func NewGitlabProjectMembershipDataSource() datasource.DataSource {
	return &gitlabProjectMembershipDataSource{}
}

type gitlabProjectMembershipDataSource struct {
	client *gitlab.Client
}

type gitlabProjectMembershipDataSourceModel struct {
	ID        types.String                                   `tfsdk:"id"`
	Project   types.String                                   `tfsdk:"project"`
	ProjectID types.Int64                                    `tfsdk:"project_id"`
	FullPath  types.String                                   `tfsdk:"full_path"`
	Query     types.String                                   `tfsdk:"query"`
	UserIDs   types.Set                                      `tfsdk:"user_ids"`
	Inherited types.Bool                                     `tfsdk:"inherited"`
	Members   []gitlabProjectMembershipMemberDataSourceModel `tfsdk:"members"`
}

type gitlabProjectMembershipMemberDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	Name        types.String `tfsdk:"name"`
	State       types.String `tfsdk:"state"`
	AvatarURL   types.String `tfsdk:"avatar_url"`
	WebURL      types.String `tfsdk:"web_url"`
	AccessLevel types.String `tfsdk:"access_level"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
}

func (d *gitlabProjectMembershipDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_membership"
}

func (d *gitlabProjectMembershipDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_membership`" + ` data source allows you to list and filter all members of a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_members/#list-all-members-of-a-project)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:query-hash>` if query is set, otherwise `<project>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.String{stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("project_id"), path.MatchRelative().AtParent().AtName("full_path"))},
			},
			"project_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the project. Use `project` instead. Will be removed in 19.0.",
				DeprecationMessage:  "Use `project` instead. Will be removed in 19.0.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.Int64{int64validator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("project"), path.MatchRelative().AtParent().AtName("full_path"))},
			},
			"full_path": schema.StringAttribute{
				MarkdownDescription: "The full path of the project. Use `project` instead. Will be removed in 19.0.",
				DeprecationMessage:  "Use `project` instead. Will be removed in 19.0.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.String{stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("project"), path.MatchRelative().AtParent().AtName("project_id"))},
			},
			"query": schema.StringAttribute{
				MarkdownDescription: "A query string to search for members",
				Optional:            true,
			},
			"user_ids": schema.SetAttribute{
				MarkdownDescription: "List of user ids to filter members by",
				Optional:            true,
				ElementType:         types.Int64Type,
			},
			"inherited": schema.BoolAttribute{
				MarkdownDescription: "Return all project members including members through ancestor groups",
				Optional:            true,
			},
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "The list of project members.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The unique id assigned to the user by the gitlab server.",
							Computed:            true,
						},
						"username": schema.StringAttribute{
							MarkdownDescription: "The username of the user.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the user.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: "Whether the user is active or blocked.",
							Computed:            true,
						},
						"avatar_url": schema.StringAttribute{
							MarkdownDescription: "The avatar URL of the user.",
							Computed:            true,
						},
						"web_url": schema.StringAttribute{
							MarkdownDescription: "User's website URL.",
							Computed:            true,
						},
						"access_level": schema.StringAttribute{
							MarkdownDescription: "The level of access to the group.",
							Computed:            true,
						},
						"expires_at": schema.StringAttribute{
							MarkdownDescription: "Expiration date for the group membership.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabProjectMembershipDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectMembershipDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectMembershipDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var project *gitlab.Project
	var err error

	tflog.Info(ctx, "Reading Gitlab project")

	var pid any
	if !data.Project.IsNull() && !data.Project.IsUnknown() {
		pid = data.Project.ValueString()
	} else if !data.ProjectID.IsNull() && !data.ProjectID.IsUnknown() {
		pid = int(data.ProjectID.ValueInt64())
	} else if !data.FullPath.IsNull() && !data.FullPath.IsUnknown() {
		pid = data.FullPath.ValueString()
	} else {
		resp.Diagnostics.AddError("exactly one of project, project_id, or full_path must be set.", "This is a provider bug, please report upstream at https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues")
		return
	}

	// Get project to have both, the `project_id` and `full_path` for setting the state. Remove in 19.0
	project, _, err = d.client.Projects.GetProject(pid, nil)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project: %s", err.Error()))
		return
	}

	tflog.Info(ctx, "Reading Gitlab project memberships")

	// Get project memberships
	listOptions := &gitlab.ListProjectMembersOptions{}

	if !data.Query.IsNull() && !data.Query.IsUnknown() {
		listOptions.Query = data.Query.ValueStringPointer()
	}

	if !data.UserIDs.IsNull() && !data.UserIDs.IsUnknown() {
		var userIDs []int64
		data.UserIDs.ElementsAs(ctx, &userIDs, true)
		listOptions.UserIDs = &userIDs
	}

	listMembers := d.client.ProjectMembers.ListProjectMembers
	if !data.Inherited.IsNull() && !data.Inherited.IsUnknown() && data.Inherited.ValueBool() {
		listMembers = d.client.ProjectMembers.ListAllProjectMembers
	}

	allPMs, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.ProjectMember, *gitlab.Response, error) {
		return listMembers(project.ID, listOptions, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project members: %s", err.Error()))
		return
	}

	data.modelToStateModel(project, allPMs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (data *gitlabProjectMembershipDataSourceModel) modelToStateModel(project *gitlab.Project, allPMs []*gitlab.ProjectMember) {
	var optionsHash strings.Builder
	optionsHash.WriteString(strconv.FormatInt(project.ID, 10))

	if !data.Query.IsNull() && !data.Query.IsUnknown() {
		optionsHash.WriteString(data.Query.ValueString())
	}
	hasher := sha256.New()
	hasher.Write([]byte(optionsHash.String()))
	id := hex.EncodeToString(hasher.Sum(nil))
	data.ID = types.StringValue(id)
	data.ProjectID = types.Int64Value(int64(project.ID))
	data.FullPath = types.StringValue(project.PathWithNamespace)

	if data.Project.IsNull() || data.Project.IsUnknown() {
		data.Project = types.StringValue(fmt.Sprintf("%d", data.ProjectID.ValueInt64()))
	}

	data.Members = []gitlabProjectMembershipMemberDataSourceModel{}
	for _, member := range allPMs {
		modelMember := gitlabProjectMembershipMemberDataSourceModel{
			ID:          types.Int64Value(int64(member.ID)),
			Username:    types.StringValue(member.Username),
			Name:        types.StringValue(member.Name),
			State:       types.StringValue(member.State),
			AvatarURL:   types.StringValue(member.AvatarURL),
			WebURL:      types.StringValue(member.WebURL),
			AccessLevel: types.StringValue(api.AccessLevelValueToName[gitlab.AccessLevelValue(member.AccessLevel)]),
		}
		if member.ExpiresAt == nil {
			modelMember.ExpiresAt = types.StringNull()
		} else {
			modelMember.ExpiresAt = types.StringValue(member.ExpiresAt.String())
		}
		data.Members = append(data.Members, modelMember)
	}
}
