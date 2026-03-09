package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabProjectCicdCatalogResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectCicdCatalogResource{}
	_ resource.ResourceWithImportState = &gitlabProjectCicdCatalogResource{}
)

func init() {
	registerResource(NewGitLabProjectCicdCatalogResource)
}

func NewGitLabProjectCicdCatalogResource() resource.Resource {
	return &gitlabProjectCicdCatalogResource{}
}

type gitlabProjectCicdCatalogResource struct {
	client *gitlab.Client
}

type gitlabProjectCicdCatalogResourceModel struct {
	Id                    types.String `tfsdk:"id"`
	Project               types.String `tfsdk:"project"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	KeepSettingsOnDestroy types.Bool   `tfsdk:"keep_settings_on_destroy"`
}

type gitlabProjectCicdCatalogPrivateStateModel struct {
	IsCatalogResource bool `json:"is_catalog_resource"`
}

// Metadata returns the resource name
func (r *gitlabProjectCicdCatalogResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_cicd_catalog"
}

func (r *gitlabProjectCicdCatalogResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_cicd_catalog`" + ` resource allows users to manage the lifecycle of a CI/CD Catalog project.

This resource controls whether a project is available as a CI/CD Catalog resource.

~> If ` + "`" + `keep_settings_on_destroy` + "`" + ` is set to false, destroying the resource will revert the catalog status to the value that was present when the resource was first created.
You will need to apply the resource with the new setting before destroying the resource.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/ee/api/graphql/reference/#mutationcatalogresourcescreate)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. Same as the `project` attribute.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the project should be enabled as a CI/CD Catalog resource.",
				Required:            true,
			},
			"keep_settings_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Set to true if the project CI/CD Catalog status should not be reset to its pre-terraform value on destroy. You will need to apply the resource with the new setting before destroying the resource.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectCicdCatalogResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// getProjectCatalogStatus queries the GitLab API to check if a project is a CI/CD Catalog resource.
