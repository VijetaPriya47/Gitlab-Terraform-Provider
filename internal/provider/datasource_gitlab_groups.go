package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabGroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupsDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupsDataSource)
}

func NewGitlabGroupsDataSource() datasource.DataSource {
	return &gitlabGroupsDataSource{}
}

type gitlabGroupsDataSource struct {
	client *gitlab.Client
}

type gitlabGroupsDataSourceModel struct {
	ID           types.String                            `tfsdk:"id"`
	OrderBy      types.String                            `tfsdk:"order_by"`
	Sort         types.String                            `tfsdk:"sort"`
	Search       types.String                            `tfsdk:"search"`
	TopLevelOnly types.Bool                              `tfsdk:"top_level_only"`
	Groups       []gitlabGroupsIndividualDataSourceModel `tfsdk:"groups"`
}

type gitlabGroupsIndividualDataSourceModel struct {
	GroupID                              types.Int64  `tfsdk:"group_id"`
	FullPath                             types.String `tfsdk:"full_path"`
	Name                                 types.String `tfsdk:"name"`
	FullName                             types.String `tfsdk:"full_name"`
	WebURL                               types.String `tfsdk:"web_url"`
	Path                                 types.String `tfsdk:"path"`
	Description                          types.String `tfsdk:"description"`
	LFSEnabled                           types.Bool   `tfsdk:"lfs_enabled"`
	RequestAccessEnabled                 types.Bool   `tfsdk:"request_access_enabled"`
	VisibilityLevel                      types.String `tfsdk:"visibility_level"`
	ParentID                             types.Int64  `tfsdk:"parent_id"`
	RunnersToken                         types.String `tfsdk:"runners_token"`
	DefaultBranchProtection              types.Int64  `tfsdk:"default_branch_protection"`
	PreventForkingOutsideGroup           types.Bool   `tfsdk:"prevent_forking_outside_group"`
	PreventSharingGroupsOutsideHierarchy types.Bool   `tfsdk:"prevent_sharing_groups_outside_hierarchy"`
	WikiAccessLevel                      types.String `tfsdk:"wiki_access_level"`
	SharedRunnersSetting                 types.String `tfsdk:"shared_runners_setting"`
}

func (d *gitlabGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_groups"
}

func (d *gitlabGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_groups`" + ` data source allows details of multiple groups to be retrieved given some optional filter criteria.

-> Some attributes might not be returned depending on if you're an admin or not.

-> Some available options require administrator privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/groups/#list-groups)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format of a hash of the provided search attributes.",
				Computed:            true,
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Order the groups' list by `id`, `name`, `path`, or `similarity`. (Requires administrator privileges)",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("id", "name", "path", "similarity")},
			},
			"sort": schema.StringAttribute{
				MarkdownDescription: "Sort groups' list in asc or desc order. (Requires administrator privileges)",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("desc", "asc")},
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Search groups by name or path.",
				Optional:            true,
			},
			"top_level_only": schema.BoolAttribute{
				MarkdownDescription: "Limit to top level groups, excluding all subgroups.",
				Optional:            true,
			},
			"groups": schema.ListNestedAttribute{
				MarkdownDescription: "The list of groups.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the group.",
							Computed:            true,
						},
						"full_path": schema.StringAttribute{
							MarkdownDescription: "The full path of the group.",
							Computed:            true,
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
						"description": schema.StringAttribute{
							MarkdownDescription: "The description of the group.",
							Computed:            true,
						},
						"lfs_enabled": schema.BoolAttribute{
							MarkdownDescription: "Is LFS enabled for projects in this group.",
							Computed:            true,
						},
						"request_access_enabled": schema.BoolAttribute{
							MarkdownDescription: "Is request for access enabled to the group.",
							Computed:            true,
						},
						"visibility_level": schema.StringAttribute{
							MarkdownDescription: "Visibility level of the group. Possible values are `private`, `internal`, `public`.",
							Computed:            true,
						},
						"parent_id": schema.Int64Attribute{
							MarkdownDescription: "ID of the parent group.",
							Computed:            true,
						},
						"runners_token": schema.StringAttribute{
							MarkdownDescription: "The group level registration token to use during runner setup.",
							Computed:            true,
							Sensitive:           true,
						},
						"default_branch_protection": schema.Int64Attribute{
							MarkdownDescription: "Whether developers and maintainers can push to the applicable default branch. Will be removed in 19.0.",
							Computed:            true,
							DeprecationMessage:  "Will be removed in 19.0.",
						},
						"prevent_forking_outside_group": schema.BoolAttribute{
							MarkdownDescription: "When enabled, users can not fork projects from this group to external namespaces.",
							Computed:            true,
						},
						"prevent_sharing_groups_outside_hierarchy": schema.BoolAttribute{
							MarkdownDescription: "When enabled, users cannot invite other groups outside of the top-level group’s hierarchy. This option is only available for top-level groups.",
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
					},
				},
			},
		},
	}
}

