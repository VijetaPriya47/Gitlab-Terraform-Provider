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
	_ resource.Resource                   = &gitlabProjectPackageDependencyProxyResource{}
	_ resource.ResourceWithConfigure      = &gitlabProjectPackageDependencyProxyResource{}
	_ resource.ResourceWithImportState    = &gitlabProjectPackageDependencyProxyResource{}
	_ resource.ResourceWithValidateConfig = &gitlabProjectPackageDependencyProxyResource{}
)

func init() {
	registerResource(NewGitLabProjectPackageDependencyProxyResource)
}

// NewGitLabProjectPackageDependencyProxyResource is a helper function to simplify the provider implementation.
func NewGitLabProjectPackageDependencyProxyResource() resource.Resource {
	return &gitlabProjectPackageDependencyProxyResource{}
}

type gitlabProjectPackageDependencyProxyResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Project types.String `tfsdk:"project"`

	Enabled                       types.Bool   `tfsdk:"enabled"`
	MavenExternalRegistryUrl      types.String `tfsdk:"maven_external_registry_url"`
	MavenExternalRegistryUsername types.String `tfsdk:"maven_external_registry_username"`
	MavenExternalRegistryPassword types.String `tfsdk:"maven_external_registry_password"`
}

type gitlabProjectPackageDependencyProxyResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectPackageDependencyProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_package_dependency_proxy"
}

func (r *gitlabProjectPackageDependencyProxyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Validate that username and password are either both set or both unset.
func (d *gitlabProjectPackageDependencyProxyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabProjectPackageDependencyProxyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Username and password must both be set or both be empty
	usernameSet := !data.MavenExternalRegistryUsername.IsNull() && !data.MavenExternalRegistryUsername.IsUnknown()
	passwordSet := !data.MavenExternalRegistryPassword.IsNull() && !data.MavenExternalRegistryPassword.IsUnknown()

	if usernameSet && !passwordSet {
		resp.Diagnostics.AddAttributeError(path.Root("maven_external_registry_password"), "Missing required attribute", "maven_external_registry_password must be set when maven_external_registry_username is set")
	}
	if passwordSet && !usernameSet {
		resp.Diagnostics.AddAttributeError(path.Root("maven_external_registry_username"), "Missing required attribute", "maven_external_registry_username must be set when maven_external_registry_password is set")
	}
}

func (r *gitlabProjectPackageDependencyProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_package_dependency_proxy`" + ` resource allows managing the project-level package dependency proxy for Maven packages.

This resource configures the external Maven registry settings for the dependency proxy, allowing packages to be cached and proxied through GitLab.

~> This resource requires GitLab Premium or Ultimate and the packages and dependency proxy features to be enabled on the GitLab instance.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#mutationupdatedependencyproxypackagessettings)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the dependency proxy is enabled for packages.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"maven_external_registry_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the external Maven registry.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"maven_external_registry_username": schema.StringAttribute{
				MarkdownDescription: "The username to authenticate with the external Maven registry. Must be set together with `maven_external_registry_password`.",
				Optional:            true,
			},
			"maven_external_registry_password": schema.StringAttribute{
				MarkdownDescription: "The password to authenticate with the external Maven registry. Must be set together with `maven_external_registry_username`. Cannot be imported.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabProjectPackageDependencyProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the framework helper to set the ID field properly
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Set the project field to the same value as ID to prevent replacement during import
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), req.ID)...)
}

