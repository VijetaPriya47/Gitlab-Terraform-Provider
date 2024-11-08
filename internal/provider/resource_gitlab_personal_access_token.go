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
	_ resource.Resource                = &gitlabPersonalAccessTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabPersonalAccessTokenResource{}
	_ resource.ResourceWithImportState = &gitlabPersonalAccessTokenResource{}
	_ resource.ResourceWithModifyPlan  = &gitlabProjectAccessTokenResource{}
)

func init() {
	registerResource(NewGitLabPersonalAccessTokenResource)
}

func NewGitLabPersonalAccessTokenResource() resource.Resource {
	return &gitlabPersonalAccessTokenResource{}
}

type gitlabPersonalAccessTokenResource struct {
	client *gitlab.Client
}

// The base Resource implementation struct
type gitlabPersonalAccessTokenResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Token  types.String `tfsdk:"token"`
	UserId types.Int64  `tfsdk:"user_id"`

	// []string, or a set of types.String behind the scenes.
	Scopes []types.String `tfsdk:"scopes"`

	ExpiresAt types.String `tfsdk:"expires_at"`
	CreatedAt types.String `tfsdk:"created_at"`

	Active  types.Bool `tfsdk:"active"`
	Revoked types.Bool `tfsdk:"revoked"`

	// Defined in resource_gitlab_project_access_token.go
	RotationConfiguration *gitlabAccessTokenRotationConfiguration `tfsdk:"rotation_configuration"`
}

func (r *gitlabPersonalAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_personal_access_token"
}

