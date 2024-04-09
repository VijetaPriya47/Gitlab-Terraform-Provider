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
	_ resource.Resource                = &gitlabProjectAccessTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectAccessTokenResource{}
	_ resource.ResourceWithImportState = &gitlabProjectAccessTokenResource{}
)

func init() {
	registerResource(NewGitLabProjectAccessTokenResource)
}

func NewGitLabProjectAccessTokenResource() resource.Resource {
	return &gitlabProjectAccessTokenResource{}
}

type gitlabProjectAccessTokenResource struct {
	client *gitlab.Client
}

// The base Resource implementation struct
type gitlabProjectAccessTokenResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Project     types.String `tfsdk:"project"`
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

func (r *gitlabProjectAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_access_token"
}

func (r *gitlabProjectAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_project_access_token` + "`" + ` resource allows to manage the lifecycle of a project access token.

~>  Use of the ` + "`timestamp()`" + ` function with expires_at will cause the resource to be re-created with every apply, it's recommended to use ` + "`plantimestamp()`" + ` or a static value instead.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/ee/api/project_access_tokens.html)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project access token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the project access token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: "The scopes of the project access token.",
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
				Required: true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Time the token has been created, RFC3339 format.",
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token of the project access token. **Note**: the token is not available for imported resources.",
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
				MarkdownDescription: fmt.Sprintf("The access level for the project access token. Valid values are: %s. Default is `%s`.", utils.RenderValueListForDocs(api.ValidProjectAccessLevelNames), api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
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
func (r *gitlabProjectAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

func (r *gitlabProjectAccessTokenResource) projectAccessTokenToStateModel(data *gitlabProjectAccessTokenResourceModel, token *gitlab.ProjectAccessToken, project string) diag.Diagnostics {

	data.Project = types.StringValue(project)
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
func (r *gitlabProjectAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectAccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectAccessTokenResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// get the project and tokenID from the resource ID
	project, accessTokenId, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing ID",
			"Could not parse ID into project and accessTokenId",
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Read gitlab ProjectAccessToken %s, project ID %s", accessTokenId, project))

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
	projectAccessToken, _, err := r.client.ProjectAccessTokens.GetProjectAccessToken(project, accessTokenIdInt, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			// The access token doesn't exist anymore; remove it.
			tflog.Debug(ctx, fmt.Sprintf("GitLab ProjectAccessToken %s, project ID %s not found, removing from state", accessTokenId, project))
			resp.State.RemoveResource(ctx)
			return
		}

		// Legit error, add a diagnostic and error
		resp.Diagnostics.AddError(
			"Error reading GitLab ProjectAccessToken",
			fmt.Sprintf("Could not read GitLab ProjectAccessToken, unexpected error: %v", err),
		)
		return
	}

	// Set the token information into state
	resp.Diagnostics.Append(r.projectAccessTokenToStateModel(data, projectAccessToken, project)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectAccessTokenResourceModel

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
	options := &gitlab.CreateProjectAccessTokenOptions{
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

	token, _, err := r.client.ProjectAccessTokens.CreateProjectAccessToken(data.Project.ValueString(), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GitLab ProjectAccessToken",
			fmt.Sprintf("Could not create GitLab ProjectAccessToken, unexpected error: %v", err),
		)
		return
	}

	// Set the ID for the resource
	data.ID = types.StringValue(utils.BuildTwoPartID(data.Project.ValueStringPointer(), gitlab.Ptr(strconv.Itoa(token.ID))))

	r.projectAccessTokenToStateModel(data, token, data.Project.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Update only triggers when `expires_at` is updated. Anything else should trigger
	// a "replace" operation which will destory/create.
	var data *gitlabProjectAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	project, patId, err := utils.ParseTwoPartID(data.ID.ValueString())
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

	// update with a project access token means rotate it
	token, _, err := r.client.ProjectAccessTokens.RotateProjectAccessToken(project, intPatId, &gitlab.RotateProjectAccessTokenOptions{
		ExpiresAt: &expiresAt,
	}, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error rotating GitLab ProjectAccessToken",
			fmt.Sprintf("Could not rotate GitLab ProjectAccessToken, unexpected error: %v", err),
		)
		return
	}

	r.projectAccessTokenToStateModel(data, token, data.Project.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectAccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform state data into the model to get ID
	var data *gitlabProjectAccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	project, patId, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing resource ID",
			fmt.Sprintf("Could not parse resource ID %s into two parts properly", data.ID.ValueString()),
		)
		return
	}

	projectAccessTokenID, err := strconv.Atoi(patId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing access token ID",
			fmt.Sprintf("Could not parse access token ID %s to int: %s", patId, err),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Deleting ProjectAccessToken %d from project %s", projectAccessTokenID, project))
	_, err = r.client.ProjectAccessTokens.RevokeProjectAccessToken(project, projectAccessTokenID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project access token",
			fmt.Sprintf("Could not delete project access token, unexpected error: %v", err),
		)
		return
	}

	// Deleting access token is async, so Log that we're waiting for it to delete
	tflog.Info(ctx, "Waiting up to 5 minutes for async delete of project access token")
	err = retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
		_, _, err := r.client.ProjectAccessTokens.GetProjectAccessToken(project, projectAccessTokenID, gitlab.WithContext(ctx))
		if err != nil {
			if api.Is404(err) {
				return nil
			}
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(errors.New("project access token was not deleted"))
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project access token",
			fmt.Sprintf("Could not delete project access token, unexpected error: %v", err),
		)
	}
}
