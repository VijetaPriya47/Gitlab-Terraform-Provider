package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabUserIdentityResource{}
	_ resource.ResourceWithConfigure   = &gitlabUserIdentityResource{}
	_ resource.ResourceWithImportState = &gitlabUserIdentityResource{}
)

func init() {
	registerResource(NewGitlabUserIdentityResource)
}

// NewGitlabUserIdentityResource is a helper function to simplify the provider implementation.
func NewGitlabUserIdentityResource() resource.Resource {
	return &gitlabUserIdentityResource{}
}

// gitlabUserIdentityResourceModel describes the resource data model.
type gitlabUserIdentityResourceModel struct {
	ID               types.String `tfsdk:"id"`
	UserID           types.Int64  `tfsdk:"user_id"`
	ExternalProvider types.String `tfsdk:"external_provider"`
	ExternalUID      types.String `tfsdk:"external_uid"`
}

// gitlabUserIdentityResource defines the resource implementation
type gitlabUserIdentityResource struct {
	client *gitlab.Client
}

func (r *gitlabUserIdentityResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_identity"
}

func (r *gitlabUserIdentityResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_user_identity`" + ` resource is for managing the lifecycle of a user's external identity.

-> the provider needs to be configured with admin-level access for this resource to work.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/users/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format <user-id:external-provider>",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The GitLab ID of the user.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"external_provider": schema.StringAttribute{
				MarkdownDescription: "The external provider name.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"external_uid": schema.StringAttribute{
				MarkdownDescription: "A specific external authentication provider UID.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabUserIdentityResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabUserIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Create creates a new upstream resources and adds it into the Terraform state.
func (r *gitlabUserIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabUserIdentityResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.ModifyUserOptions{
		ExternUID: data.ExternalUID.ValueStringPointer(),
		Provider:  data.ExternalProvider.ValueStringPointer(),
	}

	user, _, err := r.client.Users.ModifyUser(int(data.UserID.ValueInt64()), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to create user identity: %s", err.Error()))
		return
	}

	err = data.modelToStateModel(user.ID, data.ExternalProvider.ValueString(), user.Identities)
	if err != nil {
		resp.Diagnostics.AddError("User identity not found", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabUserIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabUserIdentityResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	userID, provider, err := resourceGitlabUserIdentityParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse ID: %s", err.Error()))
		return
	}

	user, _, err := r.client.Users.GetUser(int(userID), gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("User not found", fmt.Sprintf("[DEBUG] user %d not found so removing from state", userID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read user %d: %s", userID, err.Error()))
		return
	}

	err = data.modelToStateModel(userID, provider, user.Identities)
	if err != nil {
		resp.Diagnostics.AddWarning("User identity not found, removing from state", err.Error())
		resp.State.RemoveResource(ctx)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// All attributes require resource to be deleting and recreated on change, so this should never be called.
func (r *gitlabUserIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place update which is not possible.")
}

// Deletes removes the resource.
func (r *gitlabUserIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabUserIdentityResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID, provider, err := resourceGitlabUserIdentityParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse ID: %s", err.Error()))
		return
	}

	_, err = r.client.Users.DeleteUserIdentity(userID, provider, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete user identity provider: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (data *gitlabUserIdentityResourceModel) modelToStateModel(userID int, provider string, identities []*gitlab.UserIdentity) error {
	userIDStr := strconv.Itoa(userID)

	// Find the added identity in the return user identities
	for _, identity := range identities {
		if identity.Provider == provider {
			data.ID = types.StringValue(utils.BuildTwoPartID(&userIDStr, &identity.Provider))
			data.UserID = types.Int64Value(int64(userID))
			data.ExternalProvider = types.StringValue(identity.Provider)
			data.ExternalUID = types.StringValue(identity.ExternUID)
			return nil
		}
	}

	return fmt.Errorf("Unable to find user identity %s", data.ID.ValueString())
}

// Parse resource ID into user ID and provider.
func resourceGitlabUserIdentityParseID(id string) (int, string, error) {
	userIDStr, provider, err := utils.ParseTwoPartID(id)
	if err != nil {
		return 0, "", err
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, "", err
	}

	return userID, provider, nil
}
