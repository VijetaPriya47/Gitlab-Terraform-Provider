package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ resource.Resource                   = &gitlabGroupDependencyProxyResource{}
	_ resource.ResourceWithConfigure      = &gitlabGroupDependencyProxyResource{}
	_ resource.ResourceWithImportState    = &gitlabGroupDependencyProxyResource{}
	_ resource.ResourceWithValidateConfig = &gitlabGroupDependencyProxyResource{}
)

func init() {
	registerResource(NewGitLabGroupDependencyProxyResource)
}

// NewGitLabGroupDependencyProxyResource is a helper function to simplify the provider implementation.
func NewGitLabGroupDependencyProxyResource() resource.Resource {
	return &gitlabGroupDependencyProxyResource{}
}

type gitlabGroupDependencyProxyResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Group types.String `tfsdk:"group"`

	Enabled  types.Bool   `tfsdk:"enabled"`
	Identity types.String `tfsdk:"identity"`
	Secret   types.String `tfsdk:"secret"`
}

type gitlabGroupDependencyProxyResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupDependencyProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_dependency_proxy"
}

func (r *gitlabGroupDependencyProxyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Validate that identity/secret is only set when enabled is set to true, and not set when they're set to false.
func (d *gitlabGroupDependencyProxyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabGroupDependencyProxyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If proxy is enabled and neither the secret nor identity is set, error
	if !data.Enabled.ValueBool() {
		if !data.Identity.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("identity"), "Invalid attribute combination", "Identity cannot be set when proxy is disabled")
		}
		if !data.Secret.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("secret"), "Invalid attribute combination", "Secret cannot be set when proxy is disabled")
		}
	} else {
		// The proxy is enabled but doesn't have the necessary values, error
		if data.Identity.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("identity"), "Missing required attribute", "Identity must be set when proxy is enabled")
		}
		if data.Secret.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("secret"), "Missing required attribute", "Secret must be set when proxy is enabled")
		}
	}

}

