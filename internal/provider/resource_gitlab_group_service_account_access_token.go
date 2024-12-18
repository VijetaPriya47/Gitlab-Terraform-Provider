package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
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
	gitlab "gitlab.com/gitlab-org/api/client-go"
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
	client          *gitlab.Client
	newGitLabClient GitLabClientFactory
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

	RotationConfiguration *gitlabServiceActionAccessTokenRotationConfiguration `tfsdk:"rotation_configuration"`
}

// The struct for rotation configurations. Used when the provider is auto
// rotating tokens, and its use conflicts with `expires_at`
type gitlabServiceActionAccessTokenRotationConfiguration struct {
	RotateBeforeDays types.Int64 `tfsdk:"rotate_before_days"`
}

func (r *gitlabGroupServiceAccountAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_service_account_access_token"
}

func (r *gitlabGroupServiceAccountAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_group_service_account_access_token` + "`" + ` resource allows to manage the lifecycle of a group service account access token.

~> Use of the ` + "`timestamp()`" + ` function with expires_at will cause the resource to be re-created with every apply, it's recommended to use ` + "`plantimestamp()`" + ` or a static value instead.

~> Reading the access token status of a service account requires an admin token or a top-level group owner token on gitlab.com. As a result, this resource will ignore permission errors when attempting to read the token status, and will rely on the values in state instead. This can lead to apply-time failures if the token configured for the provider doesn't have permissions to rotate tokens for the service account.

~> Use ` + "`rotation_configuration`" + ` to automatically rotate tokens instead of using ` + "`timestamp()`" + ` as timestamp will cause changes with every plan. ` + "`terraform apply`" + ` must still be run to rotate the token.

~> Due to a limitation in the API, the ` + "`rotation_configuration`" + ` is unable to set the new expiry date. Instead, when the resource is created, it will default the expiry date to 7 days in the future. On each subsequent apply, the new expiry will be 7 days from the date of the apply. 

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
				MarkdownDescription: "The service account access token expiry date. When left blank, the token follows the standard rule of expiry for personal access tokens.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("rotation_configuration")),
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
			"rotation_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration for when to rotate a token automatically. Will not rotate a token until `terraform apply` is run.",
				Optional:            true,
				Validators: []validator.Object{
					objectvalidator.ExactlyOneOf(path.MatchRoot("expires_at")),
				},

				// Rotation attributes
				Attributes: map[string]schema.Attribute{
					"rotate_before_days": schema.Int64Attribute{
						MarkdownDescription: "The duration (in days) before the expiration when the token should be rotated. As an example, if set to 7 days, the token will rotate 7 days before the expiration date, but only when `terraform apply` is run in that timeframe.",
						Required:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
						},
					},
				},
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

	rd := req.ProviderData.(*GitLabResourceData)
	r.client = rd.Client
	r.newGitLabClient = rd.NewGitLabClient
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

