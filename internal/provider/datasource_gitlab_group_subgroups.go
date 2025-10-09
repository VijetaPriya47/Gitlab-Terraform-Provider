package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabGroupSubgroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupSubgroupsDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupSubgroupsDataSource)
}

func NewGitlabGroupSubgroupsDataSource() datasource.DataSource {
	return &gitlabGroupSubgroupsDataSource{}
}

type gitlabGroupSubgroupsDataSource struct {
	client *gitlab.Client
}

type gitlabGroupSubgroupsDataSourceModel struct {
	ID                   types.String                         `tfsdk:"id"`
	GroupID              types.Int64                          `tfsdk:"group_id"`
	SkipGroups           types.List                           `tfsdk:"skip_groups"`
	AllAvailable         types.Bool                           `tfsdk:"all_available"`
	Search               types.String                         `tfsdk:"search"`
	OrderBy              types.String                         `tfsdk:"order_by"`
	Sort                 types.String                         `tfsdk:"sort"`
	Statistics           types.Bool                           `tfsdk:"statistics"`
	WithCustomAttributes types.Bool                           `tfsdk:"with_custom_attributes"`
	Owned                types.Bool                           `tfsdk:"owned"`
	MinAccessLevel       types.String                         `tfsdk:"min_access_level"`
	Subgroups            []gitlabGroupSubgroupDataSourceModel `tfsdk:"subgroups"`
}

type gitlabGroupSubgroupDataSourceModel struct {
	GroupID                        types.Int64  `tfsdk:"group_id"`
	FullPath                       types.String `tfsdk:"full_path"`
	Name                           types.String `tfsdk:"name"`
	FullName                       types.String `tfsdk:"full_name"`
	WebURL                         types.String `tfsdk:"web_url"`
	Path                           types.String `tfsdk:"path"`
	Description                    types.String `tfsdk:"description"`
	AllowedEmailDomainsList        types.String `tfsdk:"allowed_email_domains_list"`
	AutoDevopsEnabled              types.Bool   `tfsdk:"auto_devops_enabled"`
	AvatarURL                      types.String `tfsdk:"avatar_url"`
	CreatedAt                      types.String `tfsdk:"created_at"`
	EmailsEnabled                  types.Bool   `tfsdk:"emails_enabled"`
	IPRestrictionRanges            types.String `tfsdk:"ip_restriction_ranges"`
	MentionsDisabled               types.Bool   `tfsdk:"mentions_disabled"`
	FileTemplateProjectID          types.Int64  `tfsdk:"file_template_project_id"`
	ProjectCreationLevel           types.String `tfsdk:"project_creation_level"`
	SubgroupCreationLevel          types.String `tfsdk:"subgroup_creation_level"`
	TwoFactorGracePeriod           types.Int64  `tfsdk:"two_factor_grace_period"`
	ShareWithGroupLock             types.Bool   `tfsdk:"share_with_group_lock"`
	RequireTwoFactorAuthentication types.Bool   `tfsdk:"require_two_factor_authentication"`
	LFSEnabled                     types.Bool   `tfsdk:"lfs_enabled"`
	RequestAccessEnabled           types.Bool   `tfsdk:"request_access_enabled"`
	Visibility                     types.String `tfsdk:"visibility"`
	ParentID                       types.Int64  `tfsdk:"parent_id"`
	DefaultBranchProtection        types.Int64  `tfsdk:"default_branch_protection"`
	WikiAccessLevel                types.String `tfsdk:"wiki_access_level"`
	SharedRunnersSetting           types.String `tfsdk:"shared_runners_setting"`
	Statistics                     types.Map    `tfsdk:"statistics"`
}

var (
	projectCreationLevelValues  = []string{string(gitlab.NoOneProjectCreation), string(gitlab.OwnerProjectCreation), string(gitlab.MaintainerProjectCreation), string(gitlab.DeveloperProjectCreation), string(gitlab.AdministratorProjectCreation)}
	subGroupCreationLevelValues = []string{string(gitlab.OwnerSubGroupCreationLevelValue), string(gitlab.MaintainerSubGroupCreationLevelValue)}
)

func (d *gitlabGroupSubgroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_subgroups"
}

