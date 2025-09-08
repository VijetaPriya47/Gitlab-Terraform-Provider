package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	_ datasource.DataSource              = &gitlabUserDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabUserDataSource{}
)

func init() {
	registerDataSource(NewGitlabUserDataSource)
}

func NewGitlabUserDataSource() datasource.DataSource {
	return &gitlabUserDataSource{}
}

type gitlabUserDataSource struct {
	client *gitlab.Client
}

type gitlabUserDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	UserID           types.Int64  `tfsdk:"user_id"`
	Username         types.String `tfsdk:"username"`
	Email            types.String `tfsdk:"email"`
	Name             types.String `tfsdk:"name"`
	IsAdmin          types.Bool   `tfsdk:"is_admin"`
	CanCreateGroup   types.Bool   `tfsdk:"can_create_group"`
	CanCreateProject types.Bool   `tfsdk:"can_create_project"`
	ProjectsLimit    types.Int64  `tfsdk:"projects_limit"`
	CreatedAt        types.String `tfsdk:"created_at"`
	State            types.String `tfsdk:"state"`
	External         types.Bool   `tfsdk:"external"`
	ExternUID        types.String `tfsdk:"extern_uid"`
	Organization     types.String `tfsdk:"organization"`
	TwoFactorEnabled types.Bool   `tfsdk:"two_factor_enabled"`
	Note             types.String `tfsdk:"note"`
	UserProvider     types.String `tfsdk:"user_provider"`
	AvatarURL        types.String `tfsdk:"avatar_url"`
	Bio              types.String `tfsdk:"bio"`
	IsBot            types.Bool   `tfsdk:"is_bot"`
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
	EmailExactMatch  types.Bool   `tfsdk:"email_exact_match"`
}

func (d *gitlabUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *gitlabUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_user`" + ` data source allows details of a user to be retrieved by either the user ID, username or email address.

-> Some attributes might not be returned depending on if you're an admin or not.

~> When using the ` + "`email`" + ` attribute, an exact match is not guaranteed. The most related match will be returned. The most related match will prioritize an exact match if one is available.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/users/#get-a-single-user)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<user-id>`.",
				Computed:            true,
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.Int64{int64validator.ConflictsWith(path.MatchRoot("username"), path.MatchRoot("email"))},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the user.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("user_id"), path.MatchRoot("email"))},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The public email address of the user.",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("user_id"), path.MatchRoot("username"))},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Computed:            true,
			},
			"is_admin": schema.BoolAttribute{
				MarkdownDescription: "Whether the user is an admin.",
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
			"note": schema.StringAttribute{
				MarkdownDescription: "Admin notes for this user.",
				Computed:            true,
			},
			"user_provider": schema.StringAttribute{
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
			"is_bot": schema.BoolAttribute{
				MarkdownDescription: "Whether the user is a bot.",
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
				Optional:            true,
				Computed:            true,
			},
			"email_exact_match": schema.BoolAttribute{
				MarkdownDescription: "(Experimental) If true, returns only an exact match. Otherwise, fuzzy matching might return the closest result. If no exact match is available, the data source returns an error.",
				Optional:            true,
				Validators:          []validator.Bool{boolvalidator.ConflictsWith(path.MatchRoot("user_id"), path.MatchRoot("username"))},
			},
		},
	}
}

