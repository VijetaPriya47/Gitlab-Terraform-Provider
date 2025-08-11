package provider

import (
	"context"
	"fmt"

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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource              = &gitlabProjectIntegrationHarborResource{}
	_ resource.ResourceWithConfigure = &gitlabProjectIntegrationHarborResource{}
	// _ resource.ResourceWithImportState = &gitlabProjectIntegrationHarborResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationHarborResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationHarborResource)
}

func NewGitLabProjectIntegrationHarborResource() resource.Resource {
	return &gitlabProjectIntegrationHarborResource{
		ResourceName: "_project_integration_harbor",
	}
}

// Remove in 19.0
func NewGitLabIntegrationHarborResource() resource.Resource {
	return &gitlabProjectIntegrationHarborResource{
		ResourceName:       "_integration_harbor",
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_harbor` instead.",
	}
}

type gitlabProjectIntegrationHarborResource struct {
	client *gitlab.Client

	// Represents the name of the resource, since this resource uses both `gitlab_project_integration_harbor`
	// and `gitlab_integration_harbor` for backwards compatibility reasons. Should be removed in %19.0
	ResourceName       string
	DeprecationMessage string
}

type gitlabProjectIntegrationHarborResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Project              types.String `tfsdk:"project"`
	URL                  types.String `tfsdk:"url"`
	ProjectName          types.String `tfsdk:"project_name"`
	Username             types.String `tfsdk:"username"`
	Password             types.String `tfsdk:"password"`
	UseInheritedSettings types.Bool   `tfsdk:"use_inherited_settings"`
	Active               types.Bool   `tfsdk:"active"`
}

func (r *gitlabProjectIntegrationHarborResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationHarborResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + fmt.Sprintf(`gitlab%s`, r.ResourceName) + "`" + ` resource manages the lifecycle of a project integration with Harbor.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#harbor)`,
		DeprecationMessage: r.DeprecationMessage,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID or full path of the resource. Matches the `project` value.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID of the GitLab project you want to activate integration on.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Harbor URL. Example: `http://harbor.example.com`",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The URL-friendly Harbor project name. This project needs to already exist in Harbor. Example: `my_project_name`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for authentication with the Harbor server, if authentication is required by the server.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for authentication with the Harbor server, if authentication is required by the server.",
				Required:            true,
				Sensitive:           true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"use_inherited_settings": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether or not to inherit default settings. Defaults to false.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the integration is active.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectIntegrationHarborResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationHarborResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabProjectIntegrationHarborResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating hardor integration for project", map[string]any{
		"project":             data.ID.ValueString(),
		"harbor_url":          data.URL.ValueString(),
		"harbor_project_name": data.ProjectName.ValueString(),
	})
	err := r.updateIntegration(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Harbor integration", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationHarborResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabProjectIntegrationHarborResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	harbor, _, err := r.client.Services.GetHarborService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read gitlab harbor integration: %s", err.Error()))
		return
	}
	data.modelToStateModel(harbor, projectID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationHarborResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabProjectIntegrationHarborResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating hardor integration for project", map[string]any{
		"project":             data.ID.ValueString(),
		"harbor_url":          data.URL.ValueString(),
		"harbor_project_name": data.ProjectName.ValueString(),
	})
	err := r.updateIntegration(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Harbor integration", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationHarborResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationHarborResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Services.DeleteHarborService(data.Project.ValueString(), gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete Harbor integration: %s", err.Error()),
		)
		return
	}
}

func (r *gitlabProjectIntegrationHarborResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// updateIntegration performs the API call and update the `data` object with the results
// of the API call. The calling function should ensure that `state.Set` is called on the data
// object to set the values into state.
func (r *gitlabProjectIntegrationHarborResource) updateIntegration(ctx context.Context, data *gitlabProjectIntegrationHarborResourceModel) error {
	options := &gitlab.SetHarborServiceOptions{
		URL:                  data.URL.ValueStringPointer(),
		ProjectName:          data.ProjectName.ValueStringPointer(),
		Username:             data.Username.ValueStringPointer(),
		Password:             data.Password.ValueStringPointer(),
		UseInheritedSettings: data.UseInheritedSettings.ValueBoolPointer(),
	}

	updatedHarbor, _, err := r.client.Services.SetHarborService(data.Project.ValueString(), options, gitlab.WithContext(ctx))
	if err != nil {
		return err
	}

	data.modelToStateModel(updatedHarbor, data.Project.ValueString())
	return nil
}

func (d *gitlabProjectIntegrationHarborResourceModel) modelToStateModel(r *gitlab.HarborService, projectID string) {
	d.ID = types.StringValue(projectID)
	d.Project = types.StringValue(projectID)
	d.URL = types.StringValue(r.Properties.URL)
	d.ProjectName = types.StringValue(r.Properties.ProjectName)
	d.Username = types.StringValue(r.Properties.Username)
	d.UseInheritedSettings = types.BoolValue(r.Properties.UseInheritedSettings)
	d.Active = types.BoolValue(r.Active)
}
