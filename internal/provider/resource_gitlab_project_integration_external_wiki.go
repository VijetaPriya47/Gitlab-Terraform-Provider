package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	_ resource.Resource                = &gitlabProjectIntegrationExternalWikiResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationExternalWikiResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationExternalWikiResource{}
	_ resource.ResourceWithMoveState   = &gitlabProjectIntegrationExternalWikiResource{}
)

func init() {
	registerResource(NewGitlabProjectIntegrationExternalWikiResource)

	// Remove in 19.0
	registerResource(NewGitlabIntegrationExternalWikiResource)
}

func NewGitlabProjectIntegrationExternalWikiResource() resource.Resource {
	return &gitlabProjectIntegrationExternalWikiResource{
		ResourceName: "_project_integration_external_wiki",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_external_wiki` + "`" + ` resource manages the lifecycle of a project integration with the External Wiki Service.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#external-wiki)`,
	}
}

// Remove in 19.0
func NewGitlabIntegrationExternalWikiResource() resource.Resource {
	return &gitlabProjectIntegrationExternalWikiResource{
		ResourceName: "_integration_external_wiki",
		ResourceDescription: `The ` + "`" + `gitlab_integration_external_wiki` + "`" + ` resource manages the lifecycle of a project integration with the External Wiki Service.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_external_wiki` + "`" + ` instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#external-wiki)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_external_wiki` instead.",
	}
}

type gitlabProjectIntegrationExternalWikiResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_external_wiki`
	// and `gitlab_integration_external_wiki` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationExternalWikiResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Project         types.String `tfsdk:"project"`
	ExternalWikiURL types.String `tfsdk:"external_wiki_url"`
	Title           types.String `tfsdk:"title"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	Slug            types.String `tfsdk:"slug"`
	Active          types.Bool   `tfsdk:"active"`
}

func (r *gitlabProjectIntegrationExternalWikiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationExternalWikiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				MarkdownDescription: "ID of the project you want to activate integration on.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"external_wiki_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the external wiki.",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
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

func (r *gitlabProjectIntegrationExternalWikiResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationExternalWikiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectIntegrationExternalWikiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error creating external wiki integration for project %s", err.Error()))
		return
	}

	projectID := data.Project.ValueString()
	data.ID = types.StringValue(projectID)
	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationExternalWikiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectIntegrationExternalWikiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()
	service, _, err := r.client.Services.GetExternalWikiService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "external wiki integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading external wiki integration for project %s", err.Error()))
		return
	}

	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationExternalWikiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectIntegrationExternalWikiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "external wiki integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error updating external wiki integration for project %s", err.Error()))
		return
	}

	data.modelToStateModel(service, data.Project.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationExternalWikiResource) update(ctx context.Context, data *gitlabProjectIntegrationExternalWikiResourceModel) (*gitlab.ExternalWikiService, error) {
	projectID := data.Project.ValueString()
	options := &gitlab.SetExternalWikiServiceOptions{
		ExternalWikiURL: gitlab.Ptr(data.ExternalWikiURL.ValueString()),
	}

	tflog.Debug(ctx, "Update GitLab External Wiki integration", map[string]any{
		"options": options,
	})

	_, _, err := r.client.Services.SetExternalWikiService(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	service, _, err := r.client.Services.GetExternalWikiService(projectID, gitlab.WithContext(ctx))
	return service, err
}

func (r *gitlabProjectIntegrationExternalWikiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationExternalWikiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	_, err := r.client.Services.DeleteExternalWikiService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "external wiki integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting external wiki integration for project %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationExternalWikiResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectIntegrationExternalWikiResourceModel) modelToStateModel(service *gitlab.ExternalWikiService, projectID string) {
	d.ID = types.StringValue(projectID)
	d.Project = types.StringValue(projectID)
	d.ExternalWikiURL = types.StringValue(service.Properties.ExternalWikiURL)
	d.Title = types.StringValue(service.Title)
	d.Slug = types.StringValue(service.Slug)
	d.Active = types.BoolValue(service.Active)
	d.CreatedAt = types.StringValue(service.CreatedAt.Format(time.RFC3339))
	if service.UpdatedAt != nil {
		d.UpdatedAt = types.StringValue(service.UpdatedAt.Format(time.RFC3339))
	} else {
		d.UpdatedAt = types.StringNull()
	}
}

// MoveState implements the ResourceWithMoveState interface to support moving state from the deprecated gitlab_integration_external_wiki resource.
// This enables users to migrate from gitlab_integration_external_wiki to gitlab_project_integration_external_wiki using Terraform's moved block.
// Note: Cross-resource-type state moves require Terraform 1.8 or later.
func (r *gitlabProjectIntegrationExternalWikiResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		// This first StateMover implements the migration from
		// `gitlab_integration_external_wiki` -> `gitlab_project_integration_external_wiki`.
		// The SourceSchema needs to match the deprecated `gitlab_integration_external_wiki` as a result.
		{
			SourceSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"project": schema.StringAttribute{
						Required: true,
					},
					"external_wiki_url": schema.StringAttribute{
						Required: true,
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
					"slug": schema.StringAttribute{
						Computed: true,
					},
					"active": schema.BoolAttribute{
						Computed: true,
					},
				},
			},
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Only handle moves from gitlab_integration_external_wiki resource
				if req.SourceTypeName != "gitlab_integration_external_wiki" {
					resp.Diagnostics.AddError("Invalid source resource type", fmt.Sprintf("Expected source type 'gitlab_integration_external_wiki', got '%s'", req.SourceTypeName))
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

				// Define the source model matching the old gitlab_integration_external_wiki schema
				type sourceModel struct {
					Id              types.String `tfsdk:"id"`
					Project         types.String `tfsdk:"project"`
					ExternalWikiURL types.String `tfsdk:"external_wiki_url"`
					Title           types.String `tfsdk:"title"`
					CreatedAt       types.String `tfsdk:"created_at"`
					UpdatedAt       types.String `tfsdk:"updated_at"`
					Slug            types.String `tfsdk:"slug"`
					Active          types.Bool   `tfsdk:"active"`
				}

				var sourceStateData sourceModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				project := sourceStateData.Id.ValueString()

				// Create the target state data
				targetStateData := gitlabProjectIntegrationExternalWikiResourceModel{
					ID:              types.StringValue(project),
					Project:         sourceStateData.Project,
					ExternalWikiURL: sourceStateData.ExternalWikiURL,
					Title:           sourceStateData.Title,
					CreatedAt:       sourceStateData.CreatedAt,
					UpdatedAt:       sourceStateData.UpdatedAt,
					Slug:            sourceStateData.Slug,
					Active:          sourceStateData.Active,
				}

				tflog.Debug(ctx, "Moving state from gitlab_integration_external_wiki to gitlab_project_integration_external_wiki")
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}