func (d *gitlabGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabGroupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	listGroupsOptions, id := expandGitlabGroupsOptions(data)
	groups, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
		return d.client.Groups.ListGroups(listGroupsOptions, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read groups: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(id)

	data.Groups = []gitlabGroupsIndividualDataSourceModel{}
	for _, group := range groups {
		modelGroup := gitlabGroupsIndividualDataSourceModel{
			GroupID:                    types.Int64Value(int64(group.ID)),
			FullPath:                   types.StringValue(group.FullPath),
			Name:                       types.StringValue(group.Name),
			FullName:                   types.StringValue(group.FullName),
			WebURL:                     types.StringValue(group.WebURL),
			Path:                       types.StringValue(group.Path),
			Description:                types.StringValue(group.Description),
			LFSEnabled:                 types.BoolValue(group.LFSEnabled),
			RequestAccessEnabled:       types.BoolValue(group.RequestAccessEnabled),
			VisibilityLevel:            types.StringValue(string(group.Visibility)),
			ParentID:                   types.Int64Value(int64(group.ParentID)),
			RunnersToken:               types.StringValue(group.RunnersToken),
			PreventForkingOutsideGroup: types.BoolValue(group.PreventForkingOutsideGroup),
			WikiAccessLevel:            types.StringValue(string(group.WikiAccessLevel)),
			SharedRunnersSetting:       types.StringValue(string(group.SharedRunnersSetting)),

			// nolint:staticcheck // SA1019 ignore deprecated DefaultBranchProtection
			DefaultBranchProtection: types.Int64Value(int64(group.DefaultBranchProtection)),
		}
		data.Groups = append(data.Groups, modelGroup)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func expandGitlabGroupsOptions(data gitlabGroupsDataSourceModel) (*gitlab.ListGroupsOptions, string) {
	listGroupsOptions := &gitlab.ListGroupsOptions{}
	var optionsHash strings.Builder

	var orderBy string
	if !data.OrderBy.IsNull() && !data.OrderBy.IsUnknown() {
		orderBy = data.OrderBy.ValueString()
	} else {
		orderBy = "name"
	}
	listGroupsOptions.OrderBy = &orderBy
	optionsHash.WriteString(orderBy)

	optionsHash.WriteString(",")
	var sort string
	if !data.Sort.IsNull() && !data.Sort.IsUnknown() {
		sort = data.Sort.ValueString()
	} else {
		sort = "desc"
	}
	listGroupsOptions.Sort = &sort
	optionsHash.WriteString(sort)

	optionsHash.WriteString(",")
	if !data.Search.IsNull() && !data.Search.IsUnknown() {
		search := data.Search.ValueString()
		listGroupsOptions.Search = &search
		optionsHash.WriteString(search)
	}
	optionsHash.WriteString(",")
	if !data.TopLevelOnly.IsNull() && !data.TopLevelOnly.IsUnknown() {
		topLevelOnly := data.TopLevelOnly.ValueBool()
		listGroupsOptions.TopLevelOnly = &topLevelOnly
		optionsHash.WriteString(fmt.Sprint(topLevelOnly))
	}

	hasher := sha256.New()
	hasher.Write([]byte(optionsHash.String()))
	id := hex.EncodeToString(hasher.Sum(nil))

	return listGroupsOptions, id
}
