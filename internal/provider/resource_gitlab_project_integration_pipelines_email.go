package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIntegrationPipelinesEmailResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationPipelinesEmailResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationPipelinesEmailResource{}
	_ resource.ResourceWithMoveState   = &gitlabProjectIntegrationPipelinesEmailResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationPipelinesEmailResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationPipelinesEmailResource)
}

func NewGitLabProjectIntegrationPipelinesEmailResource() resource.Resource {
	return &gitlabProjectIntegrationPipelinesEmailResource{
		ResourceName: "_project_integration_pipelines_email",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_pipelines_email` + "`" + ` resource manages the lifecycle of a project integration with the Pipeline Emails Service.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#pipeline-status-emails)`,
	}
}

// Remove in 19.0
func NewGitLabIntegrationPipelinesEmailResource() resource.Resource {
	return &gitlabProjectIntegrationPipelinesEmailResource{
		ResourceName: "_integration_pipelines_email",
		ResourceDescription: `The ` + "`" + `gitlab_integration_pipelines_email` + "`" + ` resource manages the lifecycle of a project integration with the Pipeline Emails Service.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_pipelines_email` + "`" + ` instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#pipeline-status-emails)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_pipelines_email` instead.",
	}
}

type gitlabProjectIntegrationPipelinesEmailResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_pipelines_email`
	// and `gitlab_integration_pipelines_email` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationPipelinesEmailResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Project                   types.String `tfsdk:"project"`
	Recipients                types.Set    `tfsdk:"recipients"`
	NotifyOnlyBrokenPipelines types.Bool   `tfsdk:"notify_only_broken_pipelines"`
	BranchesToBeNotified      types.String `tfsdk:"branches_to_be_notified"`
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"recipients": schema.SetAttribute{
				MarkdownDescription: "Email addresses where notifications are sent.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"notify_only_broken_pipelines": schema.BoolAttribute{
				MarkdownDescription: "Notify only broken pipelines. Default is true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"branches_to_be_notified": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Branches to send notifications for. Valid options are %s. Default is `default`.", utils.RenderValueListForDocs(api.ValidBranchesToBeNotified)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidBranchesToBeNotified...)},
			},
		},
	}
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectIntegrationPipelinesEmailResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error creating Pipelines Email integration for project: %s", err.Error()))
		return
	}

	projectID := data.Project.ValueString()
	data.ID = types.StringValue(projectID)
	data.modelToStateModel(ctx, service, projectID, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectIntegrationPipelinesEmailResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()
	service, _, err := r.client.Services.GetPipelinesEmailService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Pipelines Email integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading Pipelines Email integration for project: %s", err.Error()))
		return
	}

	data.modelToStateModel(ctx, service, projectID, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectIntegrationPipelinesEmailResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Pipelines Email integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error updating Pipelines Email integration for project: %s", err.Error()))
		return
	}

	data.modelToStateModel(ctx, service, data.Project.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) update(ctx context.Context, data *gitlabProjectIntegrationPipelinesEmailResourceModel) (*gitlab.PipelinesEmailService, error) {
	projectID := data.Project.ValueString()

	// Convert recipients Set to comma-separated string
	var recipients []string
	data.Recipients.ElementsAs(ctx, &recipients, false)
	recipientsString := strings.Join(recipients, ",")

	options := &gitlab.SetPipelinesEmailServiceOptions{
		Recipients:                &recipientsString,
		NotifyOnlyBrokenPipelines: data.NotifyOnlyBrokenPipelines.ValueBoolPointer(),
		BranchesToBeNotified:      data.BranchesToBeNotified.ValueStringPointer(),
	}

	tflog.Debug(ctx, "Update GitLab Pipelines Email integration", map[string]any{
		"options": options,
	})

	_, _, err := r.client.Services.SetPipelinesEmailService(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	service, _, err := r.client.Services.GetPipelinesEmailService(projectID, gitlab.WithContext(ctx))
	return service, err
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectIntegrationPipelinesEmailResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	_, err := r.client.Services.DeletePipelinesEmailService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Pipelines Email integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting Pipelines Email integration for project: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationPipelinesEmailResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectIntegrationPipelinesEmailResourceModel) modelToStateModel(ctx context.Context, service *gitlab.PipelinesEmailService, projectID string, diags *diag.Diagnostics) {
	d.Project = types.StringValue(projectID)
	d.NotifyOnlyBrokenPipelines = types.BoolValue(bool(service.Properties.NotifyOnlyBrokenPipelines))
	d.BranchesToBeNotified = types.StringValue(service.Properties.BranchesToBeNotified)

	// Convert comma-separated recipients string to Set
	recipientsString := service.Properties.Recipients
	var recipientsList []string
	if recipientsString != "" {
		recipientsList = strings.Split(recipientsString, ",")
		// Trim whitespace from each recipient
		for i, recipient := range recipientsList {
			recipientsList[i] = strings.TrimSpace(recipient)
		}
	}

	recipientsSet, diag := types.SetValueFrom(ctx, types.StringType, recipientsList)
	diags.Append(diag...)
	d.Recipients = recipientsSet
}

// MoveState implements the ResourceWithMoveState interface to support moving state from the deprecated gitlab_integration_pipelines_email resource.
// This enables users to migrate from gitlab_integration_pipelines_email to gitlab_project_integration_pipelines_email using Terraform's moved block.
// Note: Cross-resource-type state moves require Terraform 1.8 or later.
func (r *gitlabProjectIntegrationPipelinesEmailResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		// This first StateMover implements the migration from
		// `gitlab_integration_pipelines_email` -> `gitlab_project_integration_pipelines_email`.
		// The SourceSchema needs to match the deprecated `gitlab_integration_pipelines_email` as a result.
		{
			SourceSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"project": schema.StringAttribute{
						Required: true,
					},
					"recipients": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
					},
					"notify_only_broken_pipelines": schema.BoolAttribute{
						Computed: true,
					},
					"branches_to_be_notified": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Only handle moves from gitlab_integration_pipelines_email resource
				if req.SourceTypeName != "gitlab_integration_pipelines_email" {
					resp.Diagnostics.AddError("Invalid source resource type", fmt.Sprintf("Expected source type 'gitlab_integration_pipelines_email', got '%s'", req.SourceTypeName))
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

				// Define the source model matching the old gitlab_integration_pipelines_email schema
				type sourceModel struct {
					ID                        types.String `tfsdk:"id"`
					Project                   types.String `tfsdk:"project"`
					Recipients                types.Set    `tfsdk:"recipients"`
					NotifyOnlyBrokenPipelines types.Bool   `tfsdk:"notify_only_broken_pipelines"`
					BranchesToBeNotified      types.String `tfsdk:"branches_to_be_notified"`
				}

				var sourceStateData sourceModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				project := sourceStateData.ID.ValueString()

				// Create the target state data
				targetStateData := gitlabProjectIntegrationPipelinesEmailResourceModel{
					ID:                        types.StringValue(project),
					Project:                   sourceStateData.Project,
					Recipients:                sourceStateData.Recipients,
					NotifyOnlyBrokenPipelines: sourceStateData.NotifyOnlyBrokenPipelines,
					BranchesToBeNotified:      sourceStateData.BranchesToBeNotified,
				}

				tflog.Debug(ctx, "Moving state from gitlab_integration_pipelines_email to gitlab_project_integration_pipelines_email")
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}
