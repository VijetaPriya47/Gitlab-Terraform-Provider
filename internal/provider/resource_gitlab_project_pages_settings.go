package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ resource.Resource                = &gitlabProjectPagesSettingsResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectPagesSettingsResource{}
	_ resource.ResourceWithImportState = &gitlabProjectPagesSettingsResource{}
)

func init() {
	registerResource(NewGitlabProjectPagesSettingsResource)
}

func NewGitlabProjectPagesSettingsResource() resource.Resource {
	return &gitlabProjectPagesSettingsResource{}
}

type gitlabProjectPagesSettingsResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectPagesSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_pages_settings"
}

func (r *gitlabProjectPagesSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectPagesSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type gitlabProjectPagesSettingsResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Project               types.String `tfsdk:"project"`
	KeepSettingsOnDestroy types.Bool   `tfsdk:"keep_settings_on_destroy"`
	URL                   types.String `tfsdk:"url"`
	IsUniqueDomainEnabled types.Bool   `tfsdk:"is_unique_domain_enabled"`
	ForceHTTPS            types.Bool   `tfsdk:"force_https"`
	Deployments           types.List   `tfsdk:"deployments"`
}

type gitlabProjectPagesSettingsDeploymentResourceModel struct {
	CreatedAt     types.String `tfsdk:"created_at"`
	URL           types.String `tfsdk:"url"`
	PathPrefix    types.String `tfsdk:"path_prefix"`
	RootDirectory types.String `tfsdk:"root_directory"`
}