func (d *gitlabUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabUserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user *gitlab.User
	var err error

	tflog.Info(ctx, "Reading Gitlab user")

	if !data.UserID.IsNull() && !data.UserID.IsUnknown() {
		// Get user by id
		userID := int(data.UserID.ValueInt64())
		user, _, err = d.client.Users.GetUser(userID, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read user by id %d: %s", userID, err.Error()))
			return
		}
	} else {
		if !data.Username.IsNull() && !data.Username.IsUnknown() {
			user = d.findUserByUsername(ctx, data, &resp.Diagnostics)
		} else if !data.Email.IsNull() && !data.Email.IsUnknown() {
			// Get user by email
			// Note: Search can return multiple users potentially, but as of GitLab 16.6,
			// useing Search without "sort" will prioritize an exact match at the top
			// of the list.
			user = d.findUserByEmail(ctx, data, &resp.Diagnostics)
		} else {
			resp.Diagnostics.AddError("Missing required parameter", "One and only one of user_id, username or email must be set")
			return
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if user == nil {
		resp.Diagnostics.AddError("User not found", "No user was found matching the specified criteria")
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", user.ID))
	data.UserID = types.Int64Value(int64(user.ID))
	data.Username = types.StringValue(user.Username)
	data.Email = types.StringValue(user.Email)
	data.Name = types.StringValue(user.Name)
	data.IsAdmin = types.BoolValue(user.IsAdmin)
	data.CanCreateGroup = types.BoolValue(user.CanCreateGroup)
	data.CanCreateProject = types.BoolValue(user.CanCreateProject)
	data.ProjectsLimit = types.Int64Value(int64(user.ProjectsLimit))
	data.State = types.StringValue(user.State)
	data.External = types.BoolValue(user.External)
	data.ExternUID = types.StringValue(user.ExternUID)
	data.Organization = types.StringValue(user.Organization)
	data.TwoFactorEnabled = types.BoolValue(user.TwoFactorEnabled)
	data.Note = types.StringValue(user.Note)
	data.UserProvider = types.StringValue(user.Provider)
	data.AvatarURL = types.StringValue(user.AvatarURL)
	data.Bio = types.StringValue(user.Bio)
	data.IsBot = types.BoolValue(user.Bot)
	data.Location = types.StringValue(user.Location)
	data.Skype = types.StringValue(user.Skype)
	data.LinkedIn = types.StringValue(user.Linkedin)
	data.Twitter = types.StringValue(user.Twitter)
	data.WebsiteURL = types.StringValue(user.WebsiteURL)
	data.ThemeID = types.Int64Value(int64(user.ThemeID))
	data.ColorSchemeID = types.Int64Value(int64(user.ColorSchemeID))
	data.NamespaceID = types.Int64Value(int64(user.NamespaceID))

	if user.LastSignInAt == nil {
		data.LastSignInAt = types.StringValue("")
	} else {
		data.LastSignInAt = types.StringValue(user.LastSignInAt.String())
	}

	if user.CurrentSignInAt == nil {
		data.CurrentSignInAt = types.StringValue("")
	} else {
		data.CurrentSignInAt = types.StringValue(user.CurrentSignInAt.String())
	}

	if user.CreatedAt == nil {
		data.CreatedAt = types.StringValue("")
	} else {
		data.CreatedAt = types.StringValue(user.CreatedAt.String())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabUserDataSource) findUserByUsername(ctx context.Context, data *gitlabUserDataSourceModel, diags *diag.Diagnostics) *gitlab.User {
	username := strings.ToLower(data.Username.ValueString())
	listUsersOptions := &gitlab.ListUsersOptions{
		Username: gitlab.Ptr(username),
	}
	users, _, err := d.client.Users.ListUsers(listUsersOptions, gitlab.WithContext(ctx))
	if err != nil {
		diags.AddError("GitLab API error occurred", fmt.Sprintf("Unable to list users: %s", err.Error()))
		return nil
	}
	if len(users) != 1 {
		diags.AddError("No matching users found", fmt.Sprintf("Number of users matching username %s: %d", username, len(users)))
		return nil
	}
	return users[0]
}

func (d *gitlabUserDataSource) findUserByEmail(ctx context.Context, data *gitlabUserDataSourceModel, diags *diag.Diagnostics) *gitlab.User {
	email := strings.ToLower(data.Email.ValueString())
	listUsersOptions := &gitlab.ListUsersOptions{
		Search: gitlab.Ptr(email),
	}
	users, _, err := d.client.Users.ListUsers(listUsersOptions, gitlab.WithContext(ctx))
	if err != nil {
		diags.AddError("GitLab API error occurred", fmt.Sprintf("Unable to list users: %s", err.Error()))
		return nil
	}

	// Check we have any back from the API
	if len(users) == 0 {
		diags.AddError("No matching users found", fmt.Sprintf("No users matching email %s", email))
		return nil
	}

	// User wants an exact match, so try and find one, fail if none found
	if !data.EmailExactMatch.IsNull() && !data.EmailExactMatch.IsUnknown() && data.EmailExactMatch.ValueBool() {
		for _, v := range users {
			if strings.ToLower(v.Email) == email {
				return v
			}
		}
		diags.AddError("No exact match found", fmt.Sprintf("No users exactly matching email %s", email))
		return nil
	}

	// User doesn't want an exact match, return the first one returned from the API
	if len(users) > 1 {
		tflog.Info(ctx, "more than one user found matching defined email. Will return the first user, since this can only happen when using `search`", map[string]interface{}{
			"email": email,
		})
	}

	return users[0]
}
