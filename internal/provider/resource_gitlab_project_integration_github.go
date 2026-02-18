package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	_ resource.Resource                = &gitlabProjectIntegrationGithubResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationGithubResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationGithubResource{}
	_ resource.ResourceWithMoveState   = &gitlabProjectIntegrationGithubResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationGithubResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationGithubResource)
}

func NewGitLabProjectIntegrationGithubResource() resource.Resource {
	return &gitlabProjectIntegrationGithubResource{
		ResourceName: "_project_integration_github",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_github` + "`" + ` resource manages the lifecycle of a project integration with GitHub.

-> This resource requires a GitLab Enterprise instance.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#github)`,
	}
}

// Remove in 19.0
func NewGitLabIntegrationGithubResource() resource.Resource {
	return &gitlabProjectIntegrationGithubResource{
		ResourceName: "_integration_github",
		ResourceDescription: `The ` + "`" + `gitlab_integration_github` + "`" + ` resource manages the lifecycle of a project integration with GitHub.

-> This resource requires a GitLab Enterprise instance.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_github` + "`" + ` instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#github)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_github` instead.",
	}
}

type gitlabProjectIntegrationGithubResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_github`
	// and `gitlab_integration_github` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationGithubResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Project       types.String `tfsdk:"project"`
	Token         types.String `tfsdk:"token"`
	RepositoryURL types.String `tfsdk:"repository_url"`
	StaticContext types.Bool   `tfsdk:"static_context"`
	Title         types.String `tfsdk:"title"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
	Active        types.Bool   `tfsdk:"active"`
}

func (r *gitlabProjectIntegrationGithubResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationGithubResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				MarkdownDescription: "ID of the project you want to activate the integration on.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "A GitHub personal access token with at least the `repo:status` scope.",
				Required:            true,
				Sensitive:           true,
			},
			"repository_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the GitHub repo to integrate with. For example, https://github.com/gitlabhq/terraform-provider-gitlab.",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"static_context": schema.BoolAttribute{
				MarkdownDescription: "Append the instance name instead of the branch to the status. Must enable to set a GitLab status check as _required_ in GitHub. See [Static / dynamic status check names] to learn more.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Default:             booldefault.StaticBool(true),
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Title of the integration.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this integration was activated at in UTC.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The ISO8601 date/time that this integration was last updated at in UTC.",
				Computed:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the integration is active.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabProjectIntegrationGithubResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationGithubResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectIntegrationGithubResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error creating GitHub integration for project: %s", err.Error()))
		return
	}

	projectID := data.Project.ValueString()
	data.ID = types.StringValue(projectID)
	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationGithubResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectIntegrationGithubResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()
	service, _, err := r.client.Services.GetGithubService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "GitHub integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading GitHub integration for project: %s", err.Error()))
		return
	}

	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationGithubResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectIntegrationGithubResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "GitHub integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error updating GitHub integration for project: %s", err.Error()))
		return
	}

	data.modelToStateModel(service, data.Project.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationGithubResource) update(ctx context.Context, data *gitlabProjectIntegrationGithubResourceModel) (*gitlab.GithubService, error) {
	projectID := data.Project.ValueString()
	options := &gitlab.SetGithubServiceOptions{
		Token:         gitlab.Ptr(data.Token.ValueString()),
		RepositoryURL: gitlab.Ptr(data.RepositoryURL.ValueString()),
		StaticContext: data.StaticContext.ValueBoolPointer(),
	}

	tflog.Debug(ctx, "Update GitLab GitHub integration", map[string]any{
		"options": options,
	})

	_, _, err := r.client.Services.SetGithubService(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	service, _, err := r.client.Services.GetGithubService(projectID, gitlab.WithContext(ctx))
	return service, err
}

func (r *gitlabProjectIntegrationGithubResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationGithubResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	_, err := r.client.Services.DeleteGithubService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "GitHub integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting GitHub integration for project: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationGithubResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectIntegrationGithubResourceModel) modelToStateModel(service *gitlab.GithubService, projectID string) {
	d.Project = types.StringValue(projectID)
	d.RepositoryURL = types.StringValue(service.Properties.RepositoryURL)
	d.StaticContext = types.BoolValue(service.Properties.StaticContext)
	d.Title = types.StringValue(service.Title)
	d.Active = types.BoolValue(service.Active)
	d.CreatedAt = types.StringValue(service.CreatedAt.Format(time.RFC3339))
	if service.UpdatedAt == nil {
		d.UpdatedAt = types.StringNull()
	} else {
		d.UpdatedAt = types.StringValue(service.UpdatedAt.Format(time.RFC3339))
	}
}

// MoveState implements the ResourceWithMoveState interface to support moving state from the deprecated gitlab_integration_github resource.
// This enables users to migrate from gitlab_integration_github to gitlab_project_integration_github using Terraform's moved block.
// Note: Cross-resource-type state moves require Terraform 1.8 or later.
func (r *gitlabProjectIntegrationGithubResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		// This first StateMover implements the migration from
		// `gitlab_integration_github` -> `gitlab_project_integration_github`.
		// The SourceSchema needs to match the deprecated `gitlab_integration_github` as a result.
		{
			SourceSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"project": schema.StringAttribute{
						Required: true,
					},
					"token": schema.StringAttribute{
						Required:  true,
						Sensitive: true,
					},
					"repository_url": schema.StringAttribute{
						Required: true,
					},
					"static_context": schema.BoolAttribute{
						Computed: true,
					},
					"title": schema.StringAttribute{
						Computed: true,
					},
					"created_at": schema.StringAttribute{
						Computed: true,
					},
					"updated_at": schema.StringAttribute{
						Computed: true,
					},
					"active": schema.BoolAttribute{
						Computed: true,
					},
				},
			},
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Only handle moves from gitlab_integration_github resource
				if req.SourceTypeName != "gitlab_integration_github" {
					resp.Diagnostics.AddError("Invalid source resource type", fmt.Sprintf("Expected source type 'gitlab_integration_github', got '%s'", req.SourceTypeName))
					return
				}

				// Check provider address (without hostname for compatibility)
				// Accept anything that ends with gitlab, which seems the safest.
				//  hashicorp/gitlab is used in tests
				//  gitlab-org/gitlab is used in production
				//  gitlabhq/gitlab is referenced on the provider docs.
				if !strings.HasSuffix(req.SourceProviderAddress, "gitlab") {
					resp.Diagnostics.AddError("Invalid source provider address", fmt.Sprintf("Expected provider address ending with 'gitlab', got '%s'", req.SourceProviderAddress))
					return
				}

				// Define the source model matching the old gitlab_integration_github schema
				type sourceModel struct {
					ID            types.String `tfsdk:"id"`
					Project       types.String `tfsdk:"project"`
					Token         types.String `tfsdk:"token"`
					RepositoryURL types.String `tfsdk:"repository_url"`
					StaticContext types.Bool   `tfsdk:"static_context"`
					Title         types.String `tfsdk:"title"`
					CreatedAt     types.String `tfsdk:"created_at"`
					UpdatedAt     types.String `tfsdk:"updated_at"`
					Active        types.Bool   `tfsdk:"active"`
				}

				var sourceStateData sourceModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				project := sourceStateData.ID.ValueString()

				// Create the target state data
				targetStateData := gitlabProjectIntegrationGithubResourceModel{
					ID:            types.StringValue(project),
					Project:       sourceStateData.Project,
					Token:         sourceStateData.Token,
					RepositoryURL: sourceStateData.RepositoryURL,
					StaticContext: sourceStateData.StaticContext,
					Title:         sourceStateData.Title,
					CreatedAt:     sourceStateData.CreatedAt,
					UpdatedAt:     sourceStateData.UpdatedAt,
					Active:        sourceStateData.Active,
				}

				tflog.Debug(ctx, "Moving state from gitlab_integration_github to gitlab_project_integration_github")
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}
