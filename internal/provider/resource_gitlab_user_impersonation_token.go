package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabUserImpersonationTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabUserImpersonationTokenResource{}
	_ resource.ResourceWithImportState = &gitlabUserImpersonationTokenResource{}
)

func init() {
	registerResource(NewGitLabUserImpersonationTokenResource)
}

// NewGitLabUserImpersonationTokenResource is a helper function to simplify the provider implementation.
func NewGitLabUserImpersonationTokenResource() resource.Resource {
	return &gitlabUserImpersonationTokenResource{}
}

// gitlabUserImpersonationTokenResource defines the resource implementation.
type gitlabUserImpersonationTokenResource struct {
	client *gitlab.Client
}

// gitlabUserImpersonationTokenResourceModel describes the resource data model.
type gitlabUserImpersonationTokenResourceModel struct {
	ID                         types.String   `tfsdk:"id"`
	UserID                     types.Int64    `tfsdk:"user_id"`
	TokenID                    types.Int64    `tfsdk:"token_id"`
	Name                       types.String   `tfsdk:"name"`
	ExpiresAt                  types.String   `tfsdk:"expires_at"`
	Scopes                     []types.String `tfsdk:"scopes"`
	Revoked                    types.Bool     `tfsdk:"revoked"`
	Token                      types.String   `tfsdk:"token"`
	Active                     types.Bool     `tfsdk:"active"`
	Impersonation              types.Bool     `tfsdk:"impersonation"`
	CreatedAt                  types.String   `tfsdk:"created_at"`
	ValidatePastExpirationDate types.Bool     `tfsdk:"validate_past_expiration_date"`
}

func (r *gitlabUserImpersonationTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_impersonation_token"
}

