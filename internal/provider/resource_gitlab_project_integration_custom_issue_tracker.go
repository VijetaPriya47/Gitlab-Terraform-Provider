package provider

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIntegrationCustomIssueTrackerResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationCustomIssueTrackerResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationCustomIssueTrackerResource{}
)

func init() {
	registerResource(NewGitlabProjectIntegrationCustomIssueTrackerResource)

	// Remove in 19.0
	registerResource(NewGitlabIntegrationCustomIssueTrackerResource)
}

func NewGitlabProjectIntegrationCustomIssueTrackerResource() resource.Resource {
	return &gitlabProjectIntegrationCustomIssueTrackerResource{
		ResourceName: "_project_integration_custom_issue_tracker",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_custom_issue_tracker` + "`" + ` resource manages the lifecycle of a project integration with a Custom Issue Tracker.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#custom-issue-tracker)`,
	}
}

// Remove in 19.0
func NewGitlabIntegrationCustomIssueTrackerResource() resource.Resource {
	return &gitlabProjectIntegrationCustomIssueTrackerResource{
		ResourceName: "_integration_custom_issue_tracker",
		ResourceDescription: `The ` + "`" + `gitlab_integration_custom_issue_tracker` + "`" + ` resource manages the lifecycle of a project integration with a Custom Issue Tracker.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_custom_issue_tracker` + "`" + `instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#custom-issue-tracker)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_custom_issue_tracker` instead.",
	}
}

type gitlabProjectIntegrationCustomIssueTrackerResourceModel struct {
	Id         types.String `tfsdk:"id"`
	Project    types.String `tfsdk:"project"`
	ProjectURL types.String `tfsdk:"project_url"`
	IssuesURL  types.String `tfsdk:"issues_url"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
	Slug       types.String `tfsdk:"slug"`
	Active     types.Bool   `tfsdk:"active"`
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResourceModel) customIssueTrackerServiceToStateModel(service *gitlab.CustomIssueTrackerService, projectId string) {
	r.Id = types.StringValue(projectId)
	r.Project = types.StringValue(projectId)
	r.ProjectURL = types.StringValue(service.Properties.ProjectURL)
	r.IssuesURL = types.StringValue(service.Properties.IssuesURL)
	r.Active = types.BoolValue(service.Active)
	r.Slug = types.StringValue(service.Slug)
	r.CreatedAt = types.StringValue(service.CreatedAt.Format(time.RFC3339))
	if service.UpdatedAt != nil {
		r.UpdatedAt = types.StringValue(service.UpdatedAt.Format(time.RFC3339))
	}
}

type gitlabProjectIntegrationCustomIssueTrackerResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_custom_issue_tracker`
	// and `gitlab_integration_custom_issue_tracker` for backwards compatibility reasons. Should be removed in v19.0
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: r.ResourceDescription,
		DeprecationMessage:  r.DeprecationMessage,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project for the custom issue tracker.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"project_url": schema.StringAttribute{
				MarkdownDescription: "The URL to the project in the external issue tracker.",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"issues_url": schema.StringAttribute{
				MarkdownDescription: "The URL to view an issue in the external issue tracker. Must contain :id.",
				Required:            true,
				Validators: []validator.String{
					utils.HttpUrlValidator,
					stringvalidator.RegexMatches(regexp.MustCompile(`:id`), "value should contain :id placeholder"),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this integration was activated at in UTC.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this integration was last updated at in UTC.",
				Computed:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The name of the integration in lowercase, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.",
				Computed:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the integration is active.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create custom issue tracker service", err.Error())
	}
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabProjectIntegrationCustomIssueTrackerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.Id.ValueString()

	service, _, err := r.client.Services.GetCustomIssueTrackerService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "custom issue tracker integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading custom issue tracker integration for project %s", err.Error()))
		return
	}

	data.customIssueTrackerServiceToStateModel(service, projectId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update custom issue tracker integration", err.Error())
	}
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) update(ctx context.Context, plan *tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) error {
	var data gitlabProjectIntegrationCustomIssueTrackerResourceModel
	diags.Append(plan.Get(ctx, &data)...)
	if diags.HasError() {
		return nil
	}
	projectId := data.Project.ValueString()

	options := &gitlab.SetCustomIssueTrackerServiceOptions{
		ProjectURL: gitlab.Ptr(data.ProjectURL.ValueString()),
		IssuesURL:  gitlab.Ptr(data.IssuesURL.ValueString()),
		// According to [Custom Issue Tracker documentation](https://docs.gitlab.com/user/project/integrations/custom_issue_tracker/#enable-a-custom-issue-tracker)
		// new_issue_url isn't used, but required by API and have to be a valid URL.
		NewIssueURL: gitlab.Ptr(data.ProjectURL.ValueString()),
	}

	if _, _, err := r.client.Services.SetCustomIssueTrackerService(projectId, options, gitlab.WithContext(ctx)); err != nil {
		return err
	}

	service, _, err := r.client.Services.GetCustomIssueTrackerService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "custom issue tracker integration doesn't exist right after creation, removing from state", map[string]any{
				"project": data.Project,
			})
			state.RemoveResource(ctx)
			return nil
		}
		return err
	}

	data.customIssueTrackerServiceToStateModel(service, projectId)

	diags.Append(state.Set(ctx, &data)...)

	return nil
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationCustomIssueTrackerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.Id.ValueString()

	if _, err := r.client.Services.DeleteCustomIssueTrackerService(projectId, gitlab.WithContext(ctx)); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "custom issue tracker integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting custom issue tracker integration for project %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationCustomIssueTrackerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
