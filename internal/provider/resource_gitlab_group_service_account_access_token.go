package provider

import (
	"context"
	"fmt"
	"slices"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
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
	_ resource.Resource                   = &gitlabGroupServiceAccountAccessTokenResource{}
	_ resource.ResourceWithConfigure      = &gitlabGroupServiceAccountAccessTokenResource{}
	_ resource.ResourceWithImportState    = &gitlabGroupServiceAccountAccessTokenResource{}
	_ resource.ResourceWithValidateConfig = &gitlabGroupServiceAccountAccessTokenResource{}
)

const DEFAULT_EXPIRATION_DAYS = 7

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
	Scopes types.Set `tfsdk:"scopes"`

	ExpiresAt                  types.String `tfsdk:"expires_at"`
	CreatedAt                  types.String `tfsdk:"created_at"`
	ValidatePastExpirationDate types.Bool   `tfsdk:"validate_past_expiration_date"`

	Active  types.Bool `tfsdk:"active"`
	Revoked types.Bool `tfsdk:"revoked"`

	RotationConfiguration *utils.GitlabAccessTokenRotationConfiguration `tfsdk:"rotation_configuration"`
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

~> Due to a limitation in the API, the ` + "`rotation_configuration`" + ` is unable to set the new expiry date before GitLab 17.9. Instead, when the resource is created, it will default the expiry date to 7 days in the future. On each subsequent apply, the new expiry will be 7 days from the date of the apply. 

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/service_accounts/#create-a-personal-access-token-for-a-group-service-account)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group service account access token.",
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
				MarkdownDescription: fmt.Sprintf("The scopes of the group service account access token. Valid values are: %s. If `self_rotate` is included, you must also provide either `expires_at` or `rotation_configuration`.", utils.RenderValueListForDocs(api.ValidPersonalAccessTokenScopes)),
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
					stringvalidator.ConflictsWith(path.MatchRoot("rotation_configuration")),
				},
				Optional: true,
				Computed: true,
			},
			"validate_past_expiration_date": schema.BoolAttribute{
				MarkdownDescription: "Wether to validate if the expiration date is in the future.",
				Optional:            true,
				Computed:            true,
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
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.MatchRoot("expires_at")),
				},

				// Rotation attributes
				Attributes: map[string]schema.Attribute{
					"expiration_days": schema.Int64Attribute{
						MarkdownDescription: "The duration (in days) the new token should be valid for.",
						Optional:            true,
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
func (r *gitlabGroupServiceAccountAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	rd := req.ProviderData.(*GitLabResourceData)
	r.client = rd.Client
	r.newGitLabClient = rd.NewGitLabClient
}

// ValidateConfig checks for if they have included self_rotate in the scopes, but no
// expires_at or rotation_configuration.
func (r *gitlabGroupServiceAccountAccessTokenResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabGroupServiceAccountAccessTokenResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if !data.ExpiresAt.IsNull() || data.RotationConfiguration != nil {
		// Token will expire and therefore can be rotated.
		return
	}

	// find out whether self_rotate is one of the scopes
	var selfRotate bool
	var scopes []string
	resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &scopes, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, scope := range scopes {
		if scope == "self_rotate" {
			selfRotate = true
			break
		}
	}

	if selfRotate {
		resp.Diagnostics.AddAttributeError(path.Root("scopes"),
			`Invalid token scopes`,
			`The token scopes include "self_rotate" but "expires_at" and "rotation_configuration" have not been set. Therefore the token will never expire and cannot be rotated.`)
	}
}

func (r *gitlabGroupServiceAccountAccessTokenResource) groupServiceAccountAccessTokenToStateModel(ctx context.Context, data *gitlabGroupServiceAccountAccessTokenResourceModel, token *gitlab.PersonalAccessToken, group string) diag.Diagnostics {
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
	} else {
		// This explicit null is required when token expiration is allowed to be null
		// which can happen in self-hosted instances
		data.ExpiresAt = types.StringNull()
	}

	// parse Scopes into []types.String
	var scopes []types.String
	for _, v := range token.Scopes {
		scopes = append(scopes, types.StringValue(v))
	}
	scopesSet, err := types.SetValueFrom(ctx, types.StringType, scopes)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Error parsing scopes into Set", fmt.Sprintf("Could not parse scopes into Set: %s", err))}
	}
	data.Scopes = scopesSet

	return nil
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupServiceAccountAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Use the `ModifyPlan` to determine if we need to rotate the `token` associated to this
// resource, by checking the date that's set in the `expires_at` field is less than the `rotate_before_days`
// value.
func (r *gitlabGroupServiceAccountAccessTokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var planData, stateData, configData *gitlabGroupServiceAccountAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if planData == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no check for token rotation is needed")
		return
	}

	validatePlanConfig(ctx, r.client, planData, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check whether the token was revoked.
	// Re-create it if it's been revoked externally, to ensure its available.
	if stateData != nil && stateData.Revoked.ValueBool() {
		r.modifyPlanRevoked(ctx, stateData, planData, resp)
		return
	}

	// Check to determine if we need to rotate the expiry date
	shouldSetExpiration := false

	if configData.ExpiresAt.IsNull() && planData.RotationConfiguration == nil && stateData != nil && !stateData.ExpiresAt.IsNull() {
		// The token already exists, but we want no expiry on it anymore, so need to set the expiry to null
		shouldSetExpiration = true

		tflog.Debug(ctx, "[ServiceAccountAccessToken] Plan says to remove expiry as expires_at and rotation_configuration are not set.", map[string]any{
			"is_state_nil": stateData == nil,
			"expires_at":   stateData.ExpiresAt.String(),
		})
	} else if stateData == nil || stateData.ExpiresAt.IsNull() || stateData.ExpiresAt.IsUnknown() || stateData.ExpiresAt != planData.ExpiresAt {
		// If expiration (or ANY state) has never been set yet (we're in a "Create" plan), ensure we calculate and set the first time.
		// This should also run if the expiration date has changed between plan and state, to ensure the ID is set to unknown.
		// Log some information for debugging later.
		expiresAt := ""
		if stateData != nil {
			expiresAt = stateData.ExpiresAt.ValueString()
		}
		tflog.Debug(ctx, "[ServiceAccountAccessToken] State is not populated, or the expires_at value is nil. Creating the token for the first time.", map[string]any{
			"is_state_nil": stateData == nil,
			"expires_at":   expiresAt,
		})

		// set token for rotation
		shouldSetExpiration = true

		// Otherwise, execute the logic if rotation configuration is present
	} else if stateData.RotationConfiguration != nil {
		// We're in an "Update" plan that already has expiration set, calculate if we need to rotate
		rotateBefore := stateData.ExpiresAt.ValueString()
		rotateBeforeTime, err := time.Parse(api.Iso8601, rotateBefore)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing rotation date",
				fmt.Sprintf("Could not parse rotation date %q: %s", rotateBefore, err),
			)
			return
		}

		// Subtract the rotation days
		// This is done using `Add` because it returns "time.Time" instead of `Sub` which returns time.Duration. For some reason.
		gapTime := rotateBeforeTime.Add(-time.Duration(planData.RotationConfiguration.RotateBeforeDays.ValueInt64()) * 24 * time.Hour)
		if gapTime.Before(api.CurrentTime()) {
			shouldSetExpiration = true
		}

		// Logs for assisting with support
		tflog.Debug(ctx, "[ServiceAccountAccessToken] State is populated, and a rotation configuration is detected. Determining if token should be rotated.", map[string]any{
			"expires_at":             rotateBefore,
			"detected_current_time":  api.CurrentTime(),
			"detected_rotation_date": gapTime,
			"rotate_before_days":     planData.RotationConfiguration.RotateBeforeDays.ValueInt64(),
			"should_rotate":          shouldSetExpiration,
		})
	}

	if shouldSetExpiration {
		// We need to re-calculate the expiryDate, and set it in the plan
		fallbackExpirationDays := DEFAULT_EXPIRATION_DAYS
		expiryDate, _, err := utils.DetermineExpiryDate(planData.ExpiresAt, planData.RotationConfiguration, &fallbackExpirationDays)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error determining new expiry date",
				fmt.Sprintf("Could not determine new expiry date: %s", err),
			)
			return
		}

		// If the newly calculated expiryDate is different than what's in state, modify the plan
		// This check is required to prevent the ID being unknown on every apply with rotation_configuration even
		// if the calculated date is exactly the same as it currently is
		if stateData != nil && ((!expiryDate.IsNull() && !expiryDate.IsUnknown() && expiryDate.ValueString() != stateData.ExpiresAt.ValueString()) || (expiryDate.IsNull() && !stateData.ExpiresAt.IsNull())) {
			// Set the new expiration date in the plan
			planData.ExpiresAt = expiryDate
			// We need to re-create the token on apply because the expiration date has changed
			// Set several attributes to unknown since they will change as part of rotation
			planData.ID = types.StringUnknown()
			planData.Token = types.StringUnknown()
			planData.CreatedAt = types.StringUnknown()

			// Logs for assisting with support
			tflog.Debug(ctx, "[ServiceAccountAccessToken] Rotation is required, settings plan data", map[string]any{
				"new_expires_at": planData.ExpiresAt,
				"expires_at":     stateData.ExpiresAt.ValueString(),
				"group":          planData.Group.ValueString(),
				"name":           planData.Name.ValueString(),
			})

			resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
		}
	}
}

