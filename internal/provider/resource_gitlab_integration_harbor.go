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
	_ resource.Resource              = &gitlabIntegrationHarborResource{}
	_ resource.ResourceWithConfigure = &gitlabIntegrationHarborResource{}
	// _ resource.ResourceWithImportState = &gitlabIntegrationHarborResource{}
)

func init() {
	registerResource(NewGitLabIntegrationHarborResource)
}

func NewGitLabIntegrationHarborResource() resource.Resource {
	return &gitlabIntegrationHarborResource{}
}

type gitlabIntegrationHarborResource struct {
	client *gitlab.Client
}

type gitlabIntegrationHarborResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Project              types.String `tfsdk:"project"`
	URL                  types.String `tfsdk:"url"`
	ProjectName          types.String `tfsdk:"project_name"`
	Username             types.String `tfsdk:"username"`
	Password             types.String `tfsdk:"password"`
	UseInheritedSettings types.Bool   `tfsdk:"use_inherited_settings"`
	Active               types.Bool   `tfsdk:"active"`
}

func (r *gitlabIntegrationHarborResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_harbor"
}

func (r *gitlabIntegrationHarborResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_integration_harbor`" + ` resource allows to manage the lifecycle of a project integration with Harbor.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#harbor)`,
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

func (r *gitlabIntegrationHarborResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabIntegrationHarborResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabIntegrationHarborResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating hardor integration for project", map[string]interface{}{
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

func (r *gitlabIntegrationHarborResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabIntegrationHarborResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	harbor, _, err := r.client.Services.GetHarborService(projectID)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read gitlab harbor integration: %s", err.Error()))
		return
	}
	data.modelToStateModel(harbor, projectID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabIntegrationHarborResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabIntegrationHarborResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating hardor integration for project", map[string]interface{}{
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

func (r *gitlabIntegrationHarborResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabIntegrationHarborResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Services.DeleteHarborService(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete Harbor integration: %s", err.Error()),
		)
		return
	}
}

func (r *gitlabIntegrationHarborResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// updateIntegration performs the API call and update the `data` object with the results
// of the API call. The calling function should ensure that `state.Set` is called on the data
// object to set the values into state.
func (r *gitlabIntegrationHarborResource) updateIntegration(ctx context.Context, data *gitlabIntegrationHarborResourceModel) error {
	options := &gitlab.SetHarborServiceOptions{
		URL:                  data.URL.ValueStringPointer(),
		ProjectName:          data.ProjectName.ValueStringPointer(),
		Username:             data.Username.ValueStringPointer(),
		Password:             data.Password.ValueStringPointer(),
		UseInheritedSettings: data.UseInheritedSettings.ValueBoolPointer(),
	}

	updatedHarbor, _, err := r.client.Services.SetHarborService(data.Project.ValueString(), options)
	if err != nil {
		return err
	}

	data.modelToStateModel(updatedHarbor, data.Project.ValueString())
	return nil
}

func (d *gitlabIntegrationHarborResourceModel) modelToStateModel(r *gitlab.HarborService, projectID string) {
	d.ID = types.StringValue(projectID)
	d.Project = types.StringValue(projectID)
	d.URL = types.StringValue(r.Properties.URL)
	d.ProjectName = types.StringValue(r.Properties.ProjectName)
	d.Username = types.StringValue(r.Properties.Username)
	d.UseInheritedSettings = types.BoolValue(r.Properties.UseInheritedSettings)
	d.Active = types.BoolValue(r.Active)
}
