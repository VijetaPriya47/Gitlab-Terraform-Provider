package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	_ datasource.DataSource              = &gitlabUsersDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabUsersDataSource{}
)

func init() {
	registerDataSource(NewGitlabUsersDataSource)
}

func NewGitlabUsersDataSource() datasource.DataSource {
	return &gitlabUsersDataSource{}
}

type gitlabUsersDataSource struct {
	client *gitlab.Client
}

type gitlabUsersDataSourceModel struct {
	ID                 types.String                           `tfsdk:"id"`
	OrderBy            types.String                           `tfsdk:"order_by"`
	Sort               types.String                           `tfsdk:"sort"`
	Username           types.String                           `tfsdk:"username"`
	Search             types.String                           `tfsdk:"search"`
	Active             types.Bool                             `tfsdk:"active"`
	External           types.Bool                             `tfsdk:"external"`
	Blocked            types.Bool                             `tfsdk:"blocked"`
	ExternUID          types.String                           `tfsdk:"extern_uid"`
	ExternProvider     types.String                           `tfsdk:"extern_provider"`
	CreatedBefore      types.String                           `tfsdk:"created_before"`
	CreatedAfter       types.String                           `tfsdk:"created_after"`
	ExcludeExternal    types.Bool                             `tfsdk:"exclude_external"`
	ExcludeInternal    types.Bool                             `tfsdk:"exclude_internal"`
	WithoutProjectBots types.Bool                             `tfsdk:"without_project_bots"`
	Users              []gitlabUsersIndividualDataSourceModel `tfsdk:"users"`
}

type gitlabUsersIndividualDataSourceModel struct {
	ID               types.Int64  `tfsdk:"id"`
	Username         types.String `tfsdk:"username"`
	Email            types.String `tfsdk:"email"`
	Name             types.String `tfsdk:"name"`
	IsAdmin          types.Bool   `tfsdk:"is_admin"`
	IsBot            types.Bool   `tfsdk:"is_bot"`
	CanCreateGroup   types.Bool   `tfsdk:"can_create_group"`
	CanCreateProject types.Bool   `tfsdk:"can_create_project"`
	ProjectsLimit    types.Int64  `tfsdk:"projects_limit"`
	CreatedAt        types.String `tfsdk:"created_at"`
	State            types.String `tfsdk:"state"`
	External         types.Bool   `tfsdk:"external"`
	ExternUID        types.String `tfsdk:"extern_uid"`
	Organization     types.String `tfsdk:"organization"`
	TwoFactorEnabled types.Bool   `tfsdk:"two_factor_enabled"`
	Provider         types.String `tfsdk:"provider"`
	AvatarURL        types.String `tfsdk:"avatar_url"`
	Bio              types.String `tfsdk:"bio"`
	Location         types.String `tfsdk:"location"`
	Skype            types.String `tfsdk:"skype"`
	LinkedIn         types.String `tfsdk:"linkedin"`
	Twitter          types.String `tfsdk:"twitter"`
	WebsiteURL       types.String `tfsdk:"website_url"`
	ThemeID          types.Int64  `tfsdk:"theme_id"`
	ColorSchemeID    types.Int64  `tfsdk:"color_scheme_id"`
	LastSignInAt     types.String `tfsdk:"last_sign_in_at"`
	CurrentSignInAt  types.String `tfsdk:"current_sign_in_at"`
	NamespaceID      types.Int64  `tfsdk:"namespace_id"`
}

