package provider

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
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
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabPersonalAccessTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabPersonalAccessTokenResource{}
	_ resource.ResourceWithImportState = &gitlabPersonalAccessTokenResource{}
	_ resource.ResourceWithModifyPlan  = &gitlabPersonalAccessTokenResource{}
)

func init() {
	registerResource(NewGitLabPersonalAccessTokenResource)
}

func NewGitLabPersonalAccessTokenResource() resource.Resource {
	return &gitlabPersonalAccessTokenResource{}
}

type gitlabPersonalAccessTokenResource struct {
	client          *gitlab.Client
	newGitLabClient GitLabClientFactory
}

// The base Resource implementation struct
type gitlabPersonalAccessTokenResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Token       types.String `tfsdk:"token"`
	UserId      types.Int64  `tfsdk:"user_id"`

	// []string, or a set of types.String behind the scenes.
	Scopes types.Set `tfsdk:"scopes"`

	ExpiresAt                  types.String `tfsdk:"expires_at"`
	CreatedAt                  types.String `tfsdk:"created_at"`
	ValidatePastExpirationDate types.Bool   `tfsdk:"validate_past_expiration_date"`

	Active  types.Bool `tfsdk:"active"`
	Revoked types.Bool `tfsdk:"revoked"`

	// Defined in resource_gitlab_project_access_token.go
	RotationConfiguration *utils.GitlabAccessTokenRotationConfiguration `tfsdk:"rotation_configuration"`
}

// Implement RotatableToken interface - Getters
func (m *gitlabPersonalAccessTokenResourceModel) GetExpiresAt() types.String {
	return m.ExpiresAt
}

func (m *gitlabPersonalAccessTokenResourceModel) GetExpirationDays() types.Int64 {
	if m.RotationConfiguration == nil {
		return types.Int64Null()
	}
	return m.RotationConfiguration.ExpirationDays
}

func (m *gitlabPersonalAccessTokenResourceModel) GetRotateBeforeDays() types.Int64 {
	if m.RotationConfiguration == nil {
		return types.Int64Null()
	}
	return m.RotationConfiguration.RotateBeforeDays
}

func (m *gitlabPersonalAccessTokenResourceModel) HasRotationConfiguration() bool {
	return m.RotationConfiguration != nil
}

func (m *gitlabPersonalAccessTokenResourceModel) GetLogPrefix() string {
	return "PersonalAccessToken"
}

// Implement RotatableToken interface - Setters
func (m *gitlabPersonalAccessTokenResourceModel) SetExpiresAt(v types.String) {
	m.ExpiresAt = v
}

func (m *gitlabPersonalAccessTokenResourceModel) SetID(v types.String) {
	m.ID = v
}

func (m *gitlabPersonalAccessTokenResourceModel) SetToken(v types.String) {
	m.Token = v
}

func (m *gitlabPersonalAccessTokenResourceModel) SetCreatedAt(v types.String) {
	m.CreatedAt = v
}

func (r *gitlabPersonalAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_personal_access_token"
}

func (r *gitlabPersonalAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_personal_access_token` + "`" + ` resource manages the lifecycle of a personal access token.

-> This resource requires administration privileges.

~> Use of the ` + "`timestamp()`" + ` function with expires_at will cause the resource to be re-created with every apply, it's recommended to use ` + "`plantimestamp()`" + ` or a static value instead.

~> Observability scopes are in beta and may not work on all instances. See more details in [the documentation](https://docs.gitlab.com/development/tracing/)

~> Use ` + "`rotation_configuration`" + ` to automatically rotate tokens instead of using ` + "`timestamp()`" + ` as timestamp will cause changes with every plan. ` + "`terraform apply`" + ` must still be run to rotate the token.

~> Due to [Automatic reuse detection](https://docs.gitlab.com/api/personal_access_tokens/#automatic-reuse-detection) it's possible that a new Personal Access Token will immediately be revoked. Check if an old process using the old token is running if this happens.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/personal_access_tokens/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the personal access token.",
				Computed:            true,
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user.",
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
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the personal access token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Optional: true,
				Computed: true,
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: fmt.Sprintf("The scopes of the personal access token. valid values are: %s", utils.RenderValueListForDocs(api.ValidPersonalAccessTokenScopes)),
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
					setvalidator.SizeAtLeast(1),
				},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "When the token will expire, YYYY-MM-DD format. Is automatically set when `rotation_configuration` is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("rotation_configuration")),
				},
				Optional: true,
				Computed: true,
			},
			"validate_past_expiration_date": schema.BoolAttribute{
				MarkdownDescription: "Wether to validate if the expiration date is in the future.",
				Optional:            true,
				Computed:            true,
				// Default can't be applied even though it seems like it may be desired. This is because
				// it will cause all users with existing tokens to force an apply, making it not truly a
				// "breaking" change, but certainly an inconvenient one.
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Time the token has been created, RFC3339 format.",
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token of the personal access token. **Note**: the token is not available for imported resources.",
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
					objectvalidator.ConflictsWith(path.MatchRoot("expires_at")),
				},

				// Rotation attributes
				Attributes: map[string]schema.Attribute{
					"expiration_days": schema.Int64Attribute{
						MarkdownDescription: "The duration (in days) the new token should be valid for.",
						Required:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
						},
					},

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
func (r *gitlabPersonalAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
	r.newGitLabClient = resourceData.NewGitLabClient
}

