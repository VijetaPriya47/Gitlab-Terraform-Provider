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
	_ resource.Resource                = &gitlabProjectIntegrationMicrosoftTeamsResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationMicrosoftTeamsResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationMicrosoftTeamsResource{}
	_ resource.ResourceWithMoveState   = &gitlabProjectIntegrationMicrosoftTeamsResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationMicrosoftTeamsResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationMicrosoftTeamsResource)
}

func NewGitLabProjectIntegrationMicrosoftTeamsResource() resource.Resource {
	return &gitlabProjectIntegrationMicrosoftTeamsResource{
		ResourceName: "_project_integration_microsoft_teams",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_microsoft_teams` + "`" + ` resource manages the lifecycle of a project integration with Microsoft Teams.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#microsoft-teams-notifications)`,
	}
}

// Remove in 19.0
func NewGitLabIntegrationMicrosoftTeamsResource() resource.Resource {
	return &gitlabProjectIntegrationMicrosoftTeamsResource{
		ResourceName: "_integration_microsoft_teams",
		ResourceDescription: `The ` + "`" + `gitlab_integration_microsoft_teams` + "`" + ` resource manages the lifecycle of a project integration with Microsoft Teams.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_microsoft_teams` + "`" + ` instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#microsoft-teams-notifications)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_microsoft_teams` instead.",
	}
}

