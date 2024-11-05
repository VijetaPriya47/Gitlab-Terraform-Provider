package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabGroupServiceAccountAccessTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupServiceAccountAccessTokenResource{}
	_ resource.ResourceWithImportState = &gitlabGroupServiceAccountAccessTokenResource{}
)

func init() {
	registerResource(NewGitlabGroupServiceAccountAccessTokenResource)
}

func NewGitlabGroupServiceAccountAccessTokenResource() resource.Resource {
	return &gitlabGroupServiceAccountAccessTokenResource{}
}

type gitlabGroupServiceAccountAccessTokenResource struct {
	client *gitlab.Client
}

// The base Resource implementation struct
type gitlabGroupServiceAccountAccessTokenResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Group  types.String `tfsdk:"group"`
	Name   types.String `tfsdk:"name"`
	Token  types.String `tfsdk:"token"`
	UserID types.Int64  `tfsdk:"user_id"`

	// []string, or a set of types.String behind the scenes.
	Scopes []types.String `tfsdk:"scopes"`

	ExpiresAt types.String `tfsdk:"expires_at"`
	CreatedAt types.String `tfsdk:"created_at"`

	Active  types.Bool `tfsdk:"active"`
	Revoked types.Bool `tfsdk:"revoked"`
}

func (r *gitlabGroupServiceAccountAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_service_account_access_token"
}

func (r *gitlabGroupServiceAccountAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_group_service_account_access_token` + "`" + ` resource allows to manage the lifecycle of a group service account access token.

~> Use of the ` + "`timestamp()`" + ` function with expires_at will cause the resource to be re-created with every apply, it's recommended to use ` + "`plantimestamp()`" + ` or a static value instead.

~> Reading the access token status of a service account requires an admin token, even on gitlab.com. As a result, this resource will ignore permission errors when attempting to read the token status, and will rely on the values in state instead. This can lead to apply-time failures if the token configured for the provider doesn't have permissions to rotate tokens for the service account.

~> Deleting the access token requires an admin token. If an admin token is used to configure the provider, a direct delete will be performed during a ` + "`" + `destroy` + "`" + ` operation. Otherwise, the token will be rotated to destroy the old value, and the new token value will not be saved. This will cause the new token to expire eventually, functionally removing the old token.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/ee/api/group_service_accounts.html#create-a-personal-access-token-for-a-service-account-user)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group service account access token.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the group containing the service account. Must be a top level group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of a service account user.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the personal access token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: fmt.Sprintf("The scopes of the group service account access token. valid values are: %s", utils.RenderValueListForDocs(api.ValidAccessTokenScopes)),
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
					setplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOfCaseInsensitive(api.ValidPersonalAccessTokenScopes...),
					),
				},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "	The personal access token expiry date. When left blank, the token follows the standard rule of expiry for personal access tokens.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Optional: true,
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Time the token has been created, RFC3339 format.",
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token of the group service account access token. **Note**: the token is not available for imported resources.",
				Computed:            true,
				Sensitive:           true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "True if the token is active.",
				Computed:            true,
			},
			"revoked": schema.BoolAttribute{
				MarkdownDescription: "True if the token is revoked.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabGroupServiceAccountAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

func (r *gitlabGroupServiceAccountAccessTokenResource) groupServiceAccountAccessTokenToStateModel(data *gitlabGroupServiceAccountAccessTokenResourceModel, token *gitlab.PersonalAccessToken, group string) diag.Diagnostics {

	data.ID = types.StringValue(fmt.Sprintf("%s:%d:%d", group, token.UserID, token.ID))
	data.Group = types.StringValue(group)
	data.UserID = types.Int64Value(int64(token.UserID))
	data.Name = types.StringValue(token.Name)
	data.Active = types.BoolValue(token.Active)
	data.Revoked = types.BoolValue(token.Revoked)

	// Reading the token will not return a `token` value and we don't want to override what's in state when this happens
	if token.Token != "" {
		data.Token = types.StringValue(token.Token)
	}
	if token.CreatedAt != nil {
		data.CreatedAt = types.StringValue(token.CreatedAt.String())
	}
	if token.ExpiresAt != nil {
		data.ExpiresAt = types.StringValue(token.ExpiresAt.String())
	}

	// parse Scopes into []types.String
	var scopes []types.String
	for _, v := range token.Scopes {
		scopes = append(scopes, types.StringValue(v))
	}
	data.Scopes = scopes

	return nil
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupServiceAccountAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabGroupServiceAccountAccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupServiceAccountAccessTokenResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	splitedID := strings.SplitN(data.ID.ValueString(), ":", 3)
	if len(splitedID) != 3 {
		resp.Diagnostics.AddError(
			"Error parsing ID",
			"Could not parse ID into group, userID and accessTokenID",
		)
		return
	}

	group := splitedID[0]
	userID := splitedID[1]
	accessTokenID := splitedID[2]
	tflog.Debug(ctx, "Read gitlab GroupServiceAccountAccessToken", map[string]interface{}{"token_id": accessTokenID, "user_id": userID, "group": group})

	// Make sure the token ID is an int
	accessTokenIDInt, err := strconv.Atoi(accessTokenID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %q to int: %s", accessTokenID, err),
		)
		return
	}

	// Read the access token from the API
	accessToken, httpresp, err := r.client.PersonalAccessTokens.GetSinglePersonalAccessTokenByID(accessTokenIDInt, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			// The access token doesn't exist anymore; remove it.
			tflog.Debug(ctx, "AccessToken not found, removing from state", map[string]interface{}{"token_id": accessTokenID, "user_id": userID})
			resp.State.RemoveResource(ctx)
			return
		}

		// If the read comes back as a permission error, this can _sometimes_ mean a non-admin token is used, especially on gitlab.com.
		// until group owners can read service account access tokens, we will rely on the state and ignore a 401.
		if httpresp.StatusCode == http.StatusUnauthorized {
			tflog.Warn(ctx, "AccessToken read returned a 401, ignoring because service account access tokens can't be read without an admin token currently. This will make the tfplan rely on state data instead of the current API values.", map[string]interface{}{"token_id": accessTokenID, "user_id": userID})
			return
		}

		// Legit error, add a diagnostic and error
		resp.Diagnostics.AddError(
			"Error reading GitLab PersonalAccessToken",
			fmt.Sprintf("Could not read GitLab PersonalAccessToken, unexpected error: %v", err),
		)
		return
	}

	// Set the token information into state
	resp.Diagnostics.Append(r.groupServiceAccountAccessTokenToStateModel(data, accessToken, group)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupServiceAccountAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupServiceAccountAccessTokenResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// convert data.Scopes into []*string
	var scopes []string
	for _, v := range data.Scopes {
		s := v.ValueString()
		scopes = append(scopes, s)
	}

	// Create options struct
	options := &gitlab.CreateServiceAccountPersonalAccessTokenOptions{
		Name:   data.Name.ValueStringPointer(),
		Scopes: gitlab.Ptr(scopes),
	}

	// Optional attributes

	// Get the valid expiry date from the `expires_at`
	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		expiryDate, err := gitlab.ParseISOTime(data.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error determining expiry date",
				fmt.Sprintf("Could not determine expiry date: %s", err),
			)
			return
		}

		options.ExpiresAt = &expiryDate
	}

	token, _, err := r.client.Groups.CreateServiceAccountPersonalAccessToken(data.Group.ValueString(), int(data.UserID.ValueInt64()), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GitLab ProjectAccessToken",
			fmt.Sprintf("Could not create GitLab ProjectAccessToken, unexpected error: %v", err),
		)
		return
	}

	r.groupServiceAccountAccessTokenToStateModel(data, token, data.Group.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupServiceAccountAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

func (r *gitlabGroupServiceAccountAccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform state data into the model to get ID
	var data *gitlabGroupServiceAccountAccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	splitedID := strings.SplitN(data.ID.ValueString(), ":", 3)
	if len(splitedID) != 3 {
		resp.Diagnostics.AddError(
			"Error parsing ID",
			"Could not parse ID into group, userID and accessTokenID",
		)
		return
	}

	group := splitedID[0]
	userID := splitedID[1]
	accessTokenID := splitedID[2]
	tflog.Debug(ctx, "Read gitlab GroupServiceAccountAccessToken", map[string]interface{}{"token_id": accessTokenID, "user_id": userID, "group": group})

	accessTokenIDInt, err := strconv.Atoi(accessTokenID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %s to int: %s", accessTokenID, err),
		)
		return
	}

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing user ID",
			fmt.Sprintf("Could not parse user ID %s to int: %s", userID, err),
		)
		return
	}

	// If the user is an admin token, delete the group token directly. Otherwise we need to rotate it and discard the results instead.
	isAdmin, err := api.IsCurrentUserAdmin(ctx, r.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group service account access token",
			fmt.Sprintf("failed to check if user is admin to determine if the token should be directly deleted: %v", err),
		)
		return
	}

	if isAdmin {
		tflog.Debug(ctx, "[DEBUG] Deleting GroupServiceAccountAccessToken - direct delete due to admin privileges", map[string]interface{}{"token_id": accessTokenID, "user_id": userID})
		_, err = r.client.PersonalAccessTokens.RevokePersonalAccessToken(accessTokenIDInt, gitlab.WithContext(ctx))
	} else {
		tflog.Debug(ctx, "[DEBUG] Deleting GroupServiceAccountAccessToken - this involves rotating the token then discarding the returned result so it can't be used. This is done because the only way to fully delete a token is to have an admin token, or be signed in with the service account.", map[string]interface{}{"token_id": accessTokenID, "user_id": userID})
		_, _, err = r.client.Groups.RotateServiceAccountPersonalAccessToken(group, userIDInt, accessTokenIDInt, gitlab.WithContext(ctx))
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group service account access token",
			fmt.Sprintf("Could not delete group service account access token, unexpected error: %v", err),
		)
		return
	}
}
