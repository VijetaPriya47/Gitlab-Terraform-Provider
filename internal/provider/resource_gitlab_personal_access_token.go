package provider

import (
	"context"
	"fmt"
	"strconv"

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

~> Due to [Automatic reuse detection](https://docs.gitlab.com/ee/api/personal_access_tokens.html#automatic-reuse-detection) it's possible that a new Personal Access Token will immediately be revoked. Check if an old process using the old token is running if this happens.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/ee/api/personal_access_tokens.html)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the personal access token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
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
				MarkdownDescription: "When the token will expire, YYYY-MM-DD format.",
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
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabPersonalAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

func (r *gitlabPersonalAccessTokenResource) personalAccessTokenToStateModel(data *gitlabPersonalAccessTokenResourceModel, token *gitlab.PersonalAccessToken, userId int) diag.Diagnostics {

	data.UserId = types.Int64Value(int64(userId))
	data.Name = types.StringValue(token.Name)
	data.ExpiresAt = types.StringValue(token.ExpiresAt.String())
	data.CreatedAt = types.StringValue(token.CreatedAt.String())
	data.Active = types.BoolValue(token.Active)
	data.Revoked = types.BoolValue(token.Revoked)

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
func (r *gitlabPersonalAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
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
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
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