// modifyPlanRevoked handles token rotation if the token has been revoked externally.
func (r *gitlabGroupServiceAccountAccessTokenResource) modifyPlanRevoked(ctx context.Context, stateData *gitlabGroupServiceAccountAccessTokenResourceModel, planData *gitlabGroupServiceAccountAccessTokenResourceModel, resp *resource.ModifyPlanResponse) {
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
			tflog.Debug(ctx, "[gitlab_group_service_account_access_token] Token has been revoked externally but was expired anyway, no recreation needed")
			return
		}
	}

	tflog.Debug(ctx, "[gitlab_group_service_account_access_token] Token has been revoked externally and is still needed, marking for recreation")

	// Tell Terraform that a change to the revoked attribute requires replacing the resource
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("revoked"))

	// Calculate new expiration date
	fallbackExpirationDays := DEFAULT_EXPIRATION_DAYS
	expiryDate, _, err := utils.DetermineExpiryDate(planData.ExpiresAt, planData.RotationConfiguration, &fallbackExpirationDays)
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

// Validate that the expiration date is valid. This runs during `ModifyPlan` instead of in a `ValidateConfig` because the GitLab client hasn't been
// initialized during `ValidateConfig`. It will still fail during plan time because that's when `ModifyPlan` runs.
func validatePlanConfig(ctx context.Context, client *gitlab.Client, data *gitlabGroupServiceAccountAccessTokenResourceModel, resp *resource.ModifyPlanResponse) {
	// If expiration_days isn't set, we don't need to check the version, so return early.
	if data.RotationConfiguration == nil || data.RotationConfiguration.ExpirationDays.IsUnknown() || data.RotationConfiguration.ExpirationDays.IsNull() {
		return
	}

	// Check the GitLab version
	match, err := api.IsGitLabVersionAtLeast(ctx, client, "17.9")()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking GitLab version",
			fmt.Sprintf("Could not check GitLab version: %v", err),
		)
		return
	}
	// Error if before 17.9
	if !match {
		resp.Diagnostics.AddError(
			"Cannot use `expiration_days` with GitLab version < 17.9, the feature was not available before then.",
			"Cannot use `expiration_days` with GitLab version < 17.9, the feature was not available before then.",
		)
		return
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
	tflog.Debug(ctx, "Read gitlab GroupServiceAccountAccessToken", map[string]any{"token_id": accessTokenID, "user_id": userID, "group": group})

	// Make sure the token ID is an int
	accessTokenIDInt, err := strconv.ParseInt(accessTokenID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %q to int: %s", accessTokenID, err),
		)
		return
	}

	// Make sure the group ID is an int64
	groupIDInt, err := strconv.ParseInt(group, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing group ID",
			fmt.Sprintf("Could not parse group ID %q to int: %s", group, err),
		)
		return
	}

	// Make sure the user ID is an int64
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing user ID",
			fmt.Sprintf("Could not parse user ID %q to int: %s", userID, err),
		)
		return
	}

	// Read all the access tokens from the API
	// There is no HTTP API to get a single token by ID yet
	accessTokens, _, err := r.client.Groups.ListServiceAccountPersonalAccessTokens(groupIDInt, userIDInt, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Group or service account not found, removing from state", map[string]any{"token_id": accessTokenID, "user_id": userID})
			resp.State.RemoveResource(ctx)
		}
		resp.Diagnostics.AddError(
			"Error reading GitLab ServiceAccountPersonalAccessTokens",
			fmt.Sprintf("Could not read GitLab ServiceAccountPersonalAccessTokens, unexpected error: %v", err),
		)
		return
	}

	// Find the token we want
	var accessToken *gitlab.PersonalAccessToken
	for i := range accessTokens {
		if accessTokens[i].ID == accessTokenIDInt {
			accessToken = accessTokens[i]
			break
		}
	}
	if accessToken == nil {
		// The access token doesn't exist anymore; remove it.
		tflog.Debug(ctx, "AccessToken not found, removing from state", map[string]any{"token_id": accessTokenID, "user_id": userID})
		resp.State.RemoveResource(ctx)
		return
	}

	// Set the token information into state
	resp.Diagnostics.Append(r.groupServiceAccountAccessTokenToStateModel(ctx, data, accessToken, group)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupServiceAccountAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupServiceAccountAccessTokenResourceModel

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
	options := &gitlab.CreateServiceAccountPersonalAccessTokenOptions{
		Name:   data.Name.ValueStringPointer(),
		Scopes: gitlab.Ptr(scopes),
	}

	// Optional attributes

	// Get the valid expiry date from the `expires_at` or `rotation_configuration`
	fallbackExpirationDays := DEFAULT_EXPIRATION_DAYS
	_, expiryISOTime, err := utils.DetermineExpiryDate(data.ExpiresAt, data.RotationConfiguration, &fallbackExpirationDays)
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
				"Error creating GitLab GroupServiceAccountAccessToken",
				err.Error(),
			)
			return
		}
	} else {
		// Default to `false` if it's not set in the config/plan.
		data.ValidatePastExpirationDate = types.BoolValue(false)
	}

	options.ExpiresAt = expiryDatePtr

	token, _, err := r.client.Groups.CreateServiceAccountPersonalAccessToken(data.Group.ValueString(), data.UserID.ValueInt64(), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GitLab GroupServiceAccountAccessToken",
			fmt.Sprintf("Could not create GitLab GroupServiceAccountAccessToken, unexpected error: %v", err),
		)
		return
	}

	r.groupServiceAccountAccessTokenToStateModel(ctx, data, token, data.Group.ValueString())
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

	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing user ID",
			fmt.Sprintf("Could not parse user ID %s to int: %s", userID, err),
		)
		return
	}

	accessTokenIDInt, err := strconv.ParseInt(accessTokenID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %s to int: %s", accessTokenID, err),
		)
		return
	}

	// since modifyplan has determined the expiration date, simply retrieve it from the plan instead of re-calculating it.
	// re-calculating it here could result in a different value from the plan if the plan is run on a different date than
	// the apply, causing a "provider error" message to be sent to the user
	expiresAt, err := gitlab.ParseISOTime(planData.ExpiresAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing expiry date",
			fmt.Sprintf("Could not parse expiry date %s: %s", planData.ExpiresAt.ValueString(), err),
		)
		return
	}

	if !planData.ValidatePastExpirationDate.IsNull() && planData.ValidatePastExpirationDate.ValueBool() {
		err := utils.ValidateISOTimeExpiryDate(expiresAt)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating GitLab GroupServiceAccountAccessToken",
				err.Error(),
			)
			return
		}
	} else {
		// Default to `false` if it's not set in the config/plan.
		planData.ValidatePastExpirationDate = types.BoolValue(false)
	}

	// find out whether self_rotate is one of the scopes
	var selfRotate bool
	var scopes []string
	resp.Diagnostics.Append(planData.Scopes.ElementsAs(ctx, &scopes, true)...)
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
			"group":          group,
			"user":           userID,
			"new_expires_at": expiresAt,
			"scopes":         planData.Scopes,
		})

		token, err = r.rotateTokenSelf(ctx, stateData.Token.ValueString(), expiresAt)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error self rotating GitLab Service Account PersonalAccessToken",
				fmt.Sprintf("Could not self rotate GitLab Service Account PersonalAccessToken, unexpected error: %v", err),
			)
			return
		}
	} else {
		token, _, err = r.client.Groups.RotateServiceAccountPersonalAccessToken(group, userIDInt, accessTokenIDInt, &gitlab.RotateServiceAccountPersonalAccessTokenOptions{
			ExpiresAt: &expiresAt,
		}, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error rotating GitLab GroupServiceAccountAccessToken",
				fmt.Sprintf("Could not rotate GitLab GroupServiceAccountAccessToken, unexpected error: %v", err),
			)
			return
		}
	}

	r.groupServiceAccountAccessTokenToStateModel(ctx, planData, token, planData.Group.ValueString())
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
	tflog.Debug(ctx, "Read gitlab GroupServiceAccountAccessToken", map[string]any{"token_id": accessTokenID, "user_id": userID, "group": group})

	accessTokenIDInt, err := strconv.ParseInt(accessTokenID, 10, 64)
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
		tflog.Debug(ctx, "[DEBUG] Deleting GroupServiceAccountAccessToken - direct delete due to admin (top-level group owner on gitlab.com) privileges", map[string]any{"token_id": accessTokenID, "user_id": userID})
		_, err = r.client.PersonalAccessTokens.RevokePersonalAccessToken(accessTokenIDInt, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting group service account access token",
				fmt.Sprintf("Could not delete service access token using the provided admin (top-level group owner on gitlab.com) token: %v", err),
			)
			return
		}
	} else {
		expiresAt := data.ExpiresAt.ValueString()
		expiresAtTime, err := time.Parse(api.Iso8601, expiresAt)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing expiry date",
				fmt.Sprintf("Could not parse expiry date %q: %s", expiresAt, err),
			)
			return
		}

		if expiresAtTime.Before(api.CurrentTime()) {
			resp.Diagnostics.AddWarning("Deleting an already expired token, removing from state.", fmt.Sprintf("Token expired on %s", expiresAt))
			return
		}

		// Create a new client from the token that exists in state, and use that client to delete the existing token.
		tflog.Debug(ctx, "[DEBUG] Deleting GroupServiceAccountAccessToken - This will use the token that's in state to delete the token instead of relying on the provier's configured token.", map[string]any{"token_id": accessTokenID, "user_id": userID})
		tokenClient, err := r.newGitLabClient(ctx, WithToken(data.Token.ValueString()), WithEarlyAuth(false))
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

// Rotates the token using the token itself. Only works if the token has the `self_rotate` scope.
func (r *gitlabGroupServiceAccountAccessTokenResource) rotateTokenSelf(ctx context.Context, originalToken string, expiresAt gitlab.ISOTime) (*gitlab.PersonalAccessToken, error) {
	tokenClient, err := r.newGitLabClient(ctx, WithToken(originalToken), WithEarlyAuth(false))
	if err != nil {
		return nil, fmt.Errorf("Could not create a new client with the token that exists in state. The provider's token can't rotate the group service account access token: %v", err)
	}

	opt := &gitlab.RotatePersonalAccessTokenOptions{
		ExpiresAt: &expiresAt,
	}

	token, _, err := tokenClient.PersonalAccessTokens.RotatePersonalAccessTokenSelf(opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("Could not rotate GitLab GroupAccessToken, unexpected error: %v", err)
	}

	return token, nil
}
