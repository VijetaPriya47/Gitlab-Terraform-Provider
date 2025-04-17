package provider

import (
	"context"
	"fmt"

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
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabIntegrationJenkinsResource{}
	_ resource.ResourceWithConfigure   = &gitlabIntegrationJenkinsResource{}
	_ resource.ResourceWithImportState = &gitlabIntegrationJenkinsResource{}
)

func init() {
	registerResource(NewGitLabIntegrationJenkinsResource)
}

func NewGitLabIntegrationJenkinsResource() resource.Resource {
	return &gitlabIntegrationJenkinsResource{}
}

type gitlabIntegrationJenkinsResource struct {
	client *gitlab.Client
}

type gitlabIntegrationJenkinsResourceModel struct {
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

func (r *gitlabIntegrationJenkinsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_jenkins"
}

func (r *gitlabIntegrationJenkinsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_integration_jenkins`" + ` resource allows to manage the lifecycle of a project integration with Jenkins.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_integrations/#jenkins)`,
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

func (r *gitlabIntegrationJenkinsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabIntegrationJenkinsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Jenkins integration", err.Error())
	}
}

func (r *gitlabIntegrationJenkinsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabIntegrationJenkinsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ID.ValueString()

	jenkins, _, err := r.client.Services.GetJenkinsCIService(projectID)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read gitlab jenkins integration: %s", err.Error()))
		return
	}
	data.modelToStateModel(jenkins, projectID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabIntegrationJenkinsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	err := r.update(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Jenkins integration", err.Error())
	}
}

func (r *gitlabIntegrationJenkinsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabIntegrationJenkinsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Services.DeleteJenkinsCIService(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete Jenkins integration: %s", err.Error()),
		)
		return
	}
}

func (r *gitlabIntegrationJenkinsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabIntegrationJenkinsResource) update(ctx context.Context, plan *tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) error {
	var data gitlabIntegrationJenkinsResourceModel
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

	tflog.Debug(ctx, "Update Gitlab Jenkins integration", map[string]interface{}{
		"options": options,
	})

	projectID := data.Project.ValueString()

	_, _, err := r.client.Services.SetJenkinsCIService(projectID, options)
	if err != nil {
		return err
	}

	jenkins, _, err := r.client.Services.GetJenkinsCIService(projectID)
	if err != nil {
		return err
	}
	data.modelToStateModel(jenkins, projectID)

	diags.Append(state.Set(ctx, &data)...)

	return nil
}

func (d *gitlabIntegrationJenkinsResourceModel) modelToStateModel(r *gitlab.JenkinsCIService, projectID string) {
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
