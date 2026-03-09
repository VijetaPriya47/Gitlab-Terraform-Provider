package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectIntegrationJenkinsResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectIntegrationJenkinsResource{}
	_ resource.ResourceWithImportState = &gitlabProjectIntegrationJenkinsResource{}
	_ resource.ResourceWithMoveState   = &gitlabProjectIntegrationJenkinsResource{}
)

func init() {
	registerResource(NewGitLabProjectIntegrationJenkinsResource)

	// Remove in 19.0
	registerResource(NewGitLabIntegrationJenkinsResource)
}

func NewGitLabProjectIntegrationJenkinsResource() resource.Resource {
	return &gitlabProjectIntegrationJenkinsResource{
		ResourceName: "_project_integration_jenkins",
		ResourceDescription: `The ` + "`" + `gitlab_project_integration_jenkins` + "`" + ` resource manages the lifecycle of a project integration with Jenkins.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#jenkins)`,
	}
}

// Remove in 19.0
func NewGitLabIntegrationJenkinsResource() resource.Resource {
	return &gitlabProjectIntegrationJenkinsResource{
		ResourceName: "_integration_jenkins",
		ResourceDescription: `The ` + "`" + `gitlab_integration_jenkins` + "`" + ` resource manages the lifecycle of a project integration with Jenkins.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_integration_jenkins` + "`" + `instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#jenkins)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_integration_jenkins` instead.",
	}
}

type gitlabProjectIntegrationJenkinsResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_integration_jenkins`
	// and `gitlab_integration_jenkins` for backwards compatibility reasons. Should be removed in 19.0.
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

type gitlabProjectIntegrationJenkinsResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Project               types.String `tfsdk:"project"`
	JenkinsURL            types.String `tfsdk:"jenkins_url"`
	EnableSSLVerification types.Bool   `tfsdk:"enable_ssl_verification"`
	ProjectName           types.String `tfsdk:"project_name"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	PushEvents            types.Bool   `tfsdk:"push_events"`
	MergeRequestEvents    types.Bool   `tfsdk:"merge_request_events"`
	TagPushEvents         types.Bool   `tfsdk:"tag_push_events"`
	Active                types.Bool   `tfsdk:"active"`
}

func (r *gitlabProjectIntegrationJenkinsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectIntegrationJenkinsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"enable_ssl_verification": schema.BoolAttribute{
				MarkdownDescription: "Enable SSL verification. Defaults to `true` (enabled).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"jenkins_url": schema.StringAttribute{
				MarkdownDescription: "Jenkins URL like `http://jenkins.example.com`",
				Required:            true,
				Validators:          []validator.String{utils.HttpUrlValidator},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The URL-friendly project name. Example: `my_project_name`.",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for authentication with the Jenkins server, if authentication is required by the server.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for authentication with the Jenkins server, if authentication is required by the server.",
				Optional:            true,
				Sensitive:           true,
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push events.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"merge_request_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge request events.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for tag push events.",
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

func (r *gitlabProjectIntegrationJenkinsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectIntegrationJenkinsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Jenkins integration", err.Error())
	}
}

func (r *gitlabProjectIntegrationJenkinsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabProjectIntegrationJenkinsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	jenkins, _, err := r.client.Services.GetJenkinsCIService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read gitlab jenkins integration: %s", err.Error()))
		return
	}
	data.modelToStateModel(jenkins, projectID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectIntegrationJenkinsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Jenkins integration", err.Error())
	}
}

func (r *gitlabProjectIntegrationJenkinsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabProjectIntegrationJenkinsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Services.DeleteJenkinsCIService(data.Project.ValueString(), gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete Jenkins integration: %s", err.Error()),
		)
		return
	}
}

func (r *gitlabProjectIntegrationJenkinsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectIntegrationJenkinsResource) update(ctx context.Context, plan *tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) error {
	var data gitlabProjectIntegrationJenkinsResourceModel
	diags.Append(plan.Get(ctx, &data)...)
	if diags.HasError() {
		return nil
	}

	options := &gitlab.SetJenkinsCIServiceOptions{
		URL:                   data.JenkinsURL.ValueStringPointer(),
		EnableSSLVerification: data.EnableSSLVerification.ValueBoolPointer(),
		ProjectName:           data.ProjectName.ValueStringPointer(),
		Username:              data.Username.ValueStringPointer(),
		Password:              data.Password.ValueStringPointer(),
		PushEvents:            data.PushEvents.ValueBoolPointer(),
		MergeRequestsEvents:   data.MergeRequestEvents.ValueBoolPointer(),
		TagPushEvents:         data.TagPushEvents.ValueBoolPointer(),
	}

	tflog.Debug(ctx, "Update Gitlab Jenkins integration", map[string]any{
		"options": options,
	})

	projectID := data.Project.ValueString()

	_, _, err := r.client.Services.SetJenkinsCIService(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		return err
	}

	jenkins, _, err := r.client.Services.GetJenkinsCIService(projectID, gitlab.WithContext(ctx))
	if err != nil {
		return err
	}
	data.modelToStateModel(jenkins, projectID)

	diags.Append(state.Set(ctx, &data)...)

	return nil
}

