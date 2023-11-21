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

	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabIntegrationCustomIssueTrackerResource{}
	_ resource.ResourceWithConfigure   = &gitlabIntegrationCustomIssueTrackerResource{}
	_ resource.ResourceWithImportState = &gitlabIntegrationCustomIssueTrackerResource{}
)

func init() {
	registerResource(NewGitlabIntegrationCustomIssueTrackerResource)
}

func NewGitlabIntegrationCustomIssueTrackerResource() resource.Resource {
	return &gitlabIntegrationCustomIssueTrackerResource{}
}

type gitlabIntegrationCustomIssueTrackerResourceModel struct {
	Id         types.String `tfsdk:"id"`
	Project    types.String `tfsdk:"project"`
	ProjectURL types.String `tfsdk:"project_url"`
	IssuesURL  types.String `tfsdk:"issues_url"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
	Slug       types.String `tfsdk:"slug"`
	Active     types.Bool   `tfsdk:"active"`
}

func (r *gitlabIntegrationCustomIssueTrackerResourceModel) customIssueTrackerServiceToStateModel(service *gitlab.CustomIssueTrackerService, projectId string) {
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

type gitlabIntegrationCustomIssueTrackerResource struct {
	client *gitlab.Client
}

func (r *gitlabIntegrationCustomIssueTrackerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_custom_issue_tracker"
}

func (r *gitlabIntegrationCustomIssueTrackerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_integration_custom_issue_tracker`" + ` resource allows to manage the lifecycle of a project integration with Custom Issue Tracker.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/integrations.html#custom-issue-tracker)`,

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

func (r *gitlabIntegrationCustomIssueTrackerResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

func (r *gitlabIntegrationCustomIssueTrackerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create custom issue tracker service", err.Error())
	}
}

func (r *gitlabIntegrationCustomIssueTrackerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabIntegrationCustomIssueTrackerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.Id.ValueString()

	service, _, err := r.client.Services.GetCustomIssueTrackerService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "custom issue tracker integration doesn't exist, removing from state", map[string]interface{}{
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

func (r *gitlabIntegrationCustomIssueTrackerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update custom issue tracker integration", err.Error())
	}
}

func (r *gitlabIntegrationCustomIssueTrackerResource) update(ctx context.Context, plan *tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) error {
	var data gitlabIntegrationCustomIssueTrackerResourceModel
	diags.Append(plan.Get(ctx, &data)...)
	if diags.HasError() {
		return nil
	}
	projectId := data.Project.ValueString()

	options := &gitlab.SetCustomIssueTrackerServiceOptions{
		ProjectURL: gitlab.Ptr(data.ProjectURL.ValueString()),
		IssuesURL:  gitlab.Ptr(data.IssuesURL.ValueString()),
		// According to [Custom Issue Tracker documentation](https://docs.gitlab.com/ee/user/project/integrations/custom_issue_tracker.html#enable-a-custom-issue-tracker)
		// new_issue_url isn't used, but required by API and have to be a valid URL.
		NewIssueURL: gitlab.Ptr(data.ProjectURL.ValueString()),
	}

	if _, err := r.client.Services.SetCustomIssueTrackerService(projectId, options, gitlab.WithContext(ctx)); err != nil {
		return err
	}

	service, _, err := r.client.Services.GetCustomIssueTrackerService(projectId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "custom issue tracker integration doesn't exist right after creation, removing from state", map[string]interface{}{
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

func (r *gitlabIntegrationCustomIssueTrackerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabIntegrationCustomIssueTrackerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.Id.ValueString()

	if _, err := r.client.Services.DeleteCustomIssueTrackerService(projectId, gitlab.WithContext(ctx)); err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "custom issue tracker integration doesn't exist, removing from state", map[string]interface{}{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting custom issue tracker integration for project %s", err.Error()))
		return
	}
}

func (r *gitlabIntegrationCustomIssueTrackerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
