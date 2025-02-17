package provider

import (
	"context"
	"errors"
	"fmt"
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
	client *gitlab.Client
}

// The base Resource implementation struct
type gitlabGroupAccessTokenResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Group       types.String `tfsdk:"group"`
	Name        types.String `tfsdk:"name"`
	Token       types.String `tfsdk:"token"`
	UserId      types.Int64  `tfsdk:"user_id"`
	AccessLevel types.String `tfsdk:"access_level"`

	// []string, or a set of types.String behind the scenes.
	Scopes []types.String `tfsdk:"scopes"`

	ExpiresAt types.String `tfsdk:"expires_at"`
	CreatedAt types.String `tfsdk:"created_at"`

	Active  types.Bool `tfsdk:"active"`
	Revoked types.Bool `tfsdk:"revoked"`

	// Defined in resource_gitlab_project_access_token.go
	RotationConfiguration *gitlabAccessTokenRotationConfiguration `tfsdk:"rotation_configuration"`
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
			"scopes": schema.SetAttribute{
				MarkdownDescription: fmt.Sprintf("The scopes of the group access token. Valid values are: %s", utils.RenderValueListForDocs(api.ValidAccessTokenScopes)),
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
					setplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOfCaseInsensitive(api.ValidAccessTokenScopes...),
					),
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
}

func (r *gitlabGroupAccessTokenResource) groupAccessTokenToStateModel(data *gitlabGroupAccessTokenResourceModel, token *gitlab.GroupAccessToken, group string) diag.Diagnostics {

	data.Group = types.StringValue(group)
	data.Name = types.StringValue(token.Name)
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

	// parse Scopes into []types.String
	var scopes []types.String
	for _, v := range token.Scopes {
		scopes = append(scopes, types.StringValue(v))
	}
	data.Scopes = scopes

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
		tflog.Debug(ctx, "[GroupAccessToken] State is not populated, or the expires_at value is nil. Creating the token for the first time.", map[string]interface{}{
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
		tflog.Debug(ctx, "[GroupAccessToken] State is populated, and a rotation configuration is detected. Determining if token should be rotated.", map[string]interface{}{
			"expires_at":             rotateBefore,
			"detected_current_time":  api.CurrentTime(),
			"detected_rotation_date": gapTime,
			"rotate_before_days":     planData.RotationConfiguration.RotateBeforeDays.ValueInt64(),
			"should_rotate":          shouldSetExpiration,
		})
	}

	if shouldSetExpiration {
		// We need to re-calculate the expiryDate, and set it in the plan
		expiryDate, err := r.determineExpiryDate(planData)
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
			tflog.Debug(ctx, "[GroupAccessToken] Rotation is required, settings plan data", map[string]interface{}{
				"new_expires_at": expiryDate.String(),
				"expires_at":     stateData.ExpiresAt.ValueString(),
				"group":          planData.Group.ValueString(),
				"name":           planData.Name.ValueString(),
			})

			resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
		}
	}
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
	resp.Diagnostics.Append(r.groupAccessTokenToStateModel(data, groupAccessToken, group)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupAccessTokenResourceModel

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
	options := &gitlab.CreateGroupAccessTokenOptions{
		Name:   data.Name.ValueStringPointer(),
		Scopes: gitlab.Ptr(scopes),
	}

	// Optional attributes

	// Access level
	if !data.AccessLevel.IsNull() && !data.AccessLevel.IsUnknown() {
		accessLevel := api.AccessLevelNameToValue[data.AccessLevel.ValueString()]
		options.AccessLevel = &accessLevel
	}

	// Get the valid expiry date from the `expires_at` or `rotation_configuration`
	expiryDate, err := r.determineExpiryDate(data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error determining expiry date",
			fmt.Sprintf("Could not determine expiry date: %s", err),
		)
		return
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

	r.groupAccessTokenToStateModel(data, token, data.Group.ValueString())
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

	// update with a group access token means rotate it
	token, _, err := r.client.GroupAccessTokens.RotateGroupAccessToken(group, intPatId, &gitlab.RotateGroupAccessTokenOptions{
		ExpiresAt: &expiresAt,
	}, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error rotating GitLab GroupAccessTokens",
			fmt.Sprintf("Could not rotate GitLab GroupAccessTokens, unexpected error: %v", err),
		)
		return
	}

	// Updating an access token changes the primary key, so we need to re-set the ID of the resource
	data.ID = types.StringValue(utils.BuildTwoPartID(data.Group.ValueStringPointer(), gitlab.Ptr(strconv.Itoa(token.ID))))

	r.groupAccessTokenToStateModel(data, token, data.Group.ValueString())
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

// Takes in a resource model, and checks with the `expiry_date` or the `rotation_configuration` to determine what
// value should be set into the `expiry_date` field for the options.
// Returns a gitlab.ISOTime object of what should be set into the `expiry_date` field.
func (r *gitlabGroupAccessTokenResource) determineExpiryDate(data *gitlabGroupAccessTokenResourceModel) (*gitlab.ISOTime, error) {

	// If `expires_at` is set, then attempt to parse the time, and return the isoTime value if it
	// successfully parses
	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() && data.RotationConfiguration == nil {

		isoTime, err := gitlab.ParseISOTime(data.ExpiresAt.ValueString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse expiration date into ISOTime. Provided value: %s", data.ExpiresAt.ValueString())
		}
		return &isoTime, nil
	}

	// If `expires_at` is not set, then use the `rotation_configuration.expiration_days` if possible to to add the duration
	// to the current date to determine expiration, and return that instead. Otherwise, simply return nil, and let the default take.
	if data.RotationConfiguration != nil && !data.RotationConfiguration.ExpirationDays.IsNull() && !data.RotationConfiguration.ExpirationDays.IsUnknown() {
		now := api.CurrentTime()
		expiryDate := now.AddDate(0, 0, int(data.RotationConfiguration.ExpirationDays.ValueInt64()))
		expiryIsoTime, err := gitlab.ParseISOTime(expiryDate.Format(api.Iso8601))

		return &expiryIsoTime, err
	}

	return nil, nil
}