func (d *gitlabUsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *gitlabUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_users`" + ` data source allows details of multiple users to be retrieved given some optional filter criteria.

-> Some attributes might not be returned depending on if you're an admin or not.

-> Some available options require administrator privileges.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/users/#list-users)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format of a hash taken from the search options.",
				Computed:            true,
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Order the users' list by `id`, `name`, `username`, `created_at` or `updated_at`. (Requires administrator privileges)",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("id", "name", "username", "created_at", "updated_at")},
			},
			"sort": schema.StringAttribute{
				MarkdownDescription: "Sort users' list in asc or desc order. (Requires administrator privileges)",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("asc", "desc")},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Get a single user with a specific username.",
				Optional:            true,
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Search users by username, name or email.",
				Optional:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Filter users that are active.",
				Optional:            true,
			},
			"external": schema.BoolAttribute{
				MarkdownDescription: "Filters only external users.",
				Optional:            true,
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Filter users that are blocked.",
				Optional:            true,
			},
			"extern_uid": schema.StringAttribute{
				MarkdownDescription: "Lookup users by external UID. (Requires administrator privileges)",
				Optional:            true,
			},
			"extern_provider": schema.StringAttribute{
				MarkdownDescription: "Lookup users by external provider. (Requires administrator privileges)",
				Optional:            true,
			},
			"created_before": schema.StringAttribute{
				MarkdownDescription: "Search for users created before a specific date. (Requires administrator privileges)",
				Optional:            true,
			},
			"created_after": schema.StringAttribute{
				MarkdownDescription: "Search for users created after a specific date. (Requires administrator privileges)",
				Optional:            true,
			},
			"exclude_external": schema.BoolAttribute{
				MarkdownDescription: "Filters only non external users.",
				Optional:            true,
			},
			"exclude_internal": schema.BoolAttribute{
				MarkdownDescription: "Filters only non internal users.",
				Optional:            true,
			},
			"without_project_bots": schema.BoolAttribute{
				MarkdownDescription: "Filters user without project bots.",
				Optional:            true,
			},
			"users": schema.ListNestedAttribute{
				MarkdownDescription: "The list of users.",
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
						"email": schema.StringAttribute{
							MarkdownDescription: "The public email address of the user.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the user.",
							Computed:            true,
						},
						"is_admin": schema.BoolAttribute{
							MarkdownDescription: "Whether the user is an admin.",
							Computed:            true,
						},
						"is_bot": schema.BoolAttribute{
							MarkdownDescription: "Whether the user is a bot.",
							Computed:            true,
						},
						"can_create_group": schema.BoolAttribute{
							MarkdownDescription: "Whether the user can create groups.",
							Computed:            true,
						},
						"can_create_project": schema.BoolAttribute{
							MarkdownDescription: "Whether the user can create projects.",
							Computed:            true,
						},
						"projects_limit": schema.Int64Attribute{
							MarkdownDescription: "Number of projects the user can create.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Date the user was created at.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: "Whether the user is active or blocked.",
							Computed:            true,
						},
						"external": schema.BoolAttribute{
							MarkdownDescription: "Whether the user is external.",
							Computed:            true,
						},
						"extern_uid": schema.StringAttribute{
							MarkdownDescription: "The external UID of the user.",
							Computed:            true,
						},
						"organization": schema.StringAttribute{
							MarkdownDescription: "The organization of the user.",
							Computed:            true,
						},
						"two_factor_enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether user's two-factor auth is enabled.",
							Computed:            true,
						},
						"provider": schema.StringAttribute{
							MarkdownDescription: "The UID provider of the user.",
							Computed:            true,
						},
						"avatar_url": schema.StringAttribute{
							MarkdownDescription: "The avatar URL of the user.",
							Computed:            true,
						},
						"bio": schema.StringAttribute{
							MarkdownDescription: "The bio of the user.",
							Computed:            true,
						},
						"location": schema.StringAttribute{
							MarkdownDescription: "The location of the user.",
							Computed:            true,
						},
						"skype": schema.StringAttribute{
							MarkdownDescription: "Skype username of the user.",
							Computed:            true,
						},
						"linkedin": schema.StringAttribute{
							MarkdownDescription: "LinkedIn profile of the user.",
							Computed:            true,
						},
						"twitter": schema.StringAttribute{
							MarkdownDescription: "Twitter username of the user.",
							Computed:            true,
						},
						"website_url": schema.StringAttribute{
							MarkdownDescription: "User's website URL.",
							Computed:            true,
						},
						"theme_id": schema.Int64Attribute{
							MarkdownDescription: "User's theme ID.",
							Computed:            true,
						},
						"color_scheme_id": schema.Int64Attribute{
							MarkdownDescription: "User's color scheme ID.",
							Computed:            true,
						},
						"last_sign_in_at": schema.StringAttribute{
							MarkdownDescription: "Last user's sign-in date.",
							Computed:            true,
						},
						"current_sign_in_at": schema.StringAttribute{
							MarkdownDescription: "Current user's sign-in date.",
							Computed:            true,
						},
						"namespace_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the user's namespace. Requires admin token to access this field.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *gitlabUsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabUsersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	listUsersOptions, id, err := expandGitlabUsersOptions(data)
	if err != nil {
		resp.Diagnostics.AddError("Error building options for getting users", err.Error())
		return
	}

	users, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.User, *gitlab.Response, error) {
		return d.client.Users.ListUsers(listUsersOptions, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read users: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(id)
	data.flattenGitlabUsers(users)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (data *gitlabUsersDataSourceModel) flattenGitlabUsers(users []*gitlab.User) {
	for _, user := range users {
		modelUser := gitlabUsersIndividualDataSourceModel{
			ID:               types.Int64Value(int64(user.ID)),
			Username:         types.StringValue(user.Username),
			Email:            types.StringValue(user.Email),
			Name:             types.StringValue(user.Name),
			IsAdmin:          types.BoolValue(user.IsAdmin),
			IsBot:            types.BoolValue(user.Bot),
			CanCreateGroup:   types.BoolValue(user.CanCreateGroup),
			CanCreateProject: types.BoolValue(user.CanCreateProject),
			ProjectsLimit:    types.Int64Value(int64(user.ProjectsLimit)),
			State:            types.StringValue(user.State),
			External:         types.BoolValue(user.External),
			ExternUID:        types.StringValue(user.ExternUID),
			Provider:         types.StringValue(user.Provider),
			TwoFactorEnabled: types.BoolValue(user.TwoFactorEnabled),
			AvatarURL:        types.StringValue(user.AvatarURL),
			Bio:              types.StringValue(user.Bio),
			Location:         types.StringValue(user.Location),
			Skype:            types.StringValue(user.Skype),
			LinkedIn:         types.StringValue(user.Linkedin),
			Twitter:          types.StringValue(user.Twitter),
			WebsiteURL:       types.StringValue(user.WebsiteURL),
			Organization:     types.StringValue(user.Organization),
			ThemeID:          types.Int64Value(int64(user.ThemeID)),
			ColorSchemeID:    types.Int64Value(int64(user.ColorSchemeID)),
			NamespaceID:      types.Int64Value(int64(user.NamespaceID)),
		}

		if user.CreatedAt == nil {
			modelUser.CreatedAt = types.StringNull()
		} else {
			modelUser.CreatedAt = types.StringValue(user.CreatedAt.String())
		}
		if user.LastSignInAt == nil {
			modelUser.LastSignInAt = types.StringNull()
		} else {
			modelUser.LastSignInAt = types.StringValue(user.LastSignInAt.String())
		}
		if user.CurrentSignInAt == nil {
			modelUser.CurrentSignInAt = types.StringNull()
		} else {
			modelUser.CurrentSignInAt = types.StringValue(user.CurrentSignInAt.String())
		}

		data.Users = append(data.Users, modelUser)
	}
}

func expandGitlabUsersOptions(data *gitlabUsersDataSourceModel) (*gitlab.ListUsersOptions, string, error) {
	listUsersOptions := &gitlab.ListUsersOptions{}
	var optionsHash strings.Builder

	if !data.OrderBy.IsNull() && !data.OrderBy.IsUnknown() {
		orderBy := data.OrderBy.ValueString()
		listUsersOptions.OrderBy = &orderBy
		optionsHash.WriteString(orderBy)
	} else {
		orderBy := "id"
		listUsersOptions.OrderBy = &orderBy
		optionsHash.WriteString(orderBy)
	}
	optionsHash.WriteString(",")
	if !data.Sort.IsNull() && !data.Sort.IsUnknown() {
		sort := data.Sort.ValueString()
		listUsersOptions.Sort = &sort
		optionsHash.WriteString(sort)
	} else {
		sort := "desc"
		listUsersOptions.Sort = &sort
		optionsHash.WriteString(sort)
	}
	optionsHash.WriteString(",")
	if !data.Username.IsNull() && !data.Username.IsUnknown() {
		username := data.Username.ValueString()
		listUsersOptions.Username = &username
		optionsHash.WriteString(username)
	}
	optionsHash.WriteString(",")
	if !data.Search.IsNull() && !data.Search.IsUnknown() {
		search := data.Search.ValueString()
		listUsersOptions.Search = &search
		optionsHash.WriteString(search)
	}
	optionsHash.WriteString(",")
	if !data.Active.IsNull() && !data.Active.IsUnknown() {
		active := data.Active.ValueBool()
		listUsersOptions.Active = &active
		optionsHash.WriteString(strconv.FormatBool(active))
	}
	optionsHash.WriteString(",")
	if !data.External.IsNull() && !data.External.IsUnknown() {
		external := data.External.ValueBool()
		listUsersOptions.External = &external
		optionsHash.WriteString(strconv.FormatBool(external))
	}
	optionsHash.WriteString(",")
	if !data.Blocked.IsNull() && !data.Blocked.IsUnknown() {
		blocked := data.Blocked.ValueBool()
		listUsersOptions.Blocked = &blocked
		optionsHash.WriteString(strconv.FormatBool(blocked))
	}
	optionsHash.WriteString(",")
	if !data.ExternUID.IsNull() && !data.ExternUID.IsUnknown() {
		externalUID := data.ExternUID.ValueString()
		listUsersOptions.ExternalUID = &externalUID
		optionsHash.WriteString(externalUID)
	}
	optionsHash.WriteString(",")
	if !data.ExternProvider.IsNull() && !data.ExternProvider.IsUnknown() {
		provider := data.ExternProvider.ValueString()
		listUsersOptions.Provider = &provider
		optionsHash.WriteString(provider)
	}
	optionsHash.WriteString(",")
	if !data.CreatedBefore.IsNull() && !data.CreatedBefore.IsUnknown() {
		createdBefore := data.CreatedBefore.ValueString()
		date, err := time.Parse("2006-01-02", createdBefore)
		if err != nil {
			return nil, "", fmt.Errorf("created_before must be in yyyy-mm-dd format")
		}
		listUsersOptions.CreatedBefore = &date
		optionsHash.WriteString(createdBefore)
	}
	optionsHash.WriteString(",")
	if !data.CreatedAfter.IsNull() && !data.CreatedAfter.IsUnknown() {
		createdAfter := data.CreatedAfter.ValueString()
		date, err := time.Parse("2006-01-02", createdAfter)
		if err != nil {
			return nil, "", fmt.Errorf("created_after must be in yyyy-mm-dd format")
		}
		listUsersOptions.CreatedAfter = &date
		optionsHash.WriteString(createdAfter)
	}
	optionsHash.WriteString(",")
	if !data.ExcludeExternal.IsNull() && !data.ExcludeExternal.IsUnknown() {
		excludeExternal := data.ExcludeExternal.ValueBool()
		listUsersOptions.ExcludeExternal = &excludeExternal
		optionsHash.WriteString(strconv.FormatBool(excludeExternal))
	}
	optionsHash.WriteString(",")
	if !data.ExcludeInternal.IsNull() && !data.ExcludeInternal.IsUnknown() {
		excludeInternal := data.ExcludeInternal.ValueBool()
		listUsersOptions.ExcludeInternal = &excludeInternal
		optionsHash.WriteString(strconv.FormatBool(excludeInternal))
	}
	optionsHash.WriteString(",")
	if !data.WithoutProjectBots.IsNull() && !data.WithoutProjectBots.IsUnknown() {
		withoutProjectBots := data.WithoutProjectBots.ValueBool()
		listUsersOptions.WithoutProjectBots = &withoutProjectBots
		optionsHash.WriteString(strconv.FormatBool(withoutProjectBots))
	}

	hasher := sha256.New()
	hasher.Write([]byte(optionsHash.String()))
	id := hex.EncodeToString(hasher.Sum(nil))

	return listUsersOptions, id, nil
}
