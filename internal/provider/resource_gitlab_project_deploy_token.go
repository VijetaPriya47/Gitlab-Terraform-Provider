package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	_ resource.Resource                = &gitlabProjectDeployTokenResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectDeployTokenResource{}
	_ resource.ResourceWithImportState = &gitlabProjectDeployTokenResource{}
)

func init() {
	registerResource(NewGitLabProjectDeployTokenResource)
}

func NewGitLabProjectDeployTokenResource() resource.Resource {
	return &gitlabProjectDeployTokenResource{}
}

type gitlabProjectDeployTokenResource struct {
	client *gitlab.Client
}

type gitlabProjectDeployTokenResourceModel struct {
	Id                         types.String      `tfsdk:"id"`
	Project                    types.String      `tfsdk:"project"`
	Name                       types.String      `tfsdk:"name"`
	Username                   types.String      `tfsdk:"username"`
	Scopes                     types.Set         `tfsdk:"scopes"`
	ExpiresAt                  timetypes.RFC3339 `tfsdk:"expires_at"`
	Token                      types.String      `tfsdk:"token"`
	ValidatePastExpirationDate types.Bool        `tfsdk:"validate_past_expiration_date"`

	Expired types.Bool `tfsdk:"expired"`
	Revoked types.Bool `tfsdk:"revoked"`
}

// Metadata returns the resource name
func (d *gitlabProjectDeployTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_deploy_token"
}

func (r *gitlabProjectDeployTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_deploy_token`" + ` resource allows you to manage the lifecycle of deploy tokens on a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/deploy_tokens/#project-deploy-tokens)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Id of this Terraform resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The Id or full path of the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "A name to describe the deploy token with.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "A username for the deploy token. Default is `gitlab+deploy-token-{n}`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Optional: true,
				Computed: true,
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: fmt.Sprintf("The scopes of the project deploy token. Valid values are: %s", utils.RenderValueListForDocs(api.ValidDeployTokenScopes)),
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
					setplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOfCaseInsensitive(api.ValidDeployTokenScopes...),
					),
					setvalidator.SizeAtLeast(1),
				},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "Time the token expires in RFC3339 format. Not set by default.",
				CustomType:          timetypes.RFC3339Type{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
			"expired": schema.BoolAttribute{
				MarkdownDescription: "True if the token is expired.",
				Computed:            true,
			},
			"revoked": schema.BoolAttribute{
				MarkdownDescription: "True if the token is revoked.",
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The secret token. This is only populated when creating a new deploy token. **Note**: The token is not available for imported resources.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectDeployTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectDeployTokenResource) projectDeployTokenToStateModel(ctx context.Context, data *gitlabProjectDeployTokenResourceModel, token *gitlab.DeployToken, project string) diag.Diagnostics {
	data.Project = types.StringValue(project)
	data.Name = types.StringValue(token.Name)
	data.Username = types.StringValue(token.Username)
	data.Expired = types.BoolValue(token.Expired)
	data.Revoked = types.BoolValue(token.Revoked)

	if token.ExpiresAt != nil {
		expiresAt, diags := timetypes.NewRFC3339Value(token.ExpiresAt.Format(time.RFC3339))
		if diags.HasError() {
			return diags
		}
		data.ExpiresAt = expiresAt
	} else {
		data.ExpiresAt = timetypes.NewRFC3339Null()
	}

	// parse Scopes into types.Set
	scopesSet, diags := types.SetValueFrom(ctx, types.StringType, token.Scopes)
	if diags.HasError() {
		return diags
	}
	data.Scopes = scopesSet

	// Reading the token will not return a `token` value and we don't want to override what's in state when this happens
	if token.Token != "" {
		data.Token = types.StringValue(token.Token)
	}

	return nil
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectDeployTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectDeployTokenResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// get the project and tokenId from the resource Id
	project, deployTokenId, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing Id",
			"Could not parse Id into project and deployTokenId",
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Read GitLab ProjectDeployToken %s, project Id %s", deployTokenId, project))

	// Make sure the token Id is an int
	deployTokenIdInt, err := strconv.ParseInt(deployTokenId, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing deploy token Id",
			fmt.Sprintf("Could not parse deploy token Id %q to int: %s", deployTokenId, err),
		)
		return
	}

	// Read the deploy token from the API
	projectDeployToken, _, err := r.client.DeployTokens.GetProjectDeployToken(project, deployTokenIdInt, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			// The deploy token doesn't exist anymore; remove it.
			tflog.Debug(ctx, fmt.Sprintf("GitLab ProjectDeployToken %s, project Id %s not found, removing from state", deployTokenId, project))
			resp.State.RemoveResource(ctx)
			return
		}

		// Legit error, add a diagnostic and error
		resp.Diagnostics.AddError(
			"Error reading GitLab ProjectDeployToken",
			fmt.Sprintf("Could not read GitLab ProjectDeployToken, unexpected error: %v", err),
		)
		return
	}

	// Set the token information into state
	resp.Diagnostics.Append(r.projectDeployTokenToStateModel(ctx, data, projectDeployToken, project)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabProjectDeployTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectDeployTokenResourceModel

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
	options := &gitlab.CreateProjectDeployTokenOptions{
		Name:   data.Name.ValueStringPointer(),
		Scopes: gitlab.Ptr(scopes),
	}

	// Optional attributes
	if !data.Username.IsNull() && !data.Username.IsUnknown() {
		options.Username = data.Username.ValueStringPointer()
	}

	// Get the valid expiry date from `expires_at`
	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		parsedExpiresAt, err := utils.DetermineRFC3339ExpiryDate(data.ExpiresAt)
		if err != nil {
			resp.Diagnostics.AddError("Error determining expiry date", err.Error())
			return
		}

		if data.ValidatePastExpirationDate.ValueBool() {
			err := utils.ValidateExpiryDateValid(*parsedExpiresAt)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating GitLab ProjectDeployToken",
					err.Error(),
				)
				return
			}
		}

		options.ExpiresAt = parsedExpiresAt
	}

	token, _, err := r.client.DeployTokens.CreateProjectDeployToken(data.Project.ValueString(), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GitLab ProjectDeployToken",
			fmt.Sprintf("Could not create GitLab ProjectDeployToken, unexpected error: %v", err),
		)
		return
	}

	// Set the Id for the resource
	data.Id = types.StringValue(utils.BuildTwoPartID(data.Project.ValueStringPointer(), gitlab.Ptr(strconv.FormatInt(token.ID, 10))))

	r.projectDeployTokenToStateModel(ctx, data, token, data.Project.ValueString())

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabProjectDeployTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

// Delete removes the resource.
func (r *gitlabProjectDeployTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectDeployTokenResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, deployTokenId, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing resource Id",
			fmt.Sprintf("Could not parse resource Id %s into two parts properly", data.Id.ValueString()),
		)
		return
	}

	projectDeployTokenIdInt, err := strconv.ParseInt(deployTokenId, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing deploy token Id",
			fmt.Sprintf("Could not parse deploy token Id %s to int: %s", deployTokenId, err),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Deleting ProjectDeployToken %d from project %s", projectDeployTokenIdInt, project))
	_, err = r.client.DeployTokens.DeleteProjectDeployToken(project, projectDeployTokenIdInt, gitlab.WithContext(ctx))
	if err != nil {
		if api.ProjectMoved(err) {
			tflog.Debug(ctx, "The project the token is assigned to has been moved, gracefully deleting token from state")
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting project deploy token",
			fmt.Sprintf("Could not delete project deploy token, unexpected error: %v", err),
		)
		return
	}
}

func (r *gitlabProjectDeployTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
