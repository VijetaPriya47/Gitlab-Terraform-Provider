package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupDataSource)
}

func NewGitlabGroupDataSource() datasource.DataSource {
	return &gitlabGroupDataSource{}
}

type gitlabGroupDataSource struct {
	client *gitlab.Client
}

type gitlabGroupDataSourceModel struct {
	ID                                   types.String                                `tfsdk:"id"`
	GroupID                              types.Int64                                 `tfsdk:"group_id"`
	FullPath                             types.String                                `tfsdk:"full_path"`
	Name                                 types.String                                `tfsdk:"name"`
	FullName                             types.String                                `tfsdk:"full_name"`
	WebURL                               types.String                                `tfsdk:"web_url"`
	Path                                 types.String                                `tfsdk:"path"`
	DefaultBranch                        types.String                                `tfsdk:"default_branch"`
	Description                          types.String                                `tfsdk:"description"`
	LFSEnabled                           types.Bool                                  `tfsdk:"lfs_enabled"`
	RequestAccessEnabled                 types.Bool                                  `tfsdk:"request_access_enabled"`
	VisibilityLevel                      types.String                                `tfsdk:"visibility_level"`
	ParentID                             types.Int64                                 `tfsdk:"parent_id"`
	RunnersToken                         types.String                                `tfsdk:"runners_token"`
	DefaultBranchProtection              types.Int64                                 `tfsdk:"default_branch_protection"`
	PreventForkingOutsideGroup           types.Bool                                  `tfsdk:"prevent_forking_outside_group"`
	PreventSharingGroupsOutsideHierarchy types.Bool                                  `tfsdk:"prevent_sharing_groups_outside_hierarchy"`
	MembershipLock                       types.Bool                                  `tfsdk:"membership_lock"`
	ExtraSharedRunnersMinutesLimit       types.Int64                                 `tfsdk:"extra_shared_runners_minutes_limit"`
	SharedRunnersMinutesLimit            types.Int64                                 `tfsdk:"shared_runners_minutes_limit"`
	WikiAccessLevel                      types.String                                `tfsdk:"wiki_access_level"`
	SharedRunnersSetting                 types.String                                `tfsdk:"shared_runners_setting"`
	SharedWithGroups                     []gitlabGroupSharedWithGroupDataSourceModel `tfsdk:"shared_with_groups"`
}