func (r *gitlabPersonalAccessTokenResource) personalAccessTokenToStateModel(ctx context.Context, data *gitlabPersonalAccessTokenResourceModel, token *gitlab.PersonalAccessToken, userId int64) diag.Diagnostics {
	data.UserId = types.Int64Value(userId)
	data.Name = types.StringValue(token.Name)
	data.Description = types.StringValue(token.Description)
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
	} else {
		// This explicit null is required when token expiration is allowed to be null
		// which can happen in self-hosted instances
		data.ExpiresAt = types.StringNull()
	}

	// parse Scopes into types.Set
	scopesSet, diags := types.SetValueFrom(ctx, types.StringType, token.Scopes)
	if diags.HasError() {
		return diags
	}
	data.Scopes = scopesSet

	return nil
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabPersonalAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Use the `ModifyPlan` to determine if we need to rotate the `token` associated to this
// resource, by checking the date that's set in the `expires_at` field is less than the `rotate_before_days`
// value.
func (r *gitlabPersonalAccessTokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Retrieve the plan data to start with
	var planData, stateData *gitlabPersonalAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	// Now retrieve the `state` values instead of plan, because we need to get the expiry date from the state.
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if planData == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no check for token rotation is needed")
		return
	}

	// Check whether the token was revoked.
	// Re-create it if it's been revoked externally, to ensure its available.
	if stateData != nil && stateData.Revoked.ValueBool() {
		r.modifyPlanRevoked(ctx, stateData, planData, resp)
		return
	}

	// Use centralized rotation logic
	// Note: stateData must be converted to a utils.RotatableToken explicitly to
	// avoid passing a typed nil (which would not compare equal to nil in the interface).
	var stateForRotation utils.RotatableToken
	if stateData != nil {
		stateForRotation = stateData
	}
	shouldRotate, err := utils.ShouldRotateToken(
		ctx,
		planData,
		stateForRotation,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error determining rotation",
			fmt.Sprintf("Could not determine if token needs rotation: %s", err),
		)
		return
	}

	if shouldRotate && stateData != nil {
		// Apply rotation using centralized logic
		planModified, err := utils.ApplyTokenRotation(ctx, planData, planData.RotationConfiguration, stateData, nil)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error applying token rotation",
				fmt.Sprintf("Could not apply token rotation: %s", err),
			)
			return
		}

		if planModified {
			resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
		}
	}
}

// modifyPlanRevoked handles token rotation if the token has been revoked externally.
func (r *gitlabPersonalAccessTokenResource) modifyPlanRevoked(ctx context.Context, stateData *gitlabPersonalAccessTokenResourceModel, planData *gitlabPersonalAccessTokenResourceModel, resp *resource.ModifyPlanResponse) {
	// If it has an expiration date, check if it's in the future
	if stateData.RotationConfiguration == nil && !stateData.ExpiresAt.IsNull() && !stateData.ExpiresAt.IsUnknown() {
		expiryDate, err := time.Parse(api.Iso8601, stateData.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing expiry date",
				fmt.Sprintf("Could not parse expiry date %s: %s", stateData.ExpiresAt.ValueString(), err),
			)
			return
		}

		// If the expiry date is in the past, do not recreate the token.
		if expiryDate.Before(api.CurrentTime()) {
			tflog.Debug(ctx, "[gitlab_personal_access_token] Token has been revoked externally but was expired anyway, no recreation needed")
			return
		}
	}

	tflog.Debug(ctx, "[gitlab_personal_access_token] Token has been revoked externally and is still needed, marking for recreation")

	// Tell Terraform that a change to the revoked attribute requires replacing the resource
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("revoked"))

	// Calculate new expiration date
	expiryDate, _, err := utils.DetermineExpiryDate(planData.ExpiresAt, planData.RotationConfiguration, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error determining new expiry date",
			fmt.Sprintf("Could not determine new expiry date: %s", err),
		)
		return
	}

	// Set the planned values
	planData.ExpiresAt = expiryDate
	planData.Revoked = types.BoolValue(false) // Expect the new token to be not revoked

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
}