func (d *gitlabGroupSubgroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_subgroups`" + ` data source allows to get subgroups of a group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/groups/#list-subgroups)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<group-id>`.",
				Computed:            true,
			},
			"group_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the group.",
				Required:            true,
			},
			"skip_groups": schema.ListAttribute{
				MarkdownDescription: "Skip the group IDs passed.",
				Computed:            true,
				Optional:            true,
				ElementType:         types.Int64Type,
			},
			"all_available": schema.BoolAttribute{
				MarkdownDescription: "Show all the groups you have access to.",
				Computed:            true,
				Optional:            true,
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Return the list of authorized groups matching the search criteria.",
				Computed:            true,
				Optional:            true,
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Order groups by name, path or id.",
				Computed:            true,
				Optional:            true,
			},
			"sort": schema.StringAttribute{
				MarkdownDescription: "Order groups in asc or desc order.",
				Computed:            true,
				Optional:            true,
			},
			"statistics": schema.BoolAttribute{
				MarkdownDescription: "Include group statistics (administrators only).",
				Computed:            true,
				Optional:            true,
			},
			"with_custom_attributes": schema.BoolAttribute{
				MarkdownDescription: "Include custom attributes in response (administrators only).",
				Computed:            true,
				Optional:            true,
			},
			"owned": schema.BoolAttribute{
				MarkdownDescription: "Limit to groups explicitly owned by the current user.",
				Computed:            true,
				Optional:            true,
			},
			"min_access_level": schema.StringAttribute{
				MarkdownDescription: "Limit to groups where current user has at least this access level.",
				Computed:            true,
				Optional:            true,
			},
			"subgroups": schema.ListNestedAttribute{
				MarkdownDescription: "Subgroups of the parent group.",
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
						"allowed_email_domains_list": schema.StringAttribute{
							MarkdownDescription: "A list of email address domains to allow group access.",
							Computed:            true,
						},
						"auto_devops_enabled": schema.BoolAttribute{
							MarkdownDescription: "Default to Auto DevOps pipeline for all projects within this group.",
							Computed:            true,
						},
						"avatar_url": schema.StringAttribute{
							MarkdownDescription: "The URL of the avatar image.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Group created at date.",
							Computed:            true,
						},
						"emails_enabled": schema.BoolAttribute{
							MarkdownDescription: "Enable email notifications.",
							Computed:            true,
						},
						"ip_restriction_ranges": schema.StringAttribute{
							MarkdownDescription: "A list of IP addresses or subnet masks to restrict group access.",
							Computed:            true,
						},
						"mentions_disabled": schema.BoolAttribute{
							MarkdownDescription: "Disable the capability of a group from getting mentioned.",
							Computed:            true,
						},
						"file_template_project_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the project that will be used for file templates.",
							Computed:            true,
						},
						"project_creation_level": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("Determine if developers can create projects in the group. Valid values are: %s", utils.RenderValueListForDocs(projectCreationLevelValues)),
							Computed:            true,
						},
						"subgroup_creation_level": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("Allowed to create subgroups. Valid values are: %s.", utils.RenderValueListForDocs(subGroupCreationLevelValues)),
							Computed:            true,
						},
						"two_factor_grace_period": schema.Int64Attribute{
							MarkdownDescription: "Time before Two-factor authentication is enforced (in hours).",
							Computed:            true,
						},
						"share_with_group_lock": schema.BoolAttribute{
							MarkdownDescription: "Prevent sharing a project with another group within this group.",
							Computed:            true,
						},
						"require_two_factor_authentication": schema.BoolAttribute{
							MarkdownDescription: "Require all users in this group to setup Two-factor authentication.",
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
						"visibility": schema.StringAttribute{
							MarkdownDescription: "Limited by visibility `public`, `internal`, or `private`.",
							Computed:            true,
						},
						"parent_id": schema.Int64Attribute{
							MarkdownDescription: "ID of the parent group.",
							Computed:            true,
						},
						"default_branch_protection": schema.Int64Attribute{
							MarkdownDescription: "Whether developers and maintainers can push to the applicable default branch.",
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
						"statistics": schema.MapAttribute{
							MarkdownDescription: "Group statistics.",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabGroupSubgroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabGroupSubgroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabGroupSubgroupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Gitlab group subgroups")

	groupID := int(data.GroupID.ValueInt64())

	options := &gitlab.ListSubGroupsOptions{}
	if !data.SkipGroups.IsNull() && !data.SkipGroups.IsUnknown() {
		var skipGroups []int
		data.SkipGroups.ElementsAs(ctx, &skipGroups, true)
		options.SkipGroups = &skipGroups
	}

	subgroups, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
		return d.client.Groups.ListSubGroups(groupID, options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read subgroups: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(strconv.Itoa(groupID))
	for _, subgroup := range subgroups {
		modelSubgroup := gitlabGroupSubgroupDataSourceModel{
			GroupID:                        types.Int64Value(int64(subgroup.ID)),
			FullPath:                       types.StringValue(subgroup.FullPath),
			Name:                           types.StringValue(subgroup.Name),
			FullName:                       types.StringValue(subgroup.FullName),
			WebURL:                         types.StringValue(subgroup.WebURL),
			Path:                           types.StringValue(subgroup.Path),
			Description:                    types.StringValue(subgroup.Description),
			AllowedEmailDomainsList:        types.StringValue(subgroup.AllowedEmailDomainsList),
			AutoDevopsEnabled:              types.BoolValue(subgroup.AutoDevopsEnabled),
			AvatarURL:                      types.StringValue(subgroup.AvatarURL),
			CreatedAt:                      types.StringValue(subgroup.CreatedAt.Format(time.RFC3339)),
			EmailsEnabled:                  types.BoolValue(subgroup.EmailsEnabled),
			IPRestrictionRanges:            types.StringValue(subgroup.IPRestrictionRanges),
			MentionsDisabled:               types.BoolValue(subgroup.MentionsDisabled),
			FileTemplateProjectID:          types.Int64Value(int64(subgroup.FileTemplateProjectID)),
			ProjectCreationLevel:           types.StringValue(string(subgroup.ProjectCreationLevel)),
			SubgroupCreationLevel:          types.StringValue(string(subgroup.SubGroupCreationLevel)),
			TwoFactorGracePeriod:           types.Int64Value(int64(subgroup.TwoFactorGracePeriod)),
			ShareWithGroupLock:             types.BoolValue(subgroup.ShareWithGroupLock),
			RequireTwoFactorAuthentication: types.BoolValue(subgroup.RequireTwoFactorAuth),
			LFSEnabled:                     types.BoolValue(subgroup.LFSEnabled),
			RequestAccessEnabled:           types.BoolValue(subgroup.RequestAccessEnabled),
			Visibility:                     types.StringValue(string(subgroup.Visibility)),
			ParentID:                       types.Int64Value(int64(subgroup.ParentID)),
			WikiAccessLevel:                types.StringValue(string(subgroup.WikiAccessLevel)),
			SharedRunnersSetting:           types.StringValue(string(subgroup.SharedRunnersSetting)),

			// nolint:staticcheck // SA1019 ignore deprecated DefaultBranchProtection
			DefaultBranchProtection: types.Int64Value(int64(subgroup.DefaultBranchProtection)),
		}

		if subgroup.Statistics == nil {
			modelSubgroup.Statistics = types.MapNull(types.StringType)
		} else {
			elements := map[string]attr.Value{
				"commit_count":            types.StringValue(strconv.FormatInt(subgroup.Statistics.CommitCount, 10)),
				"storage_size":            types.StringValue(strconv.FormatInt(subgroup.Statistics.StorageSize, 10)),
				"repository_size":         types.StringValue(strconv.FormatInt(subgroup.Statistics.RepositorySize, 10)),
				"wiki_size":               types.StringValue(strconv.FormatInt(subgroup.Statistics.WikiSize, 10)),
				"lfs_objects_size":        types.StringValue(strconv.FormatInt(subgroup.Statistics.LFSObjectsSize, 10)),
				"job_artifacts_size":      types.StringValue(strconv.FormatInt(subgroup.Statistics.JobArtifactsSize, 10)),
				"pipeline_artifacts_size": types.StringValue(strconv.FormatInt(subgroup.Statistics.PipelineArtifactsSize, 10)),
				"packages_size":           types.StringValue(strconv.FormatInt(subgroup.Statistics.PackagesSize, 10)),
				"snippets_size":           types.StringValue(strconv.FormatInt(subgroup.Statistics.SnippetsSize, 10)),
				"uploads_size":            types.StringValue(strconv.FormatInt(subgroup.Statistics.UploadsSize, 10)),
				"container_registry_size": types.StringValue(strconv.FormatInt(subgroup.Statistics.ContainerRegistrySize, 10)),
			}
			statistics, diags := types.MapValue(types.StringType, elements)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			modelSubgroup.Statistics = statistics
		}

		data.Subgroups = append(data.Subgroups, modelSubgroup)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