// Use the `ModifyPlan` to determine if we need to rotate the `token` associated to this
// resource by checking the date that's set in the `expires_at` field is less than the current date.
func (r *gitlabGroupServiceAccountAccessTokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var planData, stateData *gitlabGroupServiceAccountAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if planData == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no check for token rotation is needed")
		return
	}

	if stateData != nil && stateData.RotationConfiguration != nil {
		expiresAt := stateData.ExpiresAt.ValueString()
		expiresAtTime, err := time.Parse(api.Iso8601, expiresAt)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing expiry date",
				fmt.Sprintf("Could not parse expiry date %q: %s", expiresAt, err),
			)
			return
		}

		// Subtract the rotation days
		// This is done using `Add` because it returns "time.Time" instead of `Sub` which returns time.Duration. For some reason.
		gapTime := expiresAtTime.Add(-time.Duration(planData.RotationConfiguration.RotateBeforeDays.ValueInt64()) * 24 * time.Hour)

		if gapTime.Before(api.CurrentTime()) {
			planData.ExpiresAt = types.StringUnknown()
			planData.ID = types.StringUnknown()
			planData.Token = types.StringUnknown()
			planData.CreatedAt = types.StringUnknown()

			// Logs for assisting with support
			tflog.Debug(ctx, "[ServiceAccountAccessToken] Rotation is required, setting plan data", map[string]interface{}{
				"expires_at": stateData.ExpiresAt.ValueString(),
				"group":      planData.Group.ValueString(),
				"name":       planData.Name.ValueString(),
			})

			resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
		}
	}
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

		// If the read comes back as a permission error, this can _sometimes_ mean a non-admin/owner token is used, especially on gitlab.com.
		// until group owners can read service account access tokens, we will rely on the state and ignore a 401.
		if httpresp.StatusCode == http.StatusUnauthorized {
			tflog.Warn(ctx, "AccessToken read returned a 401, ignoring because service account access tokens can't be read without an admin (top-level group owner on gitlab.com) token currently. This will make the tfplan rely on state data instead of the current API values.", map[string]interface{}{"token_id": accessTokenID, "user_id": userID})
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
	} else if !data.RotationConfiguration.RotateBeforeDays.IsNull() {
		// Default the expires at to 7 days in the future if rotation is configured.
		expiryDate := api.CurrentTime().Add(time.Duration(7) * 24 * time.Hour)
		expiryIsoTime, err := gitlab.ParseISOTime(expiryDate.Format(api.Iso8601))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error determining default expiry date",
				fmt.Sprintf("Could not determine default expiry date: %s", err),
			)
		}

		options.ExpiresAt = &expiryIsoTime
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
	// Update only triggers when `expires_at` is updated. Anything else should trigger
	// a "replace" operation which will destroy/create.
	var planData, stateData *gitlabGroupServiceAccountAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	splitedID := strings.SplitN(stateData.ID.ValueString(), ":", 3)
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

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing user ID",
			fmt.Sprintf("Could not parse user ID %s to int: %s", userID, err),
		)
		return
	}

	accessTokenIDInt, err := strconv.Atoi(accessTokenID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %s to int: %s", accessTokenID, err),
		)
		return
	}

	token, _, err := r.client.Groups.RotateServiceAccountPersonalAccessToken(group, userIDInt, accessTokenIDInt, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error rotating GitLab GroupServiceAccountAccessToken",
			fmt.Sprintf("Could not rotate GitLab GroupServiceAccountAccessToken, unexpected error: %v", err),
		)
	}

	r.groupServiceAccountAccessTokenToStateModel(planData, token, planData.Group.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
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

	// If the user is an admin/owner token, delete the group token directly. Otherwise we need to rotate it and discard the results instead.
	isAdmin, err := api.IsCurrentUserAdmin(ctx, r.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group service account access token",
			fmt.Sprintf("failed to check if user is admin (top-level group owner on gitlab.com) to determine if the token should be directly deleted: %v", err),
		)
		return
	}

	if isAdmin {
		tflog.Debug(ctx, "[DEBUG] Deleting GroupServiceAccountAccessToken - direct delete due to admin (top-level group owner on gitlab.com) privileges", map[string]interface{}{"token_id": accessTokenID, "user_id": userID})
		_, err = r.client.PersonalAccessTokens.RevokePersonalAccessToken(accessTokenIDInt, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting group service account access token",
				fmt.Sprintf("Could not delete service access token using the provided admin (top-level group owner on gitlab.com) token: %v", err),
			)
			return
		}
	} else {

		// Create a new client from the token that exists in state, and use that client to delete the existing token.
		tflog.Debug(ctx, "[DEBUG] Deleting GroupServiceAccountAccessToken - This will use the token that's in state to delete the token instead of relying on the provier's configured token.", map[string]interface{}{"token_id": accessTokenID, "user_id": userID})
		tokenClient, err := r.newGitLabClient(ctx, WithToken(data.Token.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting group service account access token",
				fmt.Sprintf("Could not create a new client with the token that exists in state. The provider's token can't delete the service account access token because it's not an admin (top-level group owner on gitlab.com): %v", err),
			)
			return
		}

		_, err = tokenClient.PersonalAccessTokens.RevokePersonalAccessTokenSelf(gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting group service account access token",
				fmt.Sprintf("Unable to delete service account access token using the token in state. This may indicate the token has already expired: %v", err),
			)
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group service account access token",
			fmt.Sprintf("Could not delete group service account access token, unexpected error: %v", err),
		)
		return
	}
}