func (r *gitlabPersonalAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_personal_access_token` + "`" + ` resource allows to manage the lifecycle of a personal access token.

-> This resource requires administration privileges.

~> Use of the ` + "`timestamp()`" + ` function with expires_at will cause the resource to be re-created with every apply, it's recommended to use ` + "`plantimestamp()`" + ` or a static value instead.

~> Observability scopes are in beta and may not work on all instances. See more details in [the documentation](https://docs.gitlab.com/ee/operations/tracing.html)

~> Use ` + "`rotation_configuration`" + ` to automatically rotate tokens instead of using ` + "`timestamp()`" + ` as timestamp will cause changes with every plan. ` + "`terraform apply`" + ` must still be run to rotate the token.

~> Due to [Automatic reuse detection](https://docs.gitlab.com/ee/api/personal_access_tokens.html#automatic-reuse-detection) it's possible that a new Personal Access Token will immediately be revoked. Check if an old process using the old token is running if this happens.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/ee/api/personal_access_tokens.html)`,
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
				},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "When the token will expire, YYYY-MM-DD format. Is automatically set when `rotation_configuration` is used.",
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
func (r *gitlabPersonalAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabPersonalAccessTokenResource) personalAccessTokenToStateModel(data *gitlabPersonalAccessTokenResourceModel, token *gitlab.PersonalAccessToken, userId int) diag.Diagnostics {

	data.UserId = types.Int64Value(int64(userId))
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

	// Check to determine if we need to rotate the expiry date
	shouldSetExpiration := false

	// If expiration (or ANY state) has never been set yet (we're in a "Create" plan), ensure we calculate and set the first time.
	if stateData == nil || stateData.ExpiresAt.IsNull() || stateData.ExpiresAt.IsUnknown() || stateData.ExpiresAt != planData.ExpiresAt {

		// Log some information for debugging later.
		expiresAt := ""
		if stateData != nil {
			expiresAt = stateData.ExpiresAt.ValueString()
		}
		tflog.Debug(ctx, "[PersonalAccessToken] State is not populated, or the expires_at value is nil. Creating the token for the first time.", map[string]interface{}{
			"is_state_nil": stateData == nil,
			"expires_at":   expiresAt,
		})

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
		tflog.Debug(ctx, "[PersonalAccessToken] State is populated, and a rotation configuration is detected. Determining if token should be rotated.", map[string]interface{}{
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
			tflog.Debug(ctx, "[PersonalAccessToken] Rotation is required, settings plan data", map[string]interface{}{
				"new_expires_at": expiryDate.String(),
				"expires_at":     stateData.ExpiresAt.ValueString(),
				"user_id":        planData.UserId.ValueInt64(),
				"name":           planData.Name.ValueString(),
			})

			resp.Diagnostics.Append(resp.Plan.Set(ctx, planData)...)
		}
	}
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
	userIdInt, err := strconv.Atoi(userId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing user ID",
			fmt.Sprintf("Could not parse user ID %q to int64: %s", userId, err),
		)
		return
	}

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
	resp.Diagnostics.Append(r.personalAccessTokenToStateModel(data, personalAccessToken, userIdInt)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPersonalAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabPersonalAccessTokenResourceModel

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
	options := &gitlab.CreatePersonalAccessTokenOptions{
		Name:   data.Name.ValueStringPointer(),
		Scopes: gitlab.Ptr(scopes),
	}

	// Optional attributes

	// // Get the valid expiry date from the `expires_at`
	expiryDate, err := r.determineExpiryDate(data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error determining expiry date",
			fmt.Sprintf("Could not determine expiry date: %s", err),
		)
		return
	}

	options.ExpiresAt = expiryDate

	token, _, err := r.client.Users.CreatePersonalAccessToken(int(data.UserId.ValueInt64()), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GitLab PersonalAccessToken",
			fmt.Sprintf("Could not create GitLab PersonalAccessToken, unexpected error: %v", err),
		)
		return
	}

	// Set the ID for the resource
	data.ID = types.StringValue(fmt.Sprintf("%d:%d", data.UserId.ValueInt64(), token.ID))

	r.personalAccessTokenToStateModel(data, token, int(data.UserId.ValueInt64()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPersonalAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state *gitlabPersonalAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Read the token and user ID from state since it may be `unknown` in the plan.
	userId, patId, err := utils.ParseTwoPartID(state.ID.ValueString())
	patIdInt, parseErrPat := strconv.Atoi(patId)
	userIdInt, parseErrUserId := strconv.Atoi(userId)
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
	expiresAt, err := gitlab.ParseISOTime(data.ExpiresAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing expiry date",
			fmt.Sprintf("Could not parse expiry date %s: %s", data.ExpiresAt.ValueString(), err),
		)
	}

	// update with a personal access token means rotate it
	token, _, err := r.client.PersonalAccessTokens.RotatePersonalAccessTokenByID(patIdInt, &gitlab.RotatePersonalAccessTokenOptions{
		ExpiresAt: &expiresAt,
	}, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error rotating GitLab PersonalAccessToken",
			fmt.Sprintf("Could not rotate GitLab PersonalAccessToken, unexpected error: %v", err),
		)
		return
	}

	// Updating an access token changes the primary key, so we need to re-set the ID of the resource
	data.ID = types.StringValue(utils.BuildTwoPartID(gitlab.Ptr(strconv.Itoa(int(data.UserId.ValueInt64()))), gitlab.Ptr(strconv.Itoa(token.ID))))

	r.personalAccessTokenToStateModel(data, token, userIdInt)
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

	personalAccessTokenID, err := strconv.Atoi(patId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %s to int: %s", patId, err),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Deleting PersonalAccessToken %d from user %s", personalAccessTokenID, userId))
	_, err = r.client.PersonalAccessTokens.RevokePersonalAccessToken(personalAccessTokenID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting personal access token",
			fmt.Sprintf("Could not delete personal access token, unexpected error: %v", err),
		)
		return
	}
}

// Takes in a resource model, and checks with the `expiry_date` or the `rotation_configuration` to determine what
// value should be set into the `expiry_date` field for the options.
// Returns a gitlab.ISOTime object of what should be set into the `expiry_date` field.
func (r *gitlabPersonalAccessTokenResource) determineExpiryDate(data *gitlabPersonalAccessTokenResourceModel) (*gitlab.ISOTime, error) {

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