// Returns (isCatalogResource, error). If the project doesn't exist in the response, isCatalogResource is false.
func (r *gitlabProjectCicdCatalogResource) getProjectCatalogStatus(ctx context.Context, projectFullPath string) (bool, error) {
	query := gitlab.GraphQLQuery{
		Query: `
			query($fullPath: ID!) {
				project(fullPath: $fullPath) {
					id
					isCatalogResource
				}
			}`,
		Variables: map[string]any{
			"fullPath": projectFullPath,
		},
	}
	tflog.Debug(ctx, "executing GraphQL Query to check CI/CD Catalog resource status", map[string]any{
		"query":     query.Query,
		"variables": query.Variables,
	})

	var response projectIsCatalogResourceQueryResponse
	if _, err := r.client.GraphQL.Do(query, &response, gitlab.WithContext(ctx)); err != nil {
		return false, err
	}

	// If the project doesn't exist (null response from GraphQL)
	if response.Data.Project == nil {
		return false, nil
	}

	return response.Data.Project.IsCatalogResource, nil
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectCicdCatalogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectCicdCatalogResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Id.ValueString()

	// Get project to get the full path
	project, _, err := r.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project does not exist, removing from state", map[string]any{
				"project": projectID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	// Check if the project is a catalog resource
	isCatalogResource, err := r.getProjectCatalogStatus(ctx, project.PathWithNamespace)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read CI/CD Catalog resource: %s", err.Error()))
		return
	}

	// Update the enabled status based on current state
	data.Enabled = types.BoolValue(isCatalogResource)
	data.Project = data.Id

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabProjectCicdCatalogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectCicdCatalogResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Project.ValueString()

	// Get project to get the full path
	project, _, err := r.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	// Store the original catalog status before making changes
	r.storeOriginalCatalogStatus(ctx, projectID, project.PathWithNamespace, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply the desired catalog status
	if err := r.setCatalogStatus(ctx, project.PathWithNamespace, data.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to set CI/CD Catalog status: %s", err.Error()))
		return
	}

	// Set the ID to the user-provided project identifier (preserves path or numeric ID)
	data.Id = data.Project

	// Log the creation of the resource
	tflog.Debug(ctx, "created a CI/CD Catalog resource", map[string]any{
		"project": data.Project.ValueString(),
		"enabled": data.Enabled.ValueBool(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabProjectCicdCatalogResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectCicdCatalogResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Id.ValueString()

	// Get project to get the full path
	project, _, err := r.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	// Apply the desired catalog status
	if err := r.setCatalogStatus(ctx, project.PathWithNamespace, data.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update CI/CD Catalog status: %s", err.Error()))
		return
	}

	// Log the update
	tflog.Debug(ctx, "updated CI/CD Catalog resource", map[string]any{
		"project": data.Project.ValueString(),
		"enabled": data.Enabled.ValueBool(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource.
func (r *gitlabProjectCicdCatalogResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectCicdCatalogResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Id.ValueString()

	// Get project to get the full path
	project, _, err := r.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			// Project doesn't exist, so the catalog resource is already gone
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	if !data.KeepSettingsOnDestroy.ValueBool() {
		// Restore the original catalog status
		original, diags := req.Private.GetKey(ctx, fmt.Sprintf("gitlab_project_cicd_catalog_original_%s", projectID))
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if original == nil {
			tflog.Debug(ctx, "Could not reset the CI/CD Catalog status", map[string]any{
				"project": projectID,
			})
			resp.Diagnostics.AddWarning("Could not reset the CI/CD Catalog status", "No original catalog status found to reset to")
			resp.State.RemoveResource(ctx)
			return
		}

		var status *gitlabProjectCicdCatalogPrivateStateModel
		err := json.Unmarshal(original, &status)
		if err != nil {
			tflog.Debug(ctx, "Could not unmarshal the original catalog status", map[string]any{
				"project":  projectID,
				"original": original,
			})
			resp.Diagnostics.AddError("Could not unmarshal the original catalog status", err.Error())
			return
		}

		tflog.Debug(ctx, "resetting original CI/CD Catalog status", map[string]any{
			"project": projectID,
			"enabled": status.IsCatalogResource,
		})

		// Restore to original status
		if err := r.setCatalogStatus(ctx, project.PathWithNamespace, status.IsCatalogResource); err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to reset CI/CD Catalog status: %s", err.Error()))
			return
		}

		tflog.Debug(ctx, "restored original CI/CD Catalog status", map[string]any{
			"project": projectID,
			"enabled": status.IsCatalogResource,
		})
	} else {
		tflog.Debug(ctx, "keeping CI/CD Catalog status on destroy", map[string]any{
			"project": projectID,
		})
	}

	resp.State.RemoveResource(ctx)
}

func (r *gitlabProjectCicdCatalogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// storeOriginalCatalogStatus stores the current catalog status before Terraform makes changes
func (r *gitlabProjectCicdCatalogResource) storeOriginalCatalogStatus(ctx context.Context, projectID string, projectPath string, resp *resource.CreateResponse) {
	isCatalogResource, err := r.getProjectCatalogStatus(ctx, projectPath)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get current CI/CD Catalog status: %s", err.Error()))
		return
	}

	status := gitlabProjectCicdCatalogPrivateStateModel{
		IsCatalogResource: isCatalogResource,
	}

	b, err := json.Marshal(status)
	if err != nil {
		resp.Diagnostics.AddError("Error marshalling status into json", fmt.Sprintf("Unable to marshall status into json: %s", err.Error()))
		return
	}

	// Use projectID as the key identifier to match what we use in Delete
	diags := resp.Private.SetKey(ctx, fmt.Sprintf("gitlab_project_cicd_catalog_original_%s", projectID), b)
	resp.Diagnostics.Append(diags...)
}

// setCatalogStatus enables or disables the catalog status for a project
func (r *gitlabProjectCicdCatalogResource) setCatalogStatus(ctx context.Context, projectPath string, enabled bool) error {
	// Get current status
	currentStatus, err := r.getProjectCatalogStatus(ctx, projectPath)
	if err != nil {
		return err
	}

	// If already in desired state, nothing to do
	if currentStatus == enabled {
		tflog.Debug(ctx, "CI/CD Catalog status already in desired state", map[string]any{
			"project": projectPath,
			"enabled": enabled,
		})
		return nil
	}

	if enabled {
		// Enable catalog
		query := gitlab.GraphQLQuery{
			Query: `
				mutation($projectPath: ID!) {
					catalogResourcesCreate(
						input: {
							projectPath: $projectPath
						}
					) {
						errors
					}
				}`,
			Variables: map[string]any{
				"projectPath": projectPath,
			},
		}
		tflog.Debug(ctx, "executing GraphQL Query to enable CI/CD Catalog", map[string]any{
			"query":     query.Query,
			"variables": query.Variables,
		})

		var response catalogResourcesCreateResponse
		if _, err := r.client.GraphQL.Do(query, &response, gitlab.WithContext(ctx)); err != nil {
			return err
		}

		// Check response for errors
		var allerr string
		if len(response.Errors) > 0 {
			for i, err := range response.Errors {
				allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
			}
		}
		if len(response.Data.CatalogResourcesCreate.Errors) > 0 {
			for i, err := range response.Data.CatalogResourcesCreate.Errors {
				allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
			}
		}
		if len(allerr) > 0 {
			return fmt.Errorf("GraphQL errors: %s", allerr)
		}
	} else {
		// Disable catalog
		query := gitlab.GraphQLQuery{
			Query: `
				mutation($projectPath: ID!) {
					catalogResourcesDestroy(
						input: {
							projectPath: $projectPath
						}
					) {
						errors
					}
				}`,
			Variables: map[string]any{
				"projectPath": projectPath,
			},
		}
		tflog.Debug(ctx, "executing GraphQL Query to disable CI/CD Catalog", map[string]any{
			"query":     query.Query,
			"variables": query.Variables,
		})

		var response catalogResourcesDestroyResponse
		if _, err := r.client.GraphQL.Do(query, &response, gitlab.WithContext(ctx)); err != nil {
			return err
		}

		// Check response for errors
		var allerr string
		if len(response.Errors) > 0 {
			for i, err := range response.Errors {
				allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
			}
		}
		if len(response.Data.CatalogResourcesDestroy.Errors) > 0 {
			for i, err := range response.Data.CatalogResourcesDestroy.Errors {
				allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
			}
		}
		if len(allerr) > 0 {
			return fmt.Errorf("GraphQL errors: %s", allerr)
		}
	}

	return nil
}

// GraphQL response types

type graphQLError struct {
	Message   string `json:"message"`
	Locations []struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"locations"`
	Path []string `json:"path"`
}

type projectIsCatalogResourceQueryResponse struct {
	Data struct {
		Project *struct {
			ID                string `json:"id"`
			IsCatalogResource bool   `json:"isCatalogResource"`
		} `json:"project"`
	} `json:"data"`
}

type catalogResourcesCreateResponse struct {
	Data struct {
		CatalogResourcesCreate struct {
			Errors []string `json:"errors"`
		} `json:"catalogResourcesCreate"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type catalogResourcesDestroyResponse struct {
	Data struct {
		CatalogResourcesDestroy struct {
			Errors []string `json:"errors"`
		} `json:"catalogResourcesDestroy"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}