func (r *gitlabProjectPagesSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`" + `gitlab_project_pages_settings` + "`" + ` resource manages project pages settings in GitLab.

~> This is an **experimental resource**. By nature it doesn't properly fit into how Terraform resources are meant to work.

~> When you destroy the resource, you can control if pages settings are saved or not. Set ` + "`" + `keep_settings_on_destroy` + "`" + ` to ` + "`" + `true` + "`" + ` (default) to save changes to pages settings. Set ` + "`" + `keep_settings_on_destroy` + "`" + ` to ` + "`" + `false` + "`" + ` to reset the pages settings to its original values.
The original values are saved in state when you create the resource. You can change the ` + "`" + `keep_settings_on_destroy` + "`" + ` value before destroying the resource to control this behavior.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/pages/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format `<project-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The project ID or path.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"keep_settings_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Set to true if the pages settings should not be reset to their pre-terraform defaults on destroy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to access the project pages.",
				Computed:            true,
			},
			"is_unique_domain_enabled": schema.BoolAttribute{
				MarkdownDescription: "Boolean indicating if a unique domain is enabled.",
				Optional:            true,
				Computed:            true,
			},
			"force_https": schema.BoolAttribute{
				MarkdownDescription: "Boolean indicating if the project is set to force https. Requires `external_https` to be configured in the GitLab instance: https://docs.gitlab.com/administration/pages/#custom-domains-with-tls-support.",
				Optional:            true,
				Computed:            true,
			},
			"deployments": projectPagesDeploymentsSchema(),
		},
	}
}

func projectPagesDeploymentsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "List of current active deployments.",
		Computed:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"created_at": schema.StringAttribute{
					MarkdownDescription: "Date the deployment was created.",
					Computed:            true,
				},
				"url": schema.StringAttribute{
					MarkdownDescription: "The URL of the deployment.",
					Computed:            true,
				},
				"path_prefix": schema.StringAttribute{
					MarkdownDescription: "The path prefix of the deployment when using parallel deployments.",
					Computed:            true,
				},
				"root_directory": schema.StringAttribute{
					MarkdownDescription: "The root directory of the deployment.",
					Computed:            true,
				},
			},
		},
	}
}

func (r *gitlabProjectPagesSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectPagesSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	r.storeOriginalSettings(ctx, project, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.changePagesSettings(project, data, ctx)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to change project pages settings: %s", err.Error()))
		return
	}

	data.settingsToStateModel(ctx, project, settings)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type gitlabProjectPagesSettingsPrivateStateModel struct {
	IsUniqueDomainEnabled bool `tfsdk:"is_unique_domain_enabled"`
	ForceHTTPS            bool `tfsdk:"force_https"`
}

func (r *gitlabProjectPagesSettingsResource) storeOriginalSettings(ctx context.Context, project string, resp *resource.CreateResponse) {
	settings, _, err := r.client.Pages.GetPages(project, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get current project pages settings: %s", err.Error()))
		return
	}

	data := gitlabProjectPagesSettingsPrivateStateModel{
		IsUniqueDomainEnabled: settings.IsUniqueDomainEnabled,
		ForceHTTPS:            settings.ForceHTTPS,
	}
	b, err := json.Marshal(data)
	if err != nil {
		resp.Diagnostics.AddError("Error marshalling settings into json", fmt.Sprintf("Unable to marshall resource model into json: %s", err.Error()))
		return
	}

	diags := resp.Private.SetKey(ctx, "gitlab_project_pages_settings_original", b)
	resp.Diagnostics.Append(diags...)
}

func (r *gitlabProjectPagesSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectPagesSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.ID.ValueString()

	settings, _, err := r.client.Pages.GetPages(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, fmt.Sprintf("GitLab project not found %s, removing from state", project))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project pages settings: %s", err.Error()))
		return
	}

	data.settingsToStateModel(ctx, project, settings)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectPagesSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectPagesSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.changePagesSettings(data.ID.ValueString(), data, ctx)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to change project pages settings: %s", err.Error()))
		return
	}

	data.settingsToStateModel(ctx, data.ID.ValueString(), settings)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectPagesSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectPagesSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.KeepSettingsOnDestroy.ValueBool() {
		original, diags := req.Private.GetKey(ctx, "gitlab_project_pages_settings_original")
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if original == nil {
			resp.Diagnostics.AddWarning("Could not reset the project pages settings", "No settings found to reset to")
			resp.State.RemoveResource(ctx)
			return
		}

		var settings *gitlabProjectPagesSettingsPrivateStateModel
		err := json.Unmarshal(original, &settings)
		if err != nil {
			resp.Diagnostics.AddError("Could not unmarshal the original settings", err.Error())
			return
		}

		options := gitlab.UpdatePagesOptions{
			PagesUniqueDomainEnabled: gitlab.Ptr(settings.IsUniqueDomainEnabled),
			PagesHTTPSOnly:           gitlab.Ptr(settings.ForceHTTPS),
		}
		_, _, err = r.client.Pages.UpdatePages(data.ID.ValueString(), options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to reset project pages settings: %s", err.Error()))
			return
		}
	}

	tflog.Debug(ctx, "Destroying the project pages settings resource does not do anything.")
	resp.State.RemoveResource(ctx)
}

func (r *gitlabProjectPagesSettingsResource) changePagesSettings(project string, data *gitlabProjectPagesSettingsResourceModel, ctx context.Context) (*gitlab.Pages, error) {
	options := gitlab.UpdatePagesOptions{
		PagesUniqueDomainEnabled: gitlab.Ptr(data.IsUniqueDomainEnabled.ValueBool()),
		PagesHTTPSOnly:           gitlab.Ptr(data.ForceHTTPS.ValueBool()),
	}
	settings, _, err := r.client.Pages.UpdatePages(project, options, gitlab.WithContext(ctx))
	return settings, err
}

func (d *gitlabProjectPagesSettingsResourceModel) settingsToStateModel(ctx context.Context, project string, settings *gitlab.Pages) diag.Diagnostics {
	d.ID = types.StringValue(project)
	d.Project = types.StringValue(project)
	d.URL = types.StringValue(settings.URL)
	d.IsUniqueDomainEnabled = types.BoolValue(settings.IsUniqueDomainEnabled)
	d.ForceHTTPS = types.BoolValue(settings.ForceHTTPS)
	deployments := []gitlabProjectPagesSettingsDeploymentResourceModel{}
	for _, deployment := range settings.Deployments {
		deployments = append(deployments, gitlabProjectPagesSettingsDeploymentResourceModel{
			CreatedAt:     types.StringValue(deployment.CreatedAt.Format(time.RFC3339)),
			URL:           types.StringValue(deployment.URL),
			PathPrefix:    types.StringValue(deployment.PathPrefix),
			RootDirectory: types.StringValue(deployment.RootDirectory),
		})
	}
	deploymentsList, diags := types.ListValueFrom(ctx, projectPagesDeploymentsSchema().NestedObject.Type(), deployments)
	d.Deployments = deploymentsList
	return diags
}