// Create configures the dependency proxy settings for a project
func (r *gitlabProjectPackageDependencyProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectPackageDependencyProxyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the project
	project, _, err := r.client.Projects.GetProject(data.Project.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	// Update the dependency proxy settings
	response, err := r.updateDependencyProxyPackagesSettings(
		project,
		data.Enabled.ValueBool(),
		data.MavenExternalRegistryUrl.ValueString(),
		data.MavenExternalRegistryUsername.ValueString(),
		data.MavenExternalRegistryPassword.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update dependency proxy packages settings: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, "Successfully configured dependency proxy packages settings for project", map[string]any{
		"project": project.PathWithNamespace,
		"enabled": data.Enabled.ValueBool(),
	})

	r.modelToStateModel(data, project, response.Data.UpdateDependencyProxyPackagesSettings.DependencyProxyPackagesSetting)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectPackageDependencyProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectPackageDependencyProxyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the project
	project, _, err := r.client.Projects.GetProject(data.ID.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occurred", fmt.Sprintf("Project doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	// Read the dependency proxy packages settings
	response, err := r.readProjectDependencyProxyPackagesSettings(project)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read dependency proxy packages settings: %s", err.Error()))
		return
	}

	// Set the ID
	data.ID = types.StringValue(strconv.FormatInt(project.ID, 10))

	// Update the model with values from the API response
	r.modelToStateModel(data, project, response.Data.Project.DependencyProxyPackagesSetting)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectPackageDependencyProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectPackageDependencyProxyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the project
	project, _, err := r.client.Projects.GetProject(data.Project.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	// Update the dependency proxy settings
	response, err := r.updateDependencyProxyPackagesSettings(
		project,
		data.Enabled.ValueBool(),
		data.MavenExternalRegistryUrl.ValueString(),
		data.MavenExternalRegistryUsername.ValueString(),
		data.MavenExternalRegistryPassword.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update dependency proxy packages settings: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully updated dependency proxy packages settings for project %s", project.PathWithNamespace))
	r.modelToStateModel(data, project, response.Data.UpdateDependencyProxyPackagesSettings.DependencyProxyPackagesSetting)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete disables the dependency proxy and clears credentials
func (r *gitlabProjectPackageDependencyProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectPackageDependencyProxyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the project
	project, _, err := r.client.Projects.GetProject(data.Project.ValueString(), nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occurred", fmt.Sprintf("Project doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	// Disable the dependency proxy by setting enabled to false.
	// We MUST provide the existing URL because the API requires it even when disabling.
	// We clear the credentials (username/password) by passing empty strings.
	_, err = r.updateDependencyProxyPackagesSettings(
		project,
		false,
		data.MavenExternalRegistryUrl.ValueString(),
		"",
		"",
	)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to disable dependency proxy packages settings: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully disabled dependency proxy packages settings for project %s", project.PathWithNamespace))
	resp.State.RemoveResource(ctx)
}

// modelToStateModel updates the model with values from the API response
func (r *gitlabProjectPackageDependencyProxyResource) modelToStateModel(data *gitlabProjectPackageDependencyProxyResourceModel, project *gitlab.Project, settings DependencyProxyPackagesSetting) {
	data.ID = types.StringValue(strconv.FormatInt(project.ID, 10))
	data.Enabled = types.BoolValue(settings.Enabled)
	data.MavenExternalRegistryUrl = types.StringValue(settings.MavenExternalRegistryUrl)
	if settings.MavenExternalRegistryUsername != "" {
		data.MavenExternalRegistryUsername = types.StringValue(settings.MavenExternalRegistryUsername)
	} else {
		data.MavenExternalRegistryUsername = types.StringNull()
	}
	// Password is not returned in the response, keep the value from the model
}

// updateDependencyProxyPackagesSettings uses the GraphQL API to update settings
func (r *gitlabProjectPackageDependencyProxyResource) updateDependencyProxyPackagesSettings(project *gitlab.Project, enabled bool, url, username, password string) (*updateProjectDependencyProxyPackagesGraphQLResponse, error) {
	graphQLcall := fmt.Sprintf(`
mutation {
  updateDependencyProxyPackagesSettings(input: {
    projectPath: "%s",
    enabled: %t,
    mavenExternalRegistryUrl: "%s",
    mavenExternalRegistryUsername: "%s",
    mavenExternalRegistryPassword: "%s"
  }){
    errors
    dependencyProxyPackagesSetting {
      enabled
      mavenExternalRegistryUrl
      mavenExternalRegistryUsername
    }
  }
}`, project.PathWithNamespace, enabled, url, username, password)

	var response *updateProjectDependencyProxyPackagesGraphQLResponse
	_, err := r.client.GraphQL.Do(gitlab.GraphQLQuery{Query: graphQLcall}, &response)
	if err != nil {
		return nil, err
	}

	// Check if there are any string errors in the json response
	if response != nil && len(response.Data.UpdateDependencyProxyPackagesSettings.Errors) > 0 {
		return nil, errors.New(strings.Join(response.Data.UpdateDependencyProxyPackagesSettings.Errors, "\n"))
	}

	return response, nil
}

// readProjectDependencyProxyPackagesSettings uses the GraphQL API to read settings
func (r *gitlabProjectPackageDependencyProxyResource) readProjectDependencyProxyPackagesSettings(project *gitlab.Project) (*readProjectDependencyProxyPackagesGraphQLResponse, error) {
	graphQLcall := fmt.Sprintf(`
query {
  project(fullPath: "%s") {
    dependencyProxyPackagesSetting {
      enabled
      mavenExternalRegistryUrl
      mavenExternalRegistryUsername
    }
  }
}
`, project.PathWithNamespace)

	var response *readProjectDependencyProxyPackagesGraphQLResponse
	_, err := r.client.GraphQL.Do(gitlab.GraphQLQuery{Query: graphQLcall}, &response)
	if err != nil {
		return nil, err
	}

	// Check if there are any errors in the response
	if response != nil && len(response.Data.Project.Errors) > 0 {
		return nil, errors.New(strings.Join(response.Data.Project.Errors, "\n"))
	}

	return response, nil
}

// Response struct for reading dependency proxy packages settings
type readProjectDependencyProxyPackagesGraphQLResponse struct {
	Data struct {
		Project struct {
			Errors                         []string                       `json:"errors"`
			DependencyProxyPackagesSetting DependencyProxyPackagesSetting `json:"dependencyProxyPackagesSetting"`
		} `json:"project"`
	} `json:"data"`
}

// Response struct for mutating dependency proxy packages settings
type updateProjectDependencyProxyPackagesGraphQLResponse struct {
	Data struct {
		UpdateDependencyProxyPackagesSettings struct {
			Errors                         []string                       `json:"errors"`
			DependencyProxyPackagesSetting DependencyProxyPackagesSetting `json:"dependencyProxyPackagesSetting"`
		} `json:"updateDependencyProxyPackagesSettings"`
	} `json:"data"`
}

type DependencyProxyPackagesSetting struct {
	Enabled                       bool   `json:"enabled"`
	MavenExternalRegistryUrl      string `json:"mavenExternalRegistryUrl"`
	MavenExternalRegistryUsername string `json:"mavenExternalRegistryUsername"`
}