func (r *gitlabGroupDependencyProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_dependency_proxy`" + ` resource allows managing the group docker dependency proxy. More than one dependency proxy per group will conflict with each other.

If you're looking to manage the project-level package dependency proxy, see the ` + "`gitlab_project_package_registry_proxy`" + ` resource instead.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#mutationupdatedependencyproxysettings)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the group.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the proxy is enabled.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"identity": schema.StringAttribute{
				MarkdownDescription: "Identity credential used to authenticate with Docker Hub when pulling images. Can be a username (for password or personal access token (PAT)) or organization name (for organization access token (OAT)).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Optional: true,
				Computed: true,
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Secret credential used to authenticate with Docker Hub when pulling images. Can be a password, personal access token (PAT), or organization access token (OAT). Cannot be imported.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupDependencyProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Creates the resource, which actually performs an "update" mutation since it's updating a setting on a group
func (r *gitlabGroupDependencyProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Get the model from the request
	var data *gitlabGroupDependencyProxyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the group
	group, _, err := r.client.Groups.GetGroup(data.Group.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get group: %s", err.Error()))
		return
	}

	// Update the dependency proxy settings
	response, err := r.updateDependencyProxySettings(ctx, r.client, group, data.Identity.ValueString(), data.Secret.ValueString(), data.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update dependency proxy settings: %s", err.Error()))
		return
	}
	// log a debug message that we've updated the proxy (both "create" and "update" technically update the proxy)
	tflog.Debug(ctx, "Successfully updated dependency proxy settings for group", map[string]any{
		"group":    group.FullPath,
		"identity": data.Identity.ValueString(),
	})
	r.modelToStateModel(data, group, *response)

	// Save the resource state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *gitlabGroupDependencyProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the model from the request
	var data *gitlabGroupDependencyProxyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the group
	group, _, err := r.client.Groups.GetGroup(data.ID.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occurred", fmt.Sprintf("Group doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get group: %s", err.Error()))
		return
	}

	// Read the dependency proxy settings
	response, err := readGroupDependencyProxySettings(ctx, r.client, group)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read dependency proxy settings: %s", err.Error()))
		return
	}

	// Note - the below items don't use the `modelToStateModel` because the response from the `Read` API is different from the mutate.
	// Set the ID
	data.ID = types.StringValue(strconv.Itoa(group.ID))

	// Update the model with the values from the API response
	if response.Data.Group.DependencyProxySettings.Enabled {
		data.Enabled = types.BoolValue(response.Data.Group.DependencyProxySettings.Enabled)
		data.Identity = types.StringValue(response.Data.Group.DependencyProxySettings.Identity)
		// Secret is not returned in the response, but we keep the value from the model
	}

	// Save the resource state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

// Updates the resource using the same mutation as the "Create" operation, since they're essentially both updates
func (r *gitlabGroupDependencyProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get the model from the request
	var data *gitlabGroupDependencyProxyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the group
	group, _, err := r.client.Groups.GetGroup(data.Group.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get group: %s", err.Error()))
		return
	}

	// Update the dependency proxy settings
	response, err := r.updateDependencyProxySettings(ctx, r.client, group, data.Identity.ValueString(), data.Secret.ValueString(), data.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update dependency proxy settings: %s", err.Error()))
		return
	}
	// log a debug message that we've updated the proxy
	tflog.Debug(ctx, fmt.Sprintf("Successfully updated dependency proxy settings for group %s", group.FullPath))
	r.modelToStateModel(data, group, *response)

	// Save the resource state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete is going to mutate the enabled value to "false" while setting identity and secret to empty strings to disable the proxy
// and remove any potentially hanging data
func (r *gitlabGroupDependencyProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the model from the request
	var data *gitlabGroupDependencyProxyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the group
	group, _, err := r.client.Groups.GetGroup(data.Group.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occurred", fmt.Sprintf("Group doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get group: %s", err.Error()))
		return
	}

	// Disable the dependency proxy by setting enabled to false and clearing credentials
	_, err = r.updateDependencyProxySettings(ctx, r.client, group, "", "", false)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to disable dependency proxy settings: %s", err.Error()))
		return
	}

	// log a debug message that we've disabled the proxy
	tflog.Debug(ctx, fmt.Sprintf("Successfully disabled dependency proxy settings for group %s", group.FullPath))
	resp.State.RemoveResource(ctx)

}

// Update the model with values from the API response
func (r *gitlabGroupDependencyProxyResource) modelToStateModel(data *gitlabGroupDependencyProxyResourceModel, group *gitlab.Group, response updateGroupDependencyProxyGraphQLResponse) {
	data.ID = types.StringValue(strconv.Itoa(group.ID))
	data.Enabled = types.BoolValue(response.Data.UpdateDependencyProxySettings.DependencyProxySettings.Enabled)
	data.Identity = types.StringValue(response.Data.UpdateDependencyProxySettings.DependencyProxySettings.Identity)
	// Secret is not returned in the response, but we keep the value from the model
}

// Uses the GraphQL API to update the dependency proxy settings for a group.
func (r *gitlabGroupDependencyProxyResource) updateDependencyProxySettings(ctx context.Context, client *gitlab.Client, group *gitlab.Group, identity, secret string, enabled bool) (*updateGroupDependencyProxyGraphQLResponse, error) {
	// The GraphQL Template for mutating the group's settings
	graphQLcall := fmt.Sprintf(`
mutation {
  updateDependencyProxySettings(input: {
    groupPath: "%s",
    enabled: %t,
    identity: "%s",
    secret: "%s"
  }){
   errors
   dependencyProxySetting{
      enabled,
      identity
   }
  }
}`, group.FullPath, enabled, identity, secret)

	// Make the GraphQL call
	var response *updateGroupDependencyProxyGraphQLResponse
	_, err := r.client.GraphQL.Do(gitlab.GraphQLQuery{Query: graphQLcall}, &response)
	if err != nil {
		return nil, err
	}

	// Check if there are any string errors in the json response
	if response != nil && len(response.Data.UpdateDependencyProxySettings.Errors) > 0 {
		return nil, errors.New(strings.Join(response.Data.UpdateDependencyProxySettings.Errors, "\n"))
	}

	return response, nil
}

// Uses the GraphQL API to read the dependency proxy settings for a group
// Note - this is not scoped to the resource because it's used in the test as well to confirm destroy
func readGroupDependencyProxySettings(ctx context.Context, client *gitlab.Client, group *gitlab.Group) (*readGroupDependencyProxyGraphQLResponse, error) {
	// The GraphQL Template for mutating the group's settings
	graphQLcall := fmt.Sprintf(`
query {
  group(fullPath:"%s") {
    dependencyProxySetting {
      enabled,
      identity
    }
  }
}
`, group.FullPath)

	// Make the GraphQL call
	var response *readGroupDependencyProxyGraphQLResponse
	_, err := client.GraphQL.Do(gitlab.GraphQLQuery{Query: graphQLcall}, &response)
	if err != nil {
		return nil, err
	}

	// Check if there are any string errors in the json response
	if response != nil && len(response.Data.Group.Errors) > 0 {
		return nil, errors.New(strings.Join(response.Data.Group.Errors, "\n"))
	}

	return response, nil
}

// A response struct for the GraphQL call to read the group's dependency proxy settings
type readGroupDependencyProxyGraphQLResponse struct {
	Data struct {
		Group struct {
			Errors []string `json:"errors"`

			DependencyProxySettings struct {
				Enabled  bool   `json:"enabled"`
				Identity string `json:"identity"`
				// Secret doesn't come back on the read for security reasons
			} `json:"dependencyProxySetting"`
		} `json:"group"`
	} `json:"data"`
}

// A response struct for the GraphQL call to mutate the group's dependency proxy settings
type updateGroupDependencyProxyGraphQLResponse struct {
	Data struct {
		UpdateDependencyProxySettings struct {
			Errors                  []string `json:"errors"`
			DependencyProxySettings struct {
				Enabled  bool   `json:"enabled"`
				Identity string `json:"identity"`
				Secret   string `json:"secret"`
			} `json:"dependencyProxySetting"`
		} `json:"updateDependencyProxySettings"`
	} `json:"data"`
}