func (r *gitlabPersonalAccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabPersonalAccessTokenResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// get the user and tokenID from the resource ID
	userId, accessTokenId, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing ID",
			"Could not parse ID into userId and accessTokenId",
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Read gitlab PersonalAccessToken %s, user ID %s", accessTokenId, userId))

	// Make sure the user ID is an int
	userIdInt, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing user ID",
			fmt.Sprintf("Could not parse user ID %q to int64: %s", userId, err),
		)
		return
	}

	// Make sure the token ID is an int
	accessTokenIdInt, err := strconv.ParseInt(accessTokenId, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %q to int: %s", accessTokenId, err),
		)
		return
	}

	// Read the access token from the API
	personalAccessToken, _, err := r.client.PersonalAccessTokens.GetSinglePersonalAccessTokenByID(accessTokenIdInt, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			// The access token doesn't exist anymore; remove it.
			tflog.Debug(ctx, fmt.Sprintf("GitLab PersonalAccessToken %s, user ID %s not found, removing from state", accessTokenId, userId))
			resp.State.RemoveResource(ctx)
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
	resp.Diagnostics.Append(r.personalAccessTokenToStateModel(ctx, data, personalAccessToken, userIdInt)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPersonalAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabPersonalAccessTokenResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// convert data.Scopes into []string
	var scopes []string
	resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &scopes, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create options struct
	options := &gitlab.CreatePersonalAccessTokenOptions{
		Name:   data.Name.ValueStringPointer(),
		Scopes: gitlab.Ptr(scopes),
	}

	// Optional attributes
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}

	// // Get the valid expiry date from the `expires_at`
	_, expiryISOTime, err := utils.DetermineExpiryDate(data.ExpiresAt, data.RotationConfiguration, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error determining expiry date",
			fmt.Sprintf("Could not determine expiry date: %s", err),
		)
		return
	}

	// Use the returned gitlab.ISOTime directly for API call
	var expiryDatePtr *gitlab.ISOTime = expiryISOTime
	if expiryISOTime != nil && !data.ValidatePastExpirationDate.IsNull() && data.ValidatePastExpirationDate.ValueBool() {
		err := utils.ValidateISOTimeExpiryDate(*expiryISOTime)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating GitLab PersonalAccessToken",
				err.Error(),
			)
			return
		}
	} else {
		// Default to `false` if it's not set in the config/plan.
		data.ValidatePastExpirationDate = types.BoolValue(false)
	}

	options.ExpiresAt = expiryDatePtr

	token, _, err := r.client.Users.CreatePersonalAccessToken(data.UserId.ValueInt64(), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GitLab PersonalAccessToken",
			fmt.Sprintf("Could not create GitLab PersonalAccessToken, unexpected error: %v", err),
		)
		return
	}

	// Set the ID for the resource
	data.ID = types.StringValue(fmt.Sprintf("%d:%d", data.UserId.ValueInt64(), token.ID))

	r.personalAccessTokenToStateModel(ctx, data, token, data.UserId.ValueInt64())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPersonalAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state *gitlabPersonalAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Read the token and user ID from state since it may be `unknown` in the plan.
	userId, patId, err := utils.ParseTwoPartID(state.ID.ValueString())
	patIdInt, parseErrPat := strconv.ParseInt(patId, 10, 64)
	userIdInt, parseErrUserId := strconv.ParseInt(userId, 10, 64)
	if joinedErr := errors.Join(err, parseErrPat, parseErrUserId); joinedErr != nil {
		resp.Diagnostics.AddError(
			"Error parsing resource ID",
			fmt.Sprintf("Could not parse resource ID %s into two parts properly", data.ID.ValueString()),
		)
		return
	}

	// since modifyplan has determined the expiration date, simply retrieve it from the plan instead of re-calculating it.
	// re-calculating it here could result in a different value from the plan if the plan is run on a different date than
	// the apply, causing a "provider error" message to be sent to the user
	var expiresAtPtr *gitlab.ISOTime = nil
	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		expiresAt, err := gitlab.ParseISOTime(data.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing expiry date",
				fmt.Sprintf("Could not parse expiry date %s: %s", data.ExpiresAt.ValueString(), err),
			)
			return
		}
		expiresAtPtr = &expiresAt
	}

	if !data.ValidatePastExpirationDate.IsNull() && data.ValidatePastExpirationDate.ValueBool() && expiresAtPtr != nil {
		err := utils.ValidateISOTimeExpiryDate(*expiresAtPtr)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating GitLab PersonalAccessToken",
				err.Error(),
			)
			return
		}
	} else {
		if data.ValidatePastExpirationDate.IsNull() || data.ValidatePastExpirationDate.IsUnknown() {
			// If we have a known value, accept that so we don't error with inconsistent values
			data.ValidatePastExpirationDate = state.ValidatePastExpirationDate
		}
		// If we're still unknown, set to False
		if data.ValidatePastExpirationDate.IsUnknown() {
			data.ValidatePastExpirationDate = types.BoolValue(false)
		}
	}

	// find out whether self_rotate is one of the scopes
	var selfRotate bool
	var scopes []string
	resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &scopes, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if slices.Contains(scopes, "self_rotate") {
		selfRotate = true
	}

	// update with a service account access token means rotate it
	var token *gitlab.PersonalAccessToken
	if selfRotate {
		tflog.Debug(ctx, "Found `self_rotate` in scopes; attempting to use the self-rotate method to update the token", map[string]interface{}{
			"user":           userId,
			"new_expires_at": expiresAtPtr,
			"scopes":         data.Scopes,
		})

		token, err = r.rotateTokenSelf(ctx, state.Token.ValueString(), expiresAtPtr)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error self rotating GitLab PersonalAccessToken",
				fmt.Sprintf("Could not self rotate GitLab PersonalAccessToken, unexpected error: %v", err),
			)
			return
		}
	} else {
		options := &gitlab.RotatePersonalAccessTokenOptions{}

		if expiresAtPtr != nil {
			options.ExpiresAt = expiresAtPtr
		}

		token, _, err = r.client.PersonalAccessTokens.RotatePersonalAccessTokenByID(patIdInt, options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error rotating GitLab PersonalAccessToken",
				fmt.Sprintf("Could not rotate GitLab PersonalAccessToken, unexpected error: %v", err),
			)
			return
		}
	}

	// Updating an access token changes the primary key, so we need to re-set the ID of the resource
	data.ID = types.StringValue(utils.BuildTwoPartID(gitlab.Ptr(strconv.FormatInt(data.UserId.ValueInt64(), 10)), gitlab.Ptr(strconv.FormatInt(token.ID, 10))))

	r.personalAccessTokenToStateModel(ctx, data, token, userIdInt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPersonalAccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform state data into the model to get ID
	var data *gitlabPersonalAccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	userId, patId, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing resource ID",
			fmt.Sprintf("Could not parse resource ID %s into two parts properly", data.ID.ValueString()),
		)
		return
	}

	personalAccessTokenID, err := strconv.ParseInt(patId, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %s to int: %s", patId, err),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Deleting PersonalAccessToken %d from user %s", personalAccessTokenID, userId))
	_, err = r.client.PersonalAccessTokens.RevokePersonalAccessTokenByID(personalAccessTokenID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting personal access token",
			fmt.Sprintf("Could not delete personal access token, unexpected error: %v", err),
		)
		return
	}
}

// Rotates the token using the token itself. Only works if the token has the `self_rotate` scope.
func (r *gitlabPersonalAccessTokenResource) rotateTokenSelf(ctx context.Context, originalToken string, expiresAt *gitlab.ISOTime) (*gitlab.PersonalAccessToken, error) {
	tokenClient, err := r.newGitLabClient(ctx, WithToken(originalToken), WithEarlyAuth(false))
	if err != nil {
		return nil, fmt.Errorf("Could not create a new client with the token that exists in state. The provider's token can't rotate the personal access token: %v", err)
	}

	opt := &gitlab.RotatePersonalAccessTokenOptions{}

	if expiresAt != nil {
		opt.ExpiresAt = expiresAt
	}

	token, _, err := tokenClient.PersonalAccessTokens.RotatePersonalAccessTokenSelf(opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("Could not rotate GitLab PersonalAccessToken, unexpected error: %v", err)
	}

	return token, nil
}
