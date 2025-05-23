package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &gitlabProjectIntegrationYouTrackResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationYouTrackResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationYouTrackResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationYouTrackResource)
}

func NewGitLabProjectIntegrationYouTrackResource() resource.Resource {
	return &gitlabProjectIntegrationYouTrackResource{}
}

type gitlabProjectIntegrationYouTrackResource struct {
	client *gitlab.Client
}

type gitlabProjectIntegrationYouTrackResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Project    types.String `tfsdk:"project"`
	IssuesURL  types.String `tfsdk:"issues_url"`
	ProjectURL types.String `tfsdk:"project_url"`
}

func (r *gitlabProjectIntegrationYouTrackResourceModel) youTrackServiceToStateModel(project string, service *gitlab.YouTrackService) {
	r.ID = types.StringValue(project)
	r.Project = types.StringValue(project)
	r.IssuesURL = types.StringValue(service.Properties.IssuesURL)
	r.ProjectURL = types.StringValue(service.Properties.ProjectURL)
}

func (r *gitlabProjectIntegrationYouTrackResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_integration_youtrack"
}

func (r *gitlabProjectIntegrationYouTrackResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationYouTrackResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectIntegrationYouTrackResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_integration_youtrack`" + ` resource manages the lifecycle of a project integration with YouTrack.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#youtrack)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the resource. Matches the `project` value.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or namespace of the project you want to activate integration on.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"issues_url": schema.StringAttribute{
				MarkdownDescription: "URL to view an issue in the external issue tracker. Must contain :id.",
				Required:            true,
				Validators: []validator.String{utils.HttpUrlValidator, stringvalidator.RegexMatches(
					regexp.MustCompile(`:id\b`),
					"must contain ':id'. GitLab replaces this ID with the issue number",
				)},
			},
			"project_url": schema.StringAttribute{
				MarkdownDescription: "URL of the project in the external issue tracker.",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
		},
	}
}

func (r *gitlabProjectIntegrationYouTrackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabProjectIntegrationYouTrackResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Create YouTrack integration.", map[string]any{
		"project":     data.Project.String(),
		"issues_url":  data.IssuesURL.String(),
		"project_url": data.ProjectURL.String(),
	})

	err := r.updateYouTrackService(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create YouTrack integration", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationYouTrackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabProjectIntegrationYouTrackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.ID.ValueString()
	service, _, err := r.client.Services.GetYouTrackService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "YouTrack integration doesn't exist, removing from state", map[string]any{
				"project":     data.Project.ValueString(),
				"issues_url":  data.IssuesURL.String(),
				"project_url": data.ProjectURL.String(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading YouTrack integration for project %s", err.Error()))
		return
	}

	data.youTrackServiceToStateModel(project, service)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationYouTrackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabProjectIntegrationYouTrackResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Update YouTrack integration.", map[string]any{
		"project":     data.Project.String(),
		"issues_url":  data.IssuesURL.String(),
		"project_url": data.ProjectURL.String(),
	})

	err := r.updateYouTrackService(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update YouTrack integration", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationYouTrackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationYouTrackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.ID.ValueString()
	_, err := r.client.Services.DeleteYouTrackService(project, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "YouTrack integration doesn't exist, removing from state", map[string]any{
				"project":     data.Project.ValueString(),
				"issues_url":  data.IssuesURL.String(),
				"project_url": data.ProjectURL.String(),
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting YouTrack integration for project %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationYouTrackResource) updateYouTrackService(ctx context.Context, data *gitlabProjectIntegrationYouTrackResourceModel) error {
	options := &gitlab.SetYouTrackServiceOptions{
		IssuesURL:  data.IssuesURL.ValueStringPointer(),
		ProjectURL: data.ProjectURL.ValueStringPointer(),
	}
	project := data.Project.ValueString()
	service, _, err := r.client.Services.SetYouTrackService(project, options, gitlab.WithContext(ctx))
	if err != nil {
		return err
	}

	data.youTrackServiceToStateModel(project, service)
	return nil
}