func (d *gitlabProjectIntegrationJenkinsResourceModel) modelToStateModel(r *gitlab.JenkinsCIService, projectID string) {
	d.ID = types.StringValue(projectID)
	d.Project = types.StringValue(projectID)
	d.JenkinsURL = types.StringValue(r.Properties.URL)
	d.ProjectName = types.StringValue(r.Properties.ProjectName)
	if r.Properties.Username == "" {
		d.Username = types.StringNull()
	} else {
		d.Username = types.StringValue(r.Properties.Username)
	}
	d.EnableSSLVerification = types.BoolValue(r.Properties.EnableSSLVerification)
	d.MergeRequestEvents = types.BoolValue(r.MergeRequestsEvents)
	d.PushEvents = types.BoolValue(r.PushEvents)
	d.TagPushEvents = types.BoolValue(r.TagPushEvents)
	d.Active = types.BoolValue(r.Active)
}

// MoveState implements the ResourceWithMoveState interface to support moving state from the deprecated gitlab_integration_jenkins resource.
// This enables users to migrate from gitlab_integration_jenkins to gitlab_project_integration_jenkins using Terraform's moved block.
// Note: Cross-resource-type state moves require Terraform 1.8 or later.
func (r *gitlabProjectIntegrationJenkinsResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		// This first StateMover implements the migration from
		// `gitlab_integration_jenkins` -> `gitlab_project_integration_jenkins`.
		// The SourceSchema needs to match the deprecated `gitlab_integration_jenkins` as a result.
		{
			SourceSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"project": schema.StringAttribute{
						Required: true,
					},
					"enable_ssl_verification": schema.BoolAttribute{
						Computed: true,
					},
					"jenkins_url": schema.StringAttribute{
						Required: true,
					},
					"project_name": schema.StringAttribute{
						Required: true,
					},
					"username": schema.StringAttribute{
						Optional: true,
					},
					"password": schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
					},
					"push_events": schema.BoolAttribute{
						Computed: true,
					},
					"merge_request_events": schema.BoolAttribute{
						Computed: true,
					},
					"tag_push_events": schema.BoolAttribute{
						Computed: true,
					},
					"active": schema.BoolAttribute{
						Computed: true,
					},
				},
			},
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Only handle moves from gitlab_integration_jenkins resource
				if req.SourceTypeName != "gitlab_integration_jenkins" {
					resp.Diagnostics.AddError("Invalid source resource type", fmt.Sprintf("Expected source type 'gitlab_integration_jenkins', got '%s'", req.SourceTypeName))
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

				// Define the source model matching the old gitlab_integration_jenkins schema
				type sourceModel struct {
					ID                    types.String `tfsdk:"id"`
					Project               types.String `tfsdk:"project"`
					EnableSSLVerification types.Bool   `tfsdk:"enable_ssl_verification"`
					JenkinsURL            types.String `tfsdk:"jenkins_url"`
					ProjectName           types.String `tfsdk:"project_name"`
					Username              types.String `tfsdk:"username"`
					Password              types.String `tfsdk:"password"`
					PushEvents            types.Bool   `tfsdk:"push_events"`
					MergeRequestEvents    types.Bool   `tfsdk:"merge_request_events"`
					TagPushEvents         types.Bool   `tfsdk:"tag_push_events"`
					Active                types.Bool   `tfsdk:"active"`
				}

				var sourceStateData sourceModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				project := sourceStateData.ID.ValueString()

				// Create the target state data
				targetStateData := gitlabProjectIntegrationJenkinsResourceModel{
					ID:                    types.StringValue(project),
					Project:               sourceStateData.Project,
					EnableSSLVerification: sourceStateData.EnableSSLVerification,
					JenkinsURL:            sourceStateData.JenkinsURL,
					ProjectName:           sourceStateData.ProjectName,
					Username:              sourceStateData.Username,
					Password:              sourceStateData.Password,
					PushEvents:            sourceStateData.PushEvents,
					MergeRequestEvents:    sourceStateData.MergeRequestEvents,
					TagPushEvents:         sourceStateData.TagPushEvents,
					Active:                sourceStateData.Active,
				}

				tflog.Debug(ctx, "Moving state from gitlab_integration_jenkins to gitlab_project_integration_jenkins")
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}
