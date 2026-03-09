package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ datasource.DataSource              = &gitlabGroupSAMLLinksDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupSAMLLinksDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupSAMLLinksDataSource)
}

func NewGitlabGroupSAMLLinksDataSource() datasource.DataSource {
	return &gitlabGroupSAMLLinksDataSource{}
}

type gitlabGroupSAMLLinksDataSource struct {
	client *gitlab.Client
}

type gitlabGroupSAMLLinksDataSourceModel struct {
	ID        types.String                    `tfsdk:"id"`
	Group     types.String                    `tfsdk:"group"`
	SAMLLinks []gitlabGroupSAMLLinksListModel `tfsdk:"saml_links"`
}

type gitlabGroupSAMLLinksListModel struct {
	Name         types.String `tfsdk:"name"`
	AccessLevel  types.String `tfsdk:"access_level"`
	MemberRoleID types.Int64  `tfsdk:"member_role_id"`
}

func (d *gitlabGroupSAMLLinksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_saml_links"
}

func (d *gitlabGroupSAMLLinksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_saml_links`" + ` data source retrieves all SAML links for a specified group.
	
**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/saml/#saml-group-links)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format <group-id>",
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The name or id of the group.",
				Required:            true,
			},
			"saml_links": schema.ListNestedAttribute{
				MarkdownDescription: "The list of group SAML links returned by the search",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the SAML group.",
							Computed:            true,
						},
						"access_level": schema.StringAttribute{
							MarkdownDescription: "The base access level for members of the SAML group.",
							Computed:            true,
						},
						"member_role_id": schema.Int64Attribute{
							MarkdownDescription: "Member Role ID (custom role for members of the SAML group.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabGroupSAMLLinksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabGroupSAMLLinksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabGroupSAMLLinksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := data.Group.ValueString()
	samlLinks, _, err := d.client.Groups.ListGroupSAMLLinks(group, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Unable to fetch SAML links", err.Error())
		return
	}

	data.SAMLLinks = []gitlabGroupSAMLLinksListModel{}

	for _, link := range samlLinks {
		data.SAMLLinks = append(data.SAMLLinks, gitlabGroupSAMLLinksListModel{
			Name:         types.StringValue(link.Name),
			AccessLevel:  types.StringValue(api.AccessLevelValueToName[link.AccessLevel]),
			MemberRoleID: types.Int64Value(int64(link.MemberRoleID)),
		})
	}

	data.ID = types.StringValue(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
