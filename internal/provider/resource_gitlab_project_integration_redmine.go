package provider

import (
	"context"
	"fmt"
	"regexp"

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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIntegrationRedmineResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationRedmineResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationRedmineResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationRedmineResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationRedmineResource)
}

func NewGitLabProjectIntegrationRedmineResource() resource.Resource {
	return &gitlabProjectIntegrationRedmineResource{
		ResourceName: "_project_integration_redmine",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_redmine` + "`" + ` resource manages the lifecycle of a project integration with Redmine.

~> Using Redmine requires that GitLab internal issue tracking is disabled for the project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#redmine)`,
	}
}

func NewGitLabIntegrationRedmineResource() resource.Resource {
	return &gitlabProjectIntegrationRedmineResource{
		ResourceName: "_integration_redmine",
		ResourceDescription: `The ` + "`" + `gitlab_integration_redmine` + "`" + ` resource manages the lifecycle of a project integration with Redmine.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_redmine` + "`" + `instead.
~> Using Redmine requires that GitLab internal issue tracking is disabled for the project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#redmine)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_redmine` instead.",
	}
}

type gitlabProjectIntegrationRedmineResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_redmine`
	// and `gitlab_integration_redmine` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationRedmineResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Project              types.String `tfsdk:"project"`
	NewIssueURL          types.String `tfsdk:"new_issue_url"`
	ProjectURL           types.String `tfsdk:"project_url"`
	IssuesURL            types.String `tfsdk:"issues_url"`
	UseInheritedSettings types.Bool   `tfsdk:"use_inherited_settings"`
}

func (r *gitlabProjectIntegrationRedmineResourceModel) redmineServiceToStateModel(projectId string, service *gitlab.RedmineService) {
	r.ID = types.StringValue(projectId)
	r.Project = types.StringValue(projectId)
	r.NewIssueURL = types.StringValue(service.Properties.NewIssueURL)
	r.ProjectURL = types.StringValue(service.Properties.ProjectURL)
	r.IssuesURL = types.StringValue(service.Properties.IssuesURL)
	r.UseInheritedSettings = types.BoolValue(bool(service.Properties.UseInheritedSettings))
}

func (r *gitlabProjectIntegrationRedmineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationRedmineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: r.ResourceDescription,
		DeprecationMessage:  r.DeprecationMessage,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the resource. Matches the `project` value.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID of the project you want to activate integration on.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"new_issue_url": schema.StringAttribute{
				MarkdownDescription: "The URL to use to create a new issue in the Redmine project linked to this GitLab project.",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"project_url": schema.StringAttribute{
				MarkdownDescription: "The URL to the Redmine project to link to this GitLab project.",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"issues_url": schema.StringAttribute{
				MarkdownDescription: "The URL to the Redmine project issue to link to this GitLab project.",
				Required:            true,
				Validators: []validator.String{utils.HttpUrlValidator, stringvalidator.RegexMatches(
					regexp.MustCompile(`:id\b`),
					"must contain ':id'. GitLab replaces this ID with the issue number",
				)},
			},
			"use_inherited_settings": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether or not to inherit default settings. Defaults to false.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *gitlabProjectIntegrationRedmineResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationRedmineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectIntegrationRedmineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabProjectIntegrationRedmineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Update Redmine integration.", map[string]any{
		"project":                data.Project,
		"new_issue_url":          data.NewIssueURL,
		"project_url":            data.ProjectURL,
		"issues_url":             data.IssuesURL,
		"use_inherited_settings": data.UseInheritedSettings,
	})

	err := r.updateRedmineService(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Redmine integration", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationRedmineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabProjectIntegrationRedmineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.ID.ValueString()
	redmineService, _, err := r.client.Services.GetRedmineService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Redmine integration doesn't exist, removing from state", map[string]any{
				"project":     data.Project,
				"project_url": data.ProjectURL,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading Redmine integration for project %s", err.Error()))
		return
	}

	data.redmineServiceToStateModel(projectId, redmineService)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationRedmineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabProjectIntegrationRedmineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Update Redmine integration.", map[string]any{
		"project":                data.Project,
		"new_issue_url":          data.NewIssueURL,
		"project_url":            data.ProjectURL,
		"issues_url":             data.IssuesURL,
		"use_inherited_settings": data.UseInheritedSettings,
	})

	err := r.updateRedmineService(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to updateRedmineService Redmine integration", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationRedmineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationRedmineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.ID.ValueString()
	if _, err := r.client.Services.DeleteRedmineService(projectId, gitlab.WithContext(ctx)); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Redmine integration doesn't exist, removing from state", map[string]any{
				"project":     data.Project,
				"project_url": data.ProjectURL,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting Redmine integration for project %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationRedmineResource) updateRedmineService(ctx context.Context, data *gitlabProjectIntegrationRedmineResourceModel) error {
	options := &gitlab.SetRedmineServiceOptions{
		NewIssueURL:          data.NewIssueURL.ValueStringPointer(),
		ProjectURL:           data.ProjectURL.ValueStringPointer(),
		IssuesURL:            data.IssuesURL.ValueStringPointer(),
		UseInheritedSettings: data.UseInheritedSettings.ValueBoolPointer(),
	}

	projectId := data.Project.ValueString()
	if _, _, err := r.client.Services.SetRedmineService(projectId, options, gitlab.WithContext(ctx)); err != nil {
		return err
	}

	redmineService, _, err := r.client.Services.GetRedmineService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		return err
	}

	data.redmineServiceToStateModel(projectId, redmineService)
	return nil
}
