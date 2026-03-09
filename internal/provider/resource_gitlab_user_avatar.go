package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabUserAvatarResource{}
	_ resource.ResourceWithConfigure   = &gitlabUserAvatarResource{}
	_ resource.ResourceWithImportState = &gitlabUserAvatarResource{}
	_ resource.ResourceWithModifyPlan  = &gitlabUserAvatarResource{}
)

func init() {
	registerResource(NewGitLabUserAvatarResource)
}

func NewGitLabUserAvatarResource() resource.Resource {
	return &gitlabUserAvatarResource{}
}

type gitlabUserAvatarResource struct {
	client          *gitlab.Client
	newGitLabClient GitLabClientFactory
}

// The base Resource implementation struct
type gitlabUserAvatarResourceModel struct {
	ID         types.String `tfsdk:"id"`
	UserId     types.Int64  `tfsdk:"user_id"`
	Token      types.String `tfsdk:"token"`
	Avatar     types.String `tfsdk:"avatar"`
	AvatarHash types.String `tfsdk:"avatar_hash"`
	AvatarURL  types.String `tfsdk:"avatar_url"`
}

func (r *gitlabUserAvatarResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_avatar"
}

func (r *gitlabUserAvatarResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_user_avatar` + "`" + ` resource allows users to manage the lifecycle of a user avatar.
The resource can also be used to set the avatar on project or group access tokens, as well as on service accounts.

~> The ` + "`token`" + ` attribute is optional only when the GitLab token used by the provider has an administrator scope, as this allows an administrator to manage user avatars.

~> The provided ` + "`token`" + ` must have the ` + "`api`" + ` scope in order to set the avatar.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/users/#upload-an-avatar-for-yourself)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the user avatar resource.  It will always be equal to the `user_id`.",
				Computed:            true,
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user.",
				Required:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The access token of the user. If this field is omitted, a GitLab token with administrator scope is required to manage the avatar for the specified user. **Note**: the token is not available for imported resources.",
				Optional:            true,
				Sensitive:           true,
			},
			"avatar": schema.StringAttribute{
				MarkdownDescription: "A local path to the avatar image to upload. **Note**: the avatar is not available for imported resources.",
				Required:            true,
			},
			"avatar_hash": schema.StringAttribute{
				MarkdownDescription: "The hash of the avatar image.  This is used to track changes to the avatar image if the image contents change but the image name remains the same. Use `filesha256(\"path/to/avatar.png\")` whenever possible.",
				Optional:            true,
			},
			"avatar_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the avatar image.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabUserAvatarResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
	r.newGitLabClient = resourceData.NewGitLabClient
}

func (r *gitlabUserAvatarResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var data *gitlabUserAvatarResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no check for token permissions is needed")
		return
	}

	// if token wasn't provded, check that the user is an admin
	if data.Token.ValueStringPointer() == nil {
		// Check if the current user has admin permissions
		isAdmin, err := api.IsCurrentUserAdmin(ctx, r.client)
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to check current user permissions: %s", err.Error()))
			return
		}
		// when token isn't provided, only admin users are allowed to set avatars for users
		if !isAdmin {
			resp.Diagnostics.AddError("Access Denied", "Current user is not an admin.  The token must be provided.")
			return
		}
	}
}

func (r *gitlabUserAvatarResource) userAvatarToStateModel(data *gitlabUserAvatarResourceModel, user *gitlab.User) diag.Diagnostics {
	data.ID = types.StringValue(strconv.FormatInt(user.ID, 10))
	data.UserId = types.Int64Value(user.ID)
	// set the avatar url
	data.AvatarURL = types.StringValue(user.AvatarURL)

	return nil
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabUserAvatarResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabUserAvatarResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabUserAvatarResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userId, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to convert resource ID: %s", err.Error()))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Read gitlab avatar information for user %d", userId))

	// Read the avatar data
	user, _, err := r.client.Users.GetUser(userId, &gitlab.GetUserOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			// The access token doesn't exist anymore; remove the avatar resource.
			tflog.Debug(ctx, fmt.Sprintf("GitLab user for user ID %d not found, removing from state", userId))
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read user by id %d: %s", userId, err.Error()))
		return
	}

	// Set the avatar information into state
	resp.Diagnostics.Append(r.userAvatarToStateModel(data, user)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabUserAvatarResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabUserAvatarResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Add the avatar to the user
	user, err := r.uploadUserAvatar(ctx, data.UserId.ValueInt64(), data.Token.ValueString(), data.Avatar.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting avatar for GitLab user",
			fmt.Sprintf("Could not create avatar for GitLab user, unexpected error: %v", err),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(data.UserId.ValueInt64(), 10))

	r.userAvatarToStateModel(data, user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabUserAvatarResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabUserAvatarResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update the avatar on the user
	user, err := r.uploadUserAvatar(ctx, data.UserId.ValueInt64(), data.Token.ValueString(), data.Avatar.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating avatar for GitLab user",
			fmt.Sprintf("Could not update avatar for GitLab user, unexpected error: %v", err),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(data.UserId.ValueInt64(), 10))

	r.userAvatarToStateModel(data, user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabUserAvatarResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// there is no way to remove an avatar once set, even passing an empty string does not reset it to the original gravatar image
	tflog.Debug(ctx, "[DEBUG] destroying the user avatar does not do anything as the API currently doesn't support removing/resetting the avatar.")
	resp.Diagnostics.AddWarning("Could not remove the user avatar", "The API currently does not support removing/resetting the user avatar to its original value")
	resp.State.RemoveResource(ctx)
}

// Upload an avatar to the user using the access token
func (r *gitlabUserAvatarResource) uploadUserAvatar(ctx context.Context, userId int64, accessToken string, avatar string) (user *gitlab.User, err error) {
	avatarFile, err := os.Open(avatar)
	if err != nil {
		return nil, fmt.Errorf("unable to open avatar file %s: %s", avatar, err)
	}
	defer func() {
		cerr := avatarFile.Close()
		if err == nil {
			err = cerr
		}
	}()

	// if token was provided, use it, otherwise use the current client
	if accessToken != "" {
		tokenClient, err := r.newGitLabClient(ctx, WithToken(accessToken), WithEarlyAuth(false))
		if err != nil {
			return nil, fmt.Errorf("could not create a new client with the token that exists in state. The provider's token can't upload an avatar: %v", err)
		}

		user, _, err = tokenClient.Users.UploadAvatar(avatarFile, avatar, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("could not upload avatar on UserAvatar, unexpected error: %v", err)
		}

		// the API call only returns the avatar url, so set the user id to the value provided
		user.ID = userId
	} else {
		// setting user avatar using admin token from provider
		options := &gitlab.ModifyUserOptions{
			Avatar: &gitlab.UserAvatar{
				Filename: avatar,
				Image:    avatarFile,
			},
		}
		user, _, err = r.client.Users.ModifyUser(userId, options, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("could not upload avatar on user %d, unexpected error: %v", userId, err)
		}
	}

	return
}
