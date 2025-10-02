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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabGroupAccessTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupAccessTokenResource{}
	_ resource.ResourceWithImportState = &gitlabGroupAccessTokenResource{}
	_ resource.ResourceWithModifyPlan  = &gitlabGroupAccessTokenResource{}
)

func init() {
	registerResource(NewGitLabGroupAccessTokenResource)
}

func NewGitLabGroupAccessTokenResource() resource.Resource {
	return &gitlabGroupAccessTokenResource{}
}

type gitlabGroupAccessTokenResource struct {
	client          *gitlab.Client
	newGitLabClient GitLabClientFactory
}

// The base Resource implementation struct
type gitlabGroupAccessTokenResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Group       types.String `tfsdk:"group"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Token       types.String `tfsdk:"token"`
	UserId      types.Int64  `tfsdk:"user_id"`
	AccessLevel types.String `tfsdk:"access_level"`

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

func (r *gitlabGroupAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_access_token"
}

func (r *gitlabGroupAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_access_token`" + ` resource allows to manage the lifecycle of a group access token.

~> Observability scopes are in beta and may not work on all instances. See more details in [the documentation](https://docs.gitlab.com/operations/tracing/)

~> Use ` + "`rotation_configuration`" + ` to automatically rotate tokens instead of using ` + "`timestamp()`" + ` as timestamp will cause changes with every plan. ` + "`terraform apply`" + ` must still be run to rotate the token.

~> Due to [Automatic reuse detection](https://docs.gitlab.com/api/group_access_tokens/#automatic-reuse-detection) it's possible that a new Group Access Token will immediately be revoked. Check if an old process using the old token is running if this happens.

**Upstream API**: [GitLab REST API](https://docs.gitlab.com/api/group_access_tokens/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group access token.",
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the group access token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the group access token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Computed: true,
				Optional: true,
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: fmt.Sprintf("The scopes of the group access token. Valid values are: %s", utils.RenderValueListForDocs(api.ValidGroupAccessTokenScopes)),
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
					setplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOfCaseInsensitive(api.ValidGroupAccessTokenScopes...),
					),
					setvalidator.SizeAtLeast(1),
				},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "When the token will expire, YYYY-MM-DD format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("rotation_configuration")),
				},
				Optional: true,
				Computed: true,
			},
			"validate_past_expiration_date": schema.BoolAttribute{
				MarkdownDescription: "Wether to validate if the expiration date is in the future.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Time the token has been created, RFC3339 format.",
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token of the group access token. **Note**: the token is not available for imported resources.",
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
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The user_id associated to the token.",
				Computed:            true,
			},
			"access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The access level for the group access token. Valid values are: %s. Default is `%s`.", utils.RenderValueListForDocs(api.ValidProjectAccessLevelNames), api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Default: stringdefault.StaticString(
					api.AccessLevelValueToName[gitlab.MaintainerPermissions],
				),
				Computed: true,
				Optional: true,
			},
			"rotation_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration for when to rotate a token automatically. Will not rotate a token until `terraform apply` is run.",
				Optional:            true,
				Validators: []validator.Object{
					objectvalidator.ExactlyOneOf(path.MatchRoot("expires_at")),
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
func (r *gitlabGroupAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
	r.newGitLabClient = resourceData.NewGitLabClient
}

func (r *gitlabGroupAccessTokenResource) groupAccessTokenToStateModel(ctx context.Context, data *gitlabGroupAccessTokenResourceModel, token *gitlab.GroupAccessToken, group string) diag.Diagnostics {
	data.Group = types.StringValue(group)
	data.Name = types.StringValue(token.Name)
	data.Description = types.StringValue(token.Description)
	data.ExpiresAt = types.StringValue(token.ExpiresAt.String())
	data.CreatedAt = types.StringValue(token.CreatedAt.String())
	data.Active = types.BoolValue(token.Active)
	data.Revoked = types.BoolValue(token.Revoked)
	data.UserId = types.Int64Value(int64(token.UserID))
	data.AccessLevel = types.StringValue(api.AccessLevelValueToName[token.AccessLevel])

	// Reading the token will not return a `token` value and we don't want to override what's in state when this happens
	if token.Token != "" {
		data.Token = types.StringValue(token.Token)
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
func (r *gitlabGroupAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Use the `ModifyPlan` to determine if we need to rotate the `token` associated to this
// resource, by checking the date that's set in the `expires_at` field is less than the `rotate_before_days`
// value.
func (r *gitlabGroupAccessTokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Retrieve the plan data to start with
	var planData, stateData *gitlabGroupAccessTokenResourceModel
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

	// Check to determine if we need to rotate the expiry date
	shouldSetExpiration := false

	// If expiration (or ANY state) has never been set yet (we're in a "Create" plan), ensure we calculate and set the first time.
	// This should also run if the expiration date has changed between plan and state, to ensure the ID is set to unknown.
	if stateData == nil || stateData.ExpiresAt.IsNull() || stateData.ExpiresAt.IsUnknown() || stateData.ExpiresAt != planData.ExpiresAt {
		// Log some information for debugging later.
		expiresAt := ""
		if stateData != nil {
			expiresAt = stateData.ExpiresAt.ValueString()
		}
		tflog.Debug(ctx, "[GroupAccessToken] State is not populated, or the expires_at value is nil. Creating the token for the first time.", map[string]any{
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
		tflog.Debug(ctx, "[GroupAccessToken] State is populated, and a rotation configuration is detected. Determining if token should be rotated.", map[string]any{
			"expires_at":             rotateBefore,
			"detected_current_time":  api.CurrentTime(),
			"detected_rotation_date": gapTime,
			"rotate_before_days":     planData.RotationConfiguration.RotateBeforeDays.ValueInt64(),
			"should_rotate":          shouldSetExpiration,
		})
	}

	if shouldSetExpiration {
		// We need to re-calculate the expiryDate, and set it in the plan
		expiryDate, err := utils.DetermineExpiryDate(planData.ExpiresAt, planData.RotationConfiguration, nil)
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
		if stateData != nil && expiryDate != nil && expiryDate.String() != stateData.ExpiresAt.ValueString() {
			// Set the new expiration date in the plan
			planData.ExpiresAt = types.StringValue(expiryDate.String())
			// Set several attributes to unknown since they will change as part of rotation
			planData.ID = types.StringUnknown()
			planData.Token = types.StringUnknown()
			planData.CreatedAt = types.StringUnknown()

			// Logs for assisting with support
			tflog.Debug(ctx, "[GroupAccessToken] Rotation is required, settings plan data", map[string]any{
				"new_expires_at": expiryDate.String(),
				"expires_at":     stateData.ExpiresAt.ValueString(),
				"group":          planData.Group.ValueString(),
				"name":           planData.Name.ValueString(),
			})

			resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
		}
	}
}

// modifyPlanRevoked handles token rotation if the token has been revoked externally.
func (r *gitlabGroupAccessTokenResource) modifyPlanRevoked(ctx context.Context, stateData *gitlabGroupAccessTokenResourceModel, planData *gitlabGroupAccessTokenResourceModel, resp *resource.ModifyPlanResponse) {
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
			tflog.Debug(ctx, "[gitlab_group_access_token] Token has been revoked externally but was expired anyway, no recreation needed")
			return
		}
	}

	tflog.Debug(ctx, "[gitlab_group_access_token] Token has been revoked externally and is still needed, marking for recreation")

	// Tell Terraform that a change to the revoked attribute requires replacing the resource
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("revoked"))

	// Calculate new expiration date
	expiryDate, err := utils.DetermineExpiryDate(planData.ExpiresAt, planData.RotationConfiguration, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error determining new expiry date",
			fmt.Sprintf("Could not determine new expiry date: %s", err),
		)
		return
	}

	// Set the planned values
	planData.ExpiresAt = types.StringValue(expiryDate.String())
	planData.Revoked = types.BoolValue(false) // Expect the new token to be not revoked

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
}

func (r *gitlabGroupAccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupAccessTokenResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// get the group and tokenID from the resource ID
	group, accessTokenId, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing ID",
			"Could not parse ID into group and accessTokenId",
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Read gitlab GroupAccessToken %s, group ID %s", accessTokenId, group))

	// Make sure the token ID is an int
	accessTokenIdInt, err := strconv.Atoi(accessTokenId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %q to int: %s", accessTokenId, err),
		)
		return
	}

	// Read the access token from the API
	groupAccessToken, _, err := r.client.GroupAccessTokens.GetGroupAccessToken(group, accessTokenIdInt, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			// The access token doesn't exist anymore; remove it.
			tflog.Debug(ctx, fmt.Sprintf("GitLab GroupAccessTokens %s, group ID %s not found, removing from state", accessTokenId, group))
			resp.State.RemoveResource(ctx)
			return
		}

		// Legit error, add a diagnostic and error
		resp.Diagnostics.AddError(
			"Error reading GitLab GroupAccessTokens",
			fmt.Sprintf("Could not read GitLab GroupAccessTokens, unexpected error: %v", err),
		)
		return
	}

	// Set the token information into state
	resp.Diagnostics.Append(r.groupAccessTokenToStateModel(ctx, data, groupAccessToken, group)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupAccessTokenResourceModel

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
	options := &gitlab.CreateGroupAccessTokenOptions{
		Name:   data.Name.ValueStringPointer(),
		Scopes: gitlab.Ptr(scopes),
	}

	// Optional attributes
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = data.Description.ValueStringPointer()
	}

	// Access level
	if !data.AccessLevel.IsNull() && !data.AccessLevel.IsUnknown() {
		accessLevel := api.AccessLevelNameToValue[data.AccessLevel.ValueString()]
		options.AccessLevel = &accessLevel
	}

	// Get the valid expiry date from the `expires_at` or `rotation_configuration`
	expiryDate, err := utils.DetermineExpiryDate(data.ExpiresAt, data.RotationConfiguration, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error determining expiry date",
			fmt.Sprintf("Could not determine expiry date: %s", err),
		)
		return
	}

	if data.ValidatePastExpirationDate.ValueBool() {
		err := utils.ValidateISOTimeExpiryDate(*expiryDate)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating GitLab GroupAccessToken",
				err.Error(),
			)
			return
		}

	}

	options.ExpiresAt = expiryDate

	token, _, err := r.client.GroupAccessTokens.CreateGroupAccessToken(data.Group.ValueString(), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GitLab GroupAccessTokens",
			fmt.Sprintf("Could not create GitLab GroupAccessTokens, unexpected error: %v", err),
		)
		return
	}

	// Set the ID for the resource
	data.ID = types.StringValue(utils.BuildTwoPartID(data.Group.ValueStringPointer(), gitlab.Ptr(strconv.Itoa(token.ID))))

	r.groupAccessTokenToStateModel(ctx, data, token, data.Group.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Update only triggers when `expires_at` is updated. Anything else should trigger
	// a "replace" operation which will destory/create.
	var data, state *gitlabGroupAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Read the ID from state since it may be `unknown` in the plan.
	group, patId, err := utils.ParseTwoPartID(state.ID.ValueString())
	intPatId, parseErr := strconv.Atoi(patId)
	if joinedErr := errors.Join(err, parseErr); joinedErr != nil {
		resp.Diagnostics.AddError(
			"Error parsing resource ID",
			fmt.Sprintf("Could not parse resource ID %s into two parts properly", data.ID.ValueString()),
		)
		return
	}

	// since modifyplan has determined the expiration date, simply retrieve it from the plan instead of re-calculating it.
	// re-calculating it here could result in a different value from the plan if the plan is run on a different date than
	// the apply, causing a "provider error" message to be sent to the user
	expiresAt, err := gitlab.ParseISOTime(data.ExpiresAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing expiry date",
			fmt.Sprintf("Could not parse expiry date %s: %s", data.ExpiresAt.ValueString(), err),
		)
		return
	}

	if data.ValidatePastExpirationDate.ValueBool() {
		err := utils.ValidateISOTimeExpiryDate(expiresAt)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating GitLab GroupAccessToken",
				err.Error(),
			)
			return
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

	// update with a group access token means rotate it
	var token *gitlab.GroupAccessToken
	if selfRotate {

		tflog.Debug(ctx, "Found `self_rotate` in scopes; attempting to use the self-rotate method to update the token", map[string]interface{}{
			"group":          group,
			"new_expires_at": expiresAt,
			"scopes":         data.Scopes,
		})

		token, err = r.rotateTokenSelf(ctx, state.Token.ValueString(), expiresAt, group)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error self rotating GitLab GroupAccessToken",
				fmt.Sprintf("Could not self rotate GitLab GroupAccessToken, unexpected error: %v", err),
			)
			return
		}

	} else {
		token, _, err = r.client.GroupAccessTokens.RotateGroupAccessToken(group, intPatId, &gitlab.RotateGroupAccessTokenOptions{
			ExpiresAt: &expiresAt,
		}, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error rotating GitLab GroupAccessTokens",
				fmt.Sprintf("Could not rotate GitLab GroupAccessTokens, unexpected error: %v", err),
			)
			return
		}
	}

	// Updating an access token changes the primary key, so we need to re-set the ID of the resource
	data.ID = types.StringValue(utils.BuildTwoPartID(data.Group.ValueStringPointer(), gitlab.Ptr(strconv.Itoa(token.ID))))

	r.groupAccessTokenToStateModel(ctx, data, token, data.Group.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupAccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform state data into the model to get ID
	var data *gitlabGroupAccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	group, patId, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing resource ID",
			fmt.Sprintf("Could not parse resource ID %s into two parts properly", data.ID.ValueString()),
		)
		return
	}

	groupAccessTokenID, err := strconv.Atoi(patId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %s to int: %s", patId, err),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Deleting GroupAccessTokens %d from group %s", groupAccessTokenID, group))
	_, err = r.client.GroupAccessTokens.RevokeGroupAccessToken(group, groupAccessTokenID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group access token",
			fmt.Sprintf("Could not delete group access token, unexpected error: %v", err),
		)
		return
	}

	// Deleting access token is async, so Log that we're waiting for it to delete
	tflog.Info(ctx, "Waiting up to 5 minutes for async delete of group access token")
	err = retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
		token, _, err := r.client.GroupAccessTokens.GetGroupAccessToken(group, groupAccessTokenID, gitlab.WithContext(ctx))
		if err != nil {
			if api.Is404(err) {
				tflog.Info(ctx, "Token is fully deleted.")
				return nil
			}
			return retry.NonRetryableError(err)
		}

		// Check if the token is revoked, and return nil because the token is "deleted" if it's been revoked.
		if token != nil && token.Revoked {
			tflog.Info(ctx, "Token is revoked. Treating as successfully deleted.")
			return nil
		}

		return retry.RetryableError(errors.New("group access token was not deleted"))
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group access token",
			fmt.Sprintf("Could not delete group access token, unexpected error: %v", err),
		)
	}
}

// Rotates the token using the token itself. Only works if the token has the `self_rotate` scope.
func (r *gitlabGroupAccessTokenResource) rotateTokenSelf(ctx context.Context, originalToken string, expiresAt gitlab.ISOTime, group string) (*gitlab.GroupAccessToken, error) {
	tokenClient, err := r.newGitLabClient(ctx, WithToken(originalToken), WithEarlyAuth(false))
	if err != nil {
		return nil, fmt.Errorf("Could not create a new client with the token that exists in state. The provider's token can't rotate the group access token: %v", err)
	}

	opt := &gitlab.RotateGroupAccessTokenOptions{
		ExpiresAt: &expiresAt,
	}

	token, _, err := tokenClient.GroupAccessTokens.RotateGroupAccessTokenSelf(group, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("Could not rotate GitLab GroupAccessToken, unexpected error: %v", err)
	}

	return token, nil
}