type gitlabProjectIntegrationMicrosoftTeamsResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_microsoft_teams`
	// and `gitlab_integration_microsoft_teams` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationMicrosoftTeamsResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Project                   types.String `tfsdk:"project"`
	Webhook                   types.String `tfsdk:"webhook"`
	NotifyOnlyBrokenPipelines types.Bool   `tfsdk:"notify_only_broken_pipelines"`
	BranchesToBeNotified      types.String `tfsdk:"branches_to_be_notified"`
	PushEvents                types.Bool   `tfsdk:"push_events"`
	IssuesEvents              types.Bool   `tfsdk:"issues_events"`
	ConfidentialIssuesEvents  types.Bool   `tfsdk:"confidential_issues_events"`
	MergeRequestsEvents       types.Bool   `tfsdk:"merge_requests_events"`
	TagPushEvents             types.Bool   `tfsdk:"tag_push_events"`
	NoteEvents                types.Bool   `tfsdk:"note_events"`
	ConfidentialNoteEvents    types.Bool   `tfsdk:"confidential_note_events"`
	PipelineEvents            types.Bool   `tfsdk:"pipeline_events"`
	WikiPageEvents            types.Bool   `tfsdk:"wiki_page_events"`
	CreatedAt                 types.String `tfsdk:"created_at"`
	UpdatedAt                 types.String `tfsdk:"updated_at"`
	Active                    types.Bool   `tfsdk:"active"`
}

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"webhook": schema.StringAttribute{
				MarkdownDescription: "The Microsoft Teams webhook (Example, https://outlook.office.com/webhook/...). This value cannot be imported.",
				Required:            true,
				Sensitive:           true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"notify_only_broken_pipelines": schema.BoolAttribute{
				MarkdownDescription: "Send notifications for broken pipelines.",
				Optional:            true,
				Computed:            true,
			},
			"branches_to_be_notified": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Branches to send notifications for. Valid values are: %s", utils.RenderValueListForDocs(api.ValidBranchesToBeNotified)),
				Optional:            true,
				Computed:            true,
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push events.",
				Optional:            true,
				Computed:            true,
			},
			"issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for issue events.",
				Optional:            true,
				Computed:            true,
			},
			"confidential_issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential issue events.",
				Optional:            true,
				Computed:            true,
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge request events.",
				Optional:            true,
				Computed:            true,
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for tag push events.",
				Optional:            true,
				Computed:            true,
			},
			"note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for note events.",
				Optional:            true,
				Computed:            true,
			},
			"confidential_note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential note events.",
				Optional:            true,
				Computed:            true,
			},
			"pipeline_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for pipeline events.",
				Optional:            true,
				Computed:            true,
			},
			"wiki_page_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for wiki page events.",
				Optional:            true,
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

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectIntegrationMicrosoftTeamsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error creating Microsoft Teams integration for project: %s", err.Error()))
		return
	}

	projectID := data.Project.ValueString()
	data.ID = types.StringValue(projectID)
	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectIntegrationMicrosoftTeamsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()
	service, _, err := r.client.Services.GetMicrosoftTeamsService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Microsoft Teams integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading Microsoft Teams integration for project: %s", err.Error()))
		return
	}

	data.modelToStateModel(service, projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectIntegrationMicrosoftTeamsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.update(ctx, data)
	if err != nil {
		if api.Is404(err) {
			tflog.Warn(ctx, "Microsoft Teams integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error updating Microsoft Teams integration for project: %s", err.Error()))
		return
	}

	data.modelToStateModel(service, data.Project.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) update(ctx context.Context, data *gitlabProjectIntegrationMicrosoftTeamsResourceModel) (*gitlab.MicrosoftTeamsService, error) {
	projectID := data.Project.ValueString()
	options := &gitlab.SetMicrosoftTeamsServiceOptions{
		WebHook:                   data.Webhook.ValueStringPointer(),
		NotifyOnlyBrokenPipelines: data.NotifyOnlyBrokenPipelines.ValueBoolPointer(),
		BranchesToBeNotified:      data.BranchesToBeNotified.ValueStringPointer(),
		PushEvents:                data.PushEvents.ValueBoolPointer(),
		IssuesEvents:              data.IssuesEvents.ValueBoolPointer(),
		ConfidentialIssuesEvents:  data.ConfidentialIssuesEvents.ValueBoolPointer(),
		MergeRequestsEvents:       data.MergeRequestsEvents.ValueBoolPointer(),
		TagPushEvents:             data.TagPushEvents.ValueBoolPointer(),
		NoteEvents:                data.NoteEvents.ValueBoolPointer(),
		ConfidentialNoteEvents:    data.ConfidentialNoteEvents.ValueBoolPointer(),
		PipelineEvents:            data.PipelineEvents.ValueBoolPointer(),
		WikiPageEvents:            data.WikiPageEvents.ValueBoolPointer(),
	}

	tflog.Debug(ctx, "Update GitLab Microsoft Teams integration", map[string]any{
		"options": options,
	})

	_, _, err := r.client.Services.SetMicrosoftTeamsService(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	service, _, err := r.client.Services.GetMicrosoftTeamsService(projectID, gitlab.WithContext(ctx))
	return service, err
}

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectIntegrationMicrosoftTeamsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	_, err := r.client.Services.DeleteMicrosoftTeamsService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "Microsoft Teams integration doesn't exist, removing from state", map[string]any{
				"project": data.Project,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting Microsoft Teams integration for project: %s", err.Error()))
		return
	}
}

func (r *gitlabProjectIntegrationMicrosoftTeamsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectIntegrationMicrosoftTeamsResourceModel) modelToStateModel(service *gitlab.MicrosoftTeamsService, projectID string) {
	d.Project = types.StringValue(projectID)
	d.NotifyOnlyBrokenPipelines = types.BoolValue(bool(service.Properties.NotifyOnlyBrokenPipelines))
	d.BranchesToBeNotified = types.StringValue(service.Properties.BranchesToBeNotified)
	d.PushEvents = types.BoolValue(service.PushEvents)
	d.IssuesEvents = types.BoolValue(service.IssuesEvents)
	d.ConfidentialIssuesEvents = types.BoolValue(service.ConfidentialIssuesEvents)
	d.MergeRequestsEvents = types.BoolValue(service.MergeRequestsEvents)
	d.TagPushEvents = types.BoolValue(service.TagPushEvents)
	d.NoteEvents = types.BoolValue(service.NoteEvents)
	d.ConfidentialNoteEvents = types.BoolValue(service.ConfidentialNoteEvents)
	d.PipelineEvents = types.BoolValue(service.PipelineEvents)
	d.WikiPageEvents = types.BoolValue(service.WikiPageEvents)
	d.Active = types.BoolValue(service.Active)
	d.CreatedAt = types.StringValue(service.CreatedAt.Format(time.RFC3339))
	if service.UpdatedAt == nil {
		d.UpdatedAt = types.StringNull()
	} else {
		d.UpdatedAt = types.StringValue(service.UpdatedAt.Format(time.RFC3339))
	}
}

// MoveState implements the ResourceWithMoveState interface to support moving state from the deprecated gitlab_integration_microsoft_teams resource.
// This enables users to migrate from gitlab_integration_microsoft_teams to gitlab_project_integration_microsoft_teams using Terraform's moved block.
// Note: Cross-resource-type state moves require Terraform 1.8 or later.
func (r *gitlabProjectIntegrationMicrosoftTeamsResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		// This first StateMover implements the migration from
		// `gitlab_integration_microsoft_teams` -> `gitlab_project_integration_microsoft_teams`.
		// The SourceSchema needs to match the deprecated `gitlab_integration_microsoft_teams` as a result.
		{
			SourceSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"project": schema.StringAttribute{
						Required: true,
					},
					"webhook": schema.StringAttribute{
						Required:  true,
						Sensitive: true,
					},
					"notify_only_broken_pipelines": schema.BoolAttribute{
						Computed: true,
					},
					"branches_to_be_notified": schema.StringAttribute{
						Computed: true,
					},
					"push_events": schema.BoolAttribute{
						Computed: true,
					},
					"issues_events": schema.BoolAttribute{
						Computed: true,
					},
					"confidential_issues_events": schema.BoolAttribute{
						Computed: true,
					},
					"merge_requests_events": schema.BoolAttribute{
						Computed: true,
					},
					"tag_push_events": schema.BoolAttribute{
						Computed: true,
					},
					"note_events": schema.BoolAttribute{
						Computed: true,
					},
					"confidential_note_events": schema.BoolAttribute{
						Computed: true,
					},
					"pipeline_events": schema.BoolAttribute{
						Computed: true,
					},
					"wiki_page_events": schema.BoolAttribute{
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
				// Only handle moves from gitlab_integration_microsoft_teams resource
				if req.SourceTypeName != "gitlab_integration_microsoft_teams" {
					resp.Diagnostics.AddError("Invalid source resource type", fmt.Sprintf("Expected source type 'gitlab_integration_microsoft_teams', got '%s'", req.SourceTypeName))
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

				// Define the source model matching the old gitlab_integration_microsoft_teams schema
				type sourceModel struct {
					ID                        types.String `tfsdk:"id"`
					Project                   types.String `tfsdk:"project"`
					Webhook                   types.String `tfsdk:"webhook"`
					NotifyOnlyBrokenPipelines types.Bool   `tfsdk:"notify_only_broken_pipelines"`
					BranchesToBeNotified      types.String `tfsdk:"branches_to_be_notified"`
					PushEvents                types.Bool   `tfsdk:"push_events"`
					IssuesEvents              types.Bool   `tfsdk:"issues_events"`
					ConfidentialIssuesEvents  types.Bool   `tfsdk:"confidential_issues_events"`
					MergeRequestsEvents       types.Bool   `tfsdk:"merge_requests_events"`
					TagPushEvents             types.Bool   `tfsdk:"tag_push_events"`
					NoteEvents                types.Bool   `tfsdk:"note_events"`
					ConfidentialNoteEvents    types.Bool   `tfsdk:"confidential_note_events"`
					PipelineEvents            types.Bool   `tfsdk:"pipeline_events"`
					WikiPageEvents            types.Bool   `tfsdk:"wiki_page_events"`
					CreatedAt                 types.String `tfsdk:"created_at"`
					UpdatedAt                 types.String `tfsdk:"updated_at"`
					Active                    types.Bool   `tfsdk:"active"`
				}

				var sourceStateData sourceModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				project := sourceStateData.ID.ValueString()

				// Create the target state data
				targetStateData := gitlabProjectIntegrationMicrosoftTeamsResourceModel{
					ID:                        types.StringValue(project),
					Project:                   sourceStateData.Project,
					Webhook:                   sourceStateData.Webhook,
					NotifyOnlyBrokenPipelines: sourceStateData.NotifyOnlyBrokenPipelines,
					BranchesToBeNotified:      sourceStateData.BranchesToBeNotified,
					PushEvents:                sourceStateData.PushEvents,
					IssuesEvents:              sourceStateData.IssuesEvents,
					ConfidentialIssuesEvents:  sourceStateData.ConfidentialIssuesEvents,
					MergeRequestsEvents:       sourceStateData.MergeRequestsEvents,
					TagPushEvents:             sourceStateData.TagPushEvents,
					NoteEvents:                sourceStateData.NoteEvents,
					ConfidentialNoteEvents:    sourceStateData.ConfidentialNoteEvents,
					PipelineEvents:            sourceStateData.PipelineEvents,
					WikiPageEvents:            sourceStateData.WikiPageEvents,
					CreatedAt:                 sourceStateData.CreatedAt,
					UpdatedAt:                 sourceStateData.UpdatedAt,
					Active:                    sourceStateData.Active,
				}

				tflog.Debug(ctx, "Moving state from gitlab_integration_microsoft_teams to gitlab_project_integration_microsoft_teams")
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}