func (r *gitlabUserImpersonationTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_user_impersonation_token`" + ` resource manages impersonation tokens of users.
Requires administrator access. Token values are returned once. You are only able to create impersonation tokens to impersonate the user and perform both API calls and Git reads and writes. The user can’t see these tokens in their profile settings page.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/user_tokens/#create-an-impersonation-token)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<user-id>:<token-id>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"token_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the impersonation token.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the impersonation token.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "Expiration date of the impersonation token in ISO format (YYYY-MM-DD).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"validate_past_expiration_date": schema.BoolAttribute{
				MarkdownDescription: "Wether to validate if the expiration date is in the future.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: fmt.Sprintf("Array of scopes of the impersonation token. valid values are: %s", utils.RenderValueListForDocs(api.ValidPersonalAccessTokenScopes)),
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
			"revoked": schema.BoolAttribute{
				MarkdownDescription: "True if the token is revoked.",
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token of the user impersonation token. **Note**: the token is not available for imported resources.",
				Computed:            true,
				Sensitive:           true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "True if the token is active.",
				Computed:            true,
			},
			"impersonation": schema.BoolAttribute{
				MarkdownDescription: "True as the token is always an impersonation token.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Time the token has been created, RFC3339 format.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabUserImpersonationTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabUserImpersonationTokenResource) userImpersonationTokenToStateModel(data *gitlabUserImpersonationTokenResourceModel, token *gitlab.ImpersonationToken, userID int64) diag.Diagnostics {
	data.UserID = types.Int64Value(userID)
	data.TokenID = types.Int64Value(int64(token.ID))
	data.Name = types.StringValue(token.Name)
	data.ExpiresAt = types.StringValue(token.ExpiresAt.String())

	// parse Scopes into []types.String
	var scopes []types.String
	for _, v := range token.Scopes {
		scopes = append(scopes, types.StringValue(v))
	}
	data.Scopes = scopes

	data.Revoked = types.BoolValue(token.Revoked)

	// Reading the token will not return a `token` value and we don't want to override what's in state when this happens
	if token.Token != "" {
		data.Token = types.StringValue(token.Token)
	}

	data.Active = types.BoolValue(token.Active)
	data.Impersonation = types.BoolValue(true)
	data.CreatedAt = types.StringValue(token.CreatedAt.Format(time.RFC3339))
	return nil
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabUserImpersonationTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Create a new upstream resource and add it into the Terraform state.
func (r *gitlabUserImpersonationTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabUserImpersonationTokenResourceModel

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

	_, expiryISOTime, err := utils.DetermineExpiryDate(data.ExpiresAt, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error determining expiry date", err.Error())
		return
	}

	// Convert gitlab.ISOTime to *time.Time for API call
	var timeExpiresAtPtr *time.Time
	if expiryISOTime != nil {
		if data.ValidatePastExpirationDate.ValueBool() {
			err := utils.ValidateISOTimeExpiryDate(*expiryISOTime)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating GitLab UserImpersonationToken",
					err.Error(),
				)
				return
			}
		}

		timeExpiresAt := time.Time(*expiryISOTime)
		timeExpiresAtPtr = &timeExpiresAt
	}

	options := &gitlab.CreateImpersonationTokenOptions{
		Name:      data.Name.ValueStringPointer(),
		Scopes:    gitlab.Ptr(scopes),
		ExpiresAt: timeExpiresAtPtr,
	}

	userID := data.UserID.ValueInt64()
	token, _, err := r.client.Users.CreateImpersonationToken(userID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GitLab User Impersonation Token",
			fmt.Sprintf("Could not create GitLab User Impersonation Token, unexpected error: %v", err),
		)
		return
	}

	// Set the ID for the resource
	data.ID = types.StringValue(fmt.Sprintf("%d:%d", userID, token.ID))
	r.userImpersonationTokenToStateModel(data, token, userID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabUserImpersonationTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabUserImpersonationTokenResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// get the user and tokenID from the resource ID
	userID, tokenID, err := r.parseResourceID(data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing ID",
			fmt.Sprintf("Could not parse ID into userID and tokenID: %s", err),
		)
		return
	}

	// Read the user impersonation token from the API
	token, _, err := r.client.Users.GetImpersonationToken(*userID, *tokenID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			// The user impersonation token doesn't exist anymore; remove it.
			tflog.Debug(ctx, fmt.Sprintf("GitLab User Impersonation Token %d, user ID %d not found, removing from state", tokenID, data.UserID))
			resp.State.RemoveResource(ctx)
			return
		}

		// Legit error, add a diagnostic and error
		resp.Diagnostics.AddError(
			"Error reading GitLab User Impersonation Token",
			fmt.Sprintf("Could not read GitLab User Impersonation Token, unexpected error: %v", err),
		)
		return
	}

	// Set the token information into state
	resp.Diagnostics.Append(r.userImpersonationTokenToStateModel(data, token, int64(*userID))...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabUserImpersonationTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

func (r *gitlabUserImpersonationTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform state data into the model to get ID
	var data *gitlabUserImpersonationTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// get the user and tokenID from the resource ID
	userID, tokenID, err := r.parseResourceID(data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing ID",
			fmt.Sprintf("Could not parse ID into userID and tokenID: %s", err),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Deleting User Impersonation Token %d from user %d", tokenID, userID))
	_, err = r.client.Users.RevokeImpersonationToken(data.UserID.ValueInt64(), *tokenID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting user impersonation token",
			fmt.Sprintf("Could not delete user impersonation token, unexpected error: %v", err),
		)
		return
	}
}

func (r *gitlabUserImpersonationTokenResource) parseResourceID(data *gitlabUserImpersonationTokenResourceModel) (*int64, *int64, error) {
	// get the user and tokenID from the resource ID
	userIDStr, tokenIDStr, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		return nil, nil, err
	}

	// Make sure the user ID is an int
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, nil, err
	}

	// Make sure the token ID is an int
	tokenID, err := strconv.ParseInt(tokenIDStr, 10, 64)
	if err != nil {
		return nil, nil, err
	}

	return &userID, &tokenID, nil
}
