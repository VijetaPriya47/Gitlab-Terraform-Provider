package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"time"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabGroupBillableMemberMembershipsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupBillableMemberMembershipsDataSource{}
)

func init() {
	registerDataSource(newGitlabGroupBillableMemberMembershipDataSource)
}

// newGitlabGroupBillableMemberMembershipDataSource is a helper function to simplify the provider implementation.
func newGitlabGroupBillableMemberMembershipDataSource() datasource.DataSource {
	return &gitlabGroupBillableMemberMembershipsDataSource{}
}

// gitlabGroupBillableMemberMembershipsDataSource is the data source implementation.
type gitlabGroupBillableMemberMembershipsDataSource struct {
	client *gitlab.Client
}

// gitlabGroupBillableMemberMembershipsDataSourceModel describes the data source data model
type gitlabGroupBillableMemberMembershipsDataSourceModel struct {
	Id      types.String `tfsdk:"id"`
	GroupId types.String `tfsdk:"group_id"`
	UserId  types.Int64  `tfsdk:"user_id"`

	Memberships []gitlabGroupBillableMemberMembershipModel `tfsdk:"memberships"`
}

type gitlabGroupBillableMemberMembershipModel struct {
	Id               types.Int64  `tfsdk:"id"`
	SourceId         types.Int64  `tfsdk:"source_id"`
	SourceFullName   types.String `tfsdk:"source_full_name"`
	SourceMembersUrl types.String `tfsdk:"source_members_url"`
	AccessLevel      types.String `tfsdk:"access_level"`
	CreatedAt        types.String `tfsdk:"created_at"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
}

// Metadata returns the data source type name.
func (d *gitlabGroupBillableMemberMembershipsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_billable_member_memberships"
}

// Schema defines the schema for the data source.
func (d *gitlabGroupBillableMemberMembershipsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_billable_member_memberships`" + ` data source allows (sub)group- and project-memberships of a billable member of a group to be retrieved by either the user ID, username or email address.

-> You must be an administrator!

~> When using the ` + "`email`" + ` attribute, an exact match is not guaranteed. The most related match will be returned. Starting with GitLab 16.6,
the most related match will prioritize an exact match if one is available.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/members.html#list-memberships-for-a-billable-member-of-a-group)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The id of the data source. It will always be equal to the user_id",
				Computed:            true,
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user.",
				Required:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"memberships": schema.ListNestedAttribute{
				MarkdownDescription: "group- and/or project-memberships of the user.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The id of the membership.",
							Computed:            true,
						},
						"source_id": schema.Int64Attribute{
							MarkdownDescription: "The id of the group or project, the user is a (direct) member of.",
							Computed:            true,
						},
						"source_full_name": schema.StringAttribute{
							MarkdownDescription: "Breadcrumb-style, full display-name of the group or project.",
							Computed:            true,
						},
						"source_members_url": schema.StringAttribute{
							MarkdownDescription: "URL to the members-page of the group or project.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Datetime when the membership was first added.",
							Computed:            true,
						},
						"expires_at": schema.StringAttribute{
							MarkdownDescription: "Date when the membership will end.",
							Computed:            true,
						},
						"access_level": schema.StringAttribute{
							MarkdownDescription: "Access-level of the member. For details see: https://docs.gitlab.com/ee/api/access_requests.html#valid-access-levels",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabGroupBillableMemberMembershipsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabGroupBillableMemberMembershipsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitlabGroupBillableMemberMembershipsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "[INFO] Reading Gitlab user memberships")
	membership, err := d.fetchAllOfListMembershipsForBillableGroupMember(state.GroupId.ValueString(), int(state.UserId.ValueInt64()), ctx)

	if err != nil {
		resp.Diagnostics.AddError("API call to ListMembershipsForBillableGroupMember failed", err.Error())
		return
	}

	state.Id = types.StringValue(fmt.Sprintf("%d", state.UserId.ValueInt64()))
	state.Memberships = flattenBillableMemberMembershipsForState(membership)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *gitlabGroupBillableMemberMembershipsDataSource) fetchAllOfListMembershipsForBillableGroupMember(groupId interface{}, userId int, ctx context.Context) ([]*gitlab.BillableUserMembership, error) {
	var membership []*gitlab.BillableUserMembership

	listOptions := &gitlab.ListMembershipsForBillableGroupMemberOptions{
		PerPage: 20,
		Page:    1,
	}

	for listOptions.Page != 0 {
		m, resp, err := d.client.Groups.ListMembershipsForBillableGroupMember(groupId, userId, listOptions, gitlab.WithContext(ctx))
		if err != nil {
			return nil, err
		}

		membership = append(membership, m...)

		listOptions.Page = resp.NextPage
	}

	return membership, nil
}

func flattenBillableMemberMembershipsForState(memberships []*gitlab.BillableUserMembership) []gitlabGroupBillableMemberMembershipModel {
	values := make([]gitlabGroupBillableMemberMembershipModel, 0, len(memberships))
	for _, group := range memberships {
		values = append(values, gitlabBillableMemberMembershipToStateModel(group))
	}
	return values
}

func gitlabBillableMemberMembershipToStateModel(membership *gitlab.BillableUserMembership) gitlabGroupBillableMemberMembershipModel {
	var createdAt = ""
	if membership.CreatedAt != nil {
		createdAt = membership.CreatedAt.Format(time.RFC3339)
	}

	m := gitlabGroupBillableMemberMembershipModel{
		Id:               types.Int64Value(int64(membership.ID)),
		SourceId:         types.Int64Value(int64(membership.SourceID)),
		SourceFullName:   types.StringValue(membership.SourceFullName),
		SourceMembersUrl: types.StringValue(membership.SourceMembersURL),
		AccessLevel:      types.StringValue(api.AccessLevelValueToName[membership.AccessLevel.IntegerValue]),
		CreatedAt:        types.StringValue(createdAt),
	}
	if membership.ExpiresAt != nil {
		m.ExpiresAt = types.StringValue(membership.ExpiresAt.Format(time.RFC3339))
	}

	return m
}
