package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabGroupAccessTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupAccessTokenResource{}
	_ resource.ResourceWithImportState = &gitlabGroupAccessTokenResource{}
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
}

func (r *gitlabGroupAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_access_token"
}

func (r *gitlabGroupAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_access`" + `token resource allows to manage the lifecycle of a group access token.

~> Observability scopes are in beta and may not work on all instances. See more details in [the documentation](https://docs.gitlab.com/ee/operations/tracing.html)

**Upstream API**: [GitLab REST API](https://docs.gitlab.com/ee/api/group_access_tokens.html)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group access token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
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
				MarkdownDescription: "The scopes of the group access token.",
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
				Computed: true,
				Optional: true,
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
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabGroupAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

func (r *gitlabGroupAccessTokenResource) groupAccessTokenToStateModel(data *gitlabGroupAccessTokenResourceModel, token *gitlab.GroupAccessToken, project string) diag.Diagnostics {

	data.Group = types.StringValue(project)
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
	var data *gitlabGroupAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	group, patId, err := utils.ParseTwoPartID(data.ID.ValueString())
	intPatId, parseErr := strconv.Atoi(patId)
	if joinedErr := errors.Join(err, parseErr); joinedErr != nil {
		resp.Diagnostics.AddError(
			"Error parsing resource ID",
			fmt.Sprintf("Could not parse resource ID %s into two parts properly", data.ID.ValueString()),
		)
		return
	}

	// determine the new expires_at from the config
	expiresAt, err := gitlab.ParseISOTime(data.ExpiresAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to parse expires_at value into a valid ISOTime",
			fmt.Sprintf("Failed to parse expires_at value into a valid ISOTime. Error: %v", err),
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
		_, _, err := r.client.GroupAccessTokens.GetGroupAccessToken(group, groupAccessTokenID, gitlab.WithContext(ctx))
		if err != nil {
			if api.Is404(err) {
				return nil
			}
			return retry.NonRetryableError(err)
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
