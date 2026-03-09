package provider

import (
	"context"
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
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabGroupMembershipDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupMembershipDataSource{}
)

func init() {
	registerDataSource(NewGitLabGroupMembershipDataSource)
}

// NewGitLabGroupMembershipDataSource is a helper function to simplify the provider implementation.
func NewGitLabGroupMembershipDataSource() datasource.DataSource {
	return &gitlabGroupMembershipDataSource{}
}

// gitlabGroupMembershipDataSource is the data source implementation.
type gitlabGroupMembershipDataSource struct {
	client *gitlab.Client
}

// gitlabGroupMembershipDataSourceModel describes the data source data model.
type gitlabGroupMembershipDataSourceModel struct {
	ID          types.String                       `tfsdk:"id"`
	GroupID     types.Int64                        `tfsdk:"group_id"`
	FullPath    types.String                       `tfsdk:"full_path"`
	Inherited   types.Bool                         `tfsdk:"inherited"`
	AccessLevel types.String                       `tfsdk:"access_level"`
	Members     []gitlabGroupMembershipMemberModel `tfsdk:"members"`
}

type gitlabGroupMembershipMemberModel struct {
	ID                types.Int64                                  `tfsdk:"id"`
	Username          types.String                                 `tfsdk:"username"`
	Name              types.String                                 `tfsdk:"name"`
	State             types.String                                 `tfsdk:"state"`
	AvatarURL         types.String                                 `tfsdk:"avatar_url"`
	WebURL            types.String                                 `tfsdk:"web_url"`
	AccessLevel       types.String                                 `tfsdk:"access_level"`
	GroupSamlIdentity *gitlabGroupMembershipGroupSamlIdentityModel `tfsdk:"group_saml_identity"`
	ExpiresAt         types.String                                 `tfsdk:"expires_at"`
}

type gitlabGroupMembershipGroupSamlIdentityModel struct {
	ExternUID      types.String `tfsdk:"extern_uid"`
	Provider       types.String `tfsdk:"provider"`
	SamlProviderID types.Int64  `tfsdk:"saml_provider_id"`
}

// Metadata returns the data source type name.
func (d *gitlabGroupMembershipDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

// Schema defines the schema for the data source.
func (d *gitlabGroupMembershipDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_membership`" + ` data source allows to list and filter all members of a group specified by either its id or full path.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_members/#list-all-members-of-a-group)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group membership. In the format of `<group-id:access-level>`.",
				Computed:            true,
			},
			"group_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the group.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.Int64{int64validator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("group_id"), path.MatchRelative().AtParent().AtName("full_path"))},
			},
			"full_path": schema.StringAttribute{
				MarkdownDescription: "The full path of the group.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.String{stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("group_id"), path.MatchRelative().AtParent().AtName("full_path"))},
			},
			"inherited": schema.BoolAttribute{
				MarkdownDescription: "Return all project members including members through ancestor groups.",
				Optional:            true,
			},
			"access_level": schema.StringAttribute{
				MarkdownDescription: "Only return members with the desired access level. Acceptable values are: `guest`, `reporter`, `developer`, `maintainer`, `owner`.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidGroupAccessLevelNames...)},
			},
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "The list of group members.",
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
						"group_saml_identity": schema.SingleNestedAttribute{
							MarkdownDescription: "SAML identity linked to the group member.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"extern_uid": schema.StringAttribute{
									MarkdownDescription: "The external UID of the group SAML identity.",
									Computed:            true,
								},
								"provider": schema.StringAttribute{
									MarkdownDescription: "The provider of the SAML identity.",
									Computed:            true,
								},
								"saml_provider_id": schema.Int64Attribute{
									MarkdownDescription: "The ID of the SAML provider.",
									Computed:            true,
								},
							},
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

// Configure adds the provider configured client to the data source.
func (d *gitlabGroupMembershipDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabGroupMembershipDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabGroupMembershipDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "[INFO] Reading Gitlab group")

	var groupLookup string

	if !data.GroupID.IsNull() {
		groupLookup = strconv.Itoa(int(data.GroupID.ValueInt64()))
	} else {
		groupLookup = data.FullPath.ValueString()
	}

	group, _, err := d.client.Groups.GetGroup(groupLookup, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("API call to get group failed", err.Error())
		return
	}

	tflog.Info(ctx, "[INFO] Reading Gitlab group memberships")

	// Get group memberships
	listOptions := &gitlab.ListGroupMembersOptions{}
	listMembers := d.client.Groups.ListGroupMembers

	if !data.Inherited.IsNull() && data.Inherited.ValueBool() {
		listMembers = d.client.Groups.ListAllGroupMembers
	}
	allGms, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.GroupMember, *gitlab.Response, error) {
		return listMembers(group.ID, listOptions, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("API call to ListGroupMembers failed", err.Error())
		return
	}

	data.GroupID = types.Int64Value(int64(group.ID))
	data.FullPath = types.StringValue(group.FullPath)
	data.Members = flattenGitlabGroupMembers(data.AccessLevel, allGms)

	var optionsHash strings.Builder
	optionsHash.WriteString(strconv.FormatInt(group.ID, 10))
	if !data.AccessLevel.IsNull() {
		optionsHash.WriteString(data.AccessLevel.ValueString())
	}

	data.ID = types.StringValue(optionsHash.String())
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func flattenGitlabGroupMembers(accessLevel types.String, members []*gitlab.GroupMember) []gitlabGroupMembershipMemberModel {
	membersList := []gitlabGroupMembershipMemberModel{}

	filterAccessLevel := gitlab.NoPermissions
	if !accessLevel.IsNull() {
		filterAccessLevel = api.AccessLevelNameToValue[accessLevel.ValueString()]
	}

	for _, member := range members {
		if filterAccessLevel != gitlab.NoPermissions && filterAccessLevel != member.AccessLevel {
			continue
		}

		memberModel := gitlabGroupMembershipMemberModel{
			ID:          types.Int64Value(int64(member.ID)),
			Username:    types.StringValue(member.Username),
			Name:        types.StringValue(member.Name),
			State:       types.StringValue(member.State),
			AvatarURL:   types.StringValue(member.AvatarURL),
			WebURL:      types.StringValue(member.WebURL),
			AccessLevel: types.StringValue(api.AccessLevelValueToName[member.AccessLevel]),
		}
		if member.GroupSAMLIdentity != nil {
			memberModel.GroupSamlIdentity = &gitlabGroupMembershipGroupSamlIdentityModel{
				ExternUID:      types.StringValue(member.GroupSAMLIdentity.ExternUID),
				Provider:       types.StringValue(member.GroupSAMLIdentity.Provider),
				SamlProviderID: types.Int64Value(int64(member.GroupSAMLIdentity.SAMLProviderID)),
			}
		}
		if member.ExpiresAt != nil {
			memberModel.ExpiresAt = types.StringValue(member.ExpiresAt.String())
		}
		membersList = append(membersList, memberModel)
	}

	return membersList
}
