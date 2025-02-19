package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabApplicationDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabApplicationDataSource{}
)

func init() {
	registerDataSource(NewGitLabGroupProvisionedUsersDataSource)
}

func NewGitLabGroupProvisionedUsersDataSource() datasource.DataSource {
	return &gitlabGroupProvisionedUsersDataSource{}
}

type gitlabGroupProvisionedUsersDataSource struct {
	client *gitlab.Client
}

type gitlabGroupProvisionedUsersDataSourceModel struct {
	Id       types.String `tfsdk:"id"`
	UserName types.String `tfsdk:"username"`
	Search   types.String `tfsdk:"search"`

	Active  types.Bool `tfsdk:"active"`
	Blocked types.Bool `tfsdk:"blocked"`

	CreatedAfter  types.String `tfsdk:"created_after"`
	CreatedBefore types.String `tfsdk:"created_before"`

	Users []gitlabGroupProvisionedUsersObjectDataSourceModel `tfsdk:"provisioned_users"`
}

type gitlabGroupProvisionedUsersObjectDataSourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Username         types.String `tfsdk:"username"`
	State            types.String `tfsdk:"state"`
	AvatarUrl        types.String `tfsdk:"avatar_url"`
	WebUrl           types.String `tfsdk:"web_url"`
	CreatedAt        types.String `tfsdk:"created_at"`
	Bio              types.String `tfsdk:"bio"`
	Location         types.String `tfsdk:"location"`
	PublicEmail      types.String `tfsdk:"public_email"`
	Skype            types.String `tfsdk:"skype"`
	Linkedin         types.String `tfsdk:"linkedin"`
	Twitter          types.String `tfsdk:"twitter"`
	WebsiteUrl       types.String `tfsdk:"website_url"`
	Organization     types.String `tfsdk:"organization"`
	JobTitle         types.String `tfsdk:"job_title"`
	Pronouns         types.String `tfsdk:"pronouns"`
	Bot              types.Bool   `tfsdk:"bot"`
	LastSignInAt     types.String `tfsdk:"last_sign_in_at"`
	ConfirmedAt      types.String `tfsdk:"confirmed_at"`
	LastActivityOn   types.String `tfsdk:"last_activity_on"`
	Email            types.String `tfsdk:"email"`
	TwoFactorEnabled types.Bool   `tfsdk:"two_factor_enabled"`
	External         types.Bool   `tfsdk:"external"`
	PrivateProfile   types.Bool   `tfsdk:"private_profile"`
}

// Metadata returns the data source type name.
func (d *gitlabGroupProvisionedUsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_provisioned_users"
}

func (d *gitlabGroupProvisionedUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_provisioned_users`" + ` data source allows details of the provisioned users of a given group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/groups/#list-provisioned-users)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the group.",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the provisioned user.",
				Optional:            true,
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "The search query to filter the provisioned users.",
				Optional:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Return only active provisioned users.",
				Optional:            true,
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Return only blocked provisioned users.",
				Optional:            true,
			},
			"created_after": schema.StringAttribute{
				MarkdownDescription: "Return only provisioned users created on or after the specified date. Expected in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ).",
				Optional:            true,
			},
			"created_before": schema.StringAttribute{
				MarkdownDescription: "Return only provisioned users created on or before the specified date. Expected in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ).",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"provisioned_users": schema.ListNestedBlock{
				MarkdownDescription: "The list of provisioned users.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the provisioned user.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the provisioned user.",
							Computed:            true,
						},
						"username": schema.StringAttribute{
							MarkdownDescription: "The username of the provisioned user.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: "The state of the provisioned user.",
							Computed:            true,
						},
						"avatar_url": schema.StringAttribute{
							MarkdownDescription: "The avatar URL of the provisioned user.",
							Computed:            true,
						},
						"web_url": schema.StringAttribute{
							MarkdownDescription: "The web URL of the provisioned user.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "The creation date of the provisioned user.",
							Computed:            true,
						},
						"bio": schema.StringAttribute{
							MarkdownDescription: "The bio of the provisioned user.",
							Computed:            true,
						},
						"location": schema.StringAttribute{
							MarkdownDescription: "The location of the provisioned user.",
							Computed:            true,
						},
						"public_email": schema.StringAttribute{
							MarkdownDescription: "The public email of the provisioned user.",
							Computed:            true,
						},
						"skype": schema.StringAttribute{
							MarkdownDescription: "The Skype ID of the provisioned user.",
							Computed:            true,
						},
						"linkedin": schema.StringAttribute{
							MarkdownDescription: "The LinkedIn ID of the provisioned user.",
							Computed:            true,
						},
						"twitter": schema.StringAttribute{
							MarkdownDescription: "The Twitter ID of the provisioned user.",
							Computed:            true,
						},
						"website_url": schema.StringAttribute{
							MarkdownDescription: "The website URL of the provisioned user.",
							Computed:            true,
						},
						"organization": schema.StringAttribute{
							MarkdownDescription: "The organization of the provisioned user.",
							Computed:            true,
						},
						"job_title": schema.StringAttribute{
							MarkdownDescription: "The job title of the provisioned user.",
							Computed:            true,
						},
						"pronouns": schema.StringAttribute{
							MarkdownDescription: "The pronouns of the provisioned user.",
							Computed:            true,
						},
						"bot": schema.BoolAttribute{
							MarkdownDescription: "Whether the provisioned user is a bot.",
							Computed:            true,
						},
						"last_sign_in_at": schema.StringAttribute{
							MarkdownDescription: "The last sign-in date of the provisioned user.",
							Computed:            true,
						},
						"confirmed_at": schema.StringAttribute{
							MarkdownDescription: "The confirmation date of the provisioned user.",
							Computed:            true,
						},
						"last_activity_on": schema.StringAttribute{
							MarkdownDescription: "The last activity date of the provisioned user.",
							Computed:            true,
						},
						"email": schema.StringAttribute{
							MarkdownDescription: "The email of the provisioned user.",
							Computed:            true,
						},
						"two_factor_enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether two-factor authentication is enabled for the provisioned user.",
							Computed:            true,
						},
						"external": schema.BoolAttribute{
							MarkdownDescription: "Whether the provisioned user is external.",
							Computed:            true,
						},
						"private_profile": schema.BoolAttribute{
							MarkdownDescription: "Whether the provisioned user has a private profile.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabGroupProvisionedUsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabGroupProvisionedUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitlabGroupProvisionedUsersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.ListProvisionedUsersOptions{}

	if !state.Search.IsNull() && !state.Search.IsUnknown() {
		search := state.Search.ValueString()
		options.Search = &search
	}

	if !state.CreatedAfter.IsNull() {
		parsedCreatedAfter, err := time.Parse(time.RFC3339, state.CreatedAfter.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid created_after value", fmt.Sprintf("Unable to parse created_after value: %s", err.Error()))
			return
		}
		options.CreatedBefore = gitlab.Ptr(parsedCreatedAfter)
	}

	if !state.CreatedBefore.IsNull() {
		parsedCreatedBefore, err := time.Parse(time.RFC3339, state.CreatedBefore.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid created_before value", fmt.Sprintf("Unable to parse created_before value: %s", err.Error()))
			return
		}
		options.CreatedBefore = gitlab.Ptr(parsedCreatedBefore)
	}

	if !state.Active.IsNull() && !state.Active.IsUnknown() {
		options.Active = gitlab.Ptr(state.Active.ValueBool())
	}

	if !state.Blocked.IsNull() && !state.Blocked.IsUnknown() {
		options.Blocked = gitlab.Ptr(state.Blocked.ValueBool())
	}

	groupId, _ := strconv.Atoi(state.Id.ValueString())

	provisionedUsers, _, err := d.client.Groups.ListProvisionedUsers(groupId, options)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read provisioned users list: %s", err.Error()))
		return
	}

	provisionedUsersModel := make([]gitlabGroupProvisionedUsersObjectDataSourceModel, len(provisionedUsers))

	for i, provisionedUser := range provisionedUsers {
		createdAt := ""
		if provisionedUser.CreatedAt != nil {
			createdAt = provisionedUser.CreatedAt.Format(time.RFC3339)
		}

		pu := gitlabGroupProvisionedUsersObjectDataSourceModel{
			Id:               types.StringValue(fmt.Sprintf("%d", provisionedUser.ID)),
			Name:             types.StringValue(provisionedUser.Name),
			Username:         types.StringValue(provisionedUser.Username),
			State:            types.StringValue(provisionedUser.State),
			AvatarUrl:        types.StringValue(provisionedUser.AvatarURL),
			WebUrl:           types.StringValue(provisionedUser.WebURL),
			CreatedAt:        types.StringValue(createdAt),
			Bio:              types.StringValue(provisionedUser.Bio),
			Location:         types.StringValue(provisionedUser.Location),
			PublicEmail:      types.StringValue(provisionedUser.PublicEmail),
			Skype:            types.StringValue(provisionedUser.Skype),
			Linkedin:         types.StringValue(provisionedUser.Linkedin),
			Twitter:          types.StringValue(provisionedUser.Twitter),
			WebsiteUrl:       types.StringValue(provisionedUser.WebsiteURL),
			Organization:     types.StringValue(provisionedUser.Organization),
			JobTitle:         types.StringValue(provisionedUser.JobTitle),
			Email:            types.StringValue(provisionedUser.Email),
			LastSignInAt:     types.StringValue(provisionedUser.LastSignInAt.Format(time.RFC3339)),
			TwoFactorEnabled: types.BoolValue(provisionedUser.TwoFactorEnabled),
			External:         types.BoolValue(provisionedUser.External),
			PrivateProfile:   types.BoolValue(provisionedUser.PrivateProfile),
		}

		provisionedUsersModel[i] = pu
	}

	state.Users = provisionedUsersModel

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