type gitlabGroupSharedWithGroupDataSourceModel struct {
	GroupID          types.Int64  `tfsdk:"group_id"`
	GroupName        types.String `tfsdk:"group_name"`
	GroupFullPath    types.String `tfsdk:"group_full_path"`
	GroupAccessLevel types.Int64  `tfsdk:"group_access_level"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
}

var validWikiAccessLevels = []string{
	"disabled",
	"private",
	"enabled",
}

var validSharedRunnersSettings = []string{
	"enabled",
	"disabled_and_overridable",
	"disabled_and_unoverridable",

	// Deprecated
	"disabled_with_override",
}

func (d *gitlabGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *gitlabGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group`" + ` data source allows details of a group to be retrieved by its id or full path.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/groups/#get-a-single-group)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<group-id>`.",
				Computed:            true,
			},
			"group_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the group.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.Int64{int64validator.ConflictsWith(path.MatchRoot("full_path"))},
			},
			"full_path": schema.StringAttribute{
				MarkdownDescription: "The full path of the group.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("group_id"))},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of this group.",
				Computed:            true,
			},
			"full_name": schema.StringAttribute{
				MarkdownDescription: "The full name of the group.",
				Computed:            true,
			},
			"web_url": schema.StringAttribute{
				MarkdownDescription: "Web URL of the group.",
				Computed:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The path of the group.",
				Computed:            true,
			},
			"default_branch": schema.StringAttribute{
				MarkdownDescription: "The default branch of the group.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the group.",
				Computed:            true,
			},
			"lfs_enabled": schema.BoolAttribute{
				MarkdownDescription: "Boolean, is LFS enabled for projects in this group.",
				Computed:            true,
			},
			"request_access_enabled": schema.BoolAttribute{
				MarkdownDescription: "Boolean, is request for access enabled to the group.",
				Computed:            true,
			},
			"visibility_level": schema.StringAttribute{
				MarkdownDescription: "Visibility level of the group. Possible values are `private`, `internal`, `public`.",
				Computed:            true,
			},
			"parent_id": schema.Int64Attribute{
				MarkdownDescription: "Integer, ID of the parent group.",
				Computed:            true,
			},
			"runners_token": schema.StringAttribute{
				MarkdownDescription: "The group level registration token to use during runner setup.",
				Computed:            true,
				Sensitive:           true,
			},
			"default_branch_protection": schema.Int64Attribute{
				MarkdownDescription: "Whether developers and maintainers can push to the applicable default branch.",
				Computed:            true,
			},
			"prevent_forking_outside_group": schema.BoolAttribute{
				MarkdownDescription: "When enabled, users can not fork projects from this group to external namespaces.",
				Computed:            true,
			},
			"prevent_sharing_groups_outside_hierarchy": schema.BoolAttribute{
				MarkdownDescription: "When enabled, users cannot invite other groups outside of the top-level group’s hierarchy. This option is only available for top-level groups.",
				Computed:            true,
			},
			"membership_lock": schema.BoolAttribute{
				MarkdownDescription: "Users cannot be added to projects in this group.",
				Computed:            true,
			},
			"extra_shared_runners_minutes_limit": schema.Int64Attribute{
				MarkdownDescription: "Can be set by administrators only. Additional CI/CD minutes for this group.",
				Computed:            true,
			},
			"shared_runners_minutes_limit": schema.Int64Attribute{
				MarkdownDescription: "Can be set by administrators only. Maximum number of monthly CI/CD minutes for this group. Can be nil (default; inherit system default), 0 (unlimited), or > 0.",
				Computed:            true,
			},
			"wiki_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The group's wiki access level. Only available on Premium and Ultimate plans. Valid values are %s.", utils.RenderValueListForDocs(validWikiAccessLevels)),
				Computed:            true,
			},
			"shared_runners_setting": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Enable or disable shared runners for a group's subgroups and projects. Valid values are: %s.", utils.RenderValueListForDocs(validSharedRunnersSettings)),
				Computed:            true,
			},
			"shared_with_groups": schema.ListNestedAttribute{
				MarkdownDescription: "Describes groups which have access shared to this group.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the group shared with.",
							Computed:            true,
						},
						"group_name": schema.StringAttribute{
							MarkdownDescription: "The name of the group shared with.",
							Computed:            true,
						},
						"group_full_path": schema.StringAttribute{
							MarkdownDescription: "The full path of the group shared with.",
							Computed:            true,
						},
						"group_access_level": schema.Int64Attribute{
							MarkdownDescription: "The access_level permission level of the shared group.",
							Computed:            true,
						},
						"expires_at": schema.StringAttribute{
							MarkdownDescription: "Share with group expiration date.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group *gitlab.Group
	var err error

	tflog.Info(ctx, "Reading Gitlab group")

	var groupLookup string

	if !data.GroupID.IsNull() && !data.GroupID.IsUnknown() {
		groupLookup = fmt.Sprintf("%d", data.GroupID.ValueInt64())
	} else if !data.FullPath.IsNull() && !data.FullPath.IsUnknown() {
		groupLookup = data.FullPath.ValueString()
	} else {
		resp.Diagnostics.AddError("Invalid attribute combination", "one and only one of group_id or full_path must be set")
		return
	}

	group, _, err = d.client.Groups.GetGroup(groupLookup, &gitlab.GetGroupOptions{WithProjects: gitlab.Ptr(false)}, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get group %s: %s", groupLookup, err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", group.ID))
	data.GroupID = types.Int64Value(int64(group.ID))
	data.FullPath = types.StringValue(group.FullPath)
	data.Name = types.StringValue(group.Name)
	data.FullName = types.StringValue(group.FullName)
	data.WebURL = types.StringValue(group.WebURL)
	data.Path = types.StringValue(group.Path)
	data.DefaultBranch = types.StringValue(group.DefaultBranch)
	data.Description = types.StringValue(group.Description)
	data.LFSEnabled = types.BoolValue(group.LFSEnabled)
	data.RequestAccessEnabled = types.BoolValue(group.RequestAccessEnabled)
	data.VisibilityLevel = types.StringValue(string(group.Visibility))
	data.ParentID = types.Int64Value(int64(group.ParentID))
	data.RunnersToken = types.StringValue(group.RunnersToken)
	data.PreventForkingOutsideGroup = types.BoolValue(group.PreventForkingOutsideGroup)
	data.MembershipLock = types.BoolValue(group.MembershipLock)
	data.ExtraSharedRunnersMinutesLimit = types.Int64Value(int64(group.ExtraSharedRunnersMinutesLimit))
	data.SharedRunnersMinutesLimit = types.Int64Value(int64(group.SharedRunnersMinutesLimit))
	data.WikiAccessLevel = types.StringValue(string(group.WikiAccessLevel))
	data.SharedRunnersSetting = types.StringValue(string(group.SharedRunnersSetting))

	data.SharedWithGroups = []gitlabGroupSharedWithGroupDataSourceModel{}
	for _, sharedGroup := range group.SharedWithGroups {
		modelSharedGroup := gitlabGroupSharedWithGroupDataSourceModel{
			GroupID:          types.Int64Value(int64(sharedGroup.GroupID)),
			GroupName:        types.StringValue(sharedGroup.GroupName),
			GroupFullPath:    types.StringValue(sharedGroup.GroupFullPath),
			GroupAccessLevel: types.Int64Value(int64(sharedGroup.GroupAccessLevel)),
		}

		if sharedGroup.ExpiresAt != nil {
			modelSharedGroup.ExpiresAt = types.StringValue(sharedGroup.ExpiresAt.String())
		} else {
			modelSharedGroup.ExpiresAt = types.StringNull()
		}
		data.SharedWithGroups = append(data.SharedWithGroups, modelSharedGroup)
	}

	// nolint:staticcheck // SA1019 ignore deprecated DefaultBranchProtection
	data.DefaultBranchProtection = types.Int64Value(int64(group.DefaultBranchProtection))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
