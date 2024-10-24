package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                 = &gitlabProjectHookResource{}
	_ resource.ResourceWithConfigure    = &gitlabProjectHookResource{}
	_ resource.ResourceWithImportState  = &gitlabProjectHookResource{}
	_ resource.ResourceWithUpgradeState = &gitlabProjectHookResource{}
)

func init() {
	registerResource(NewGitLabProjectHookResource)
}

// NewGitLabProjectHookResource is a helper function to simplify the provider implementation.
func NewGitLabProjectHookResource() resource.Resource {
	return &gitlabProjectHookResource{}
}

type gitlabProjectHookResourceModel struct {
	ID types.String `tfsdk:"id"`

	Project   types.String `tfsdk:"project"`
	ProjectID types.Int64  `tfsdk:"project_id"`
	HookID    types.Int64  `tfsdk:"hook_id"`
	URL       types.String `tfsdk:"url"`
	Token     types.String `tfsdk:"token"`

	PushEvents               types.Bool   `tfsdk:"push_events"`
	PushEventsBranchFilter   types.String `tfsdk:"push_events_branch_filter"`
	IssuesEvents             types.Bool   `tfsdk:"issues_events"`
	ConfidentialIssuesEvents types.Bool   `tfsdk:"confidential_issues_events"`
	MergeRequestsEvents      types.Bool   `tfsdk:"merge_requests_events"`
	TagPushEvents            types.Bool   `tfsdk:"tag_push_events"`
	NoteEvents               types.Bool   `tfsdk:"note_events"`
	ConfidentialNoteEvents   types.Bool   `tfsdk:"confidential_note_events"`
	JobEvents                types.Bool   `tfsdk:"job_events"`
	PipelineEvents           types.Bool   `tfsdk:"pipeline_events"`
	WikiPageEvents           types.Bool   `tfsdk:"wiki_page_events"`
	DeploymentEvents         types.Bool   `tfsdk:"deployment_events"`
	ReleasesEvents           types.Bool   `tfsdk:"releases_events"`
	EnableSSLVerification    types.Bool   `tfsdk:"enable_ssl_verification"`

	CustomWebhookTemplate types.String `tfsdk:"custom_webhook_template"`

	// Model defined in resource_gitlab_group_hook.go
	CustomHeaders []*gitlabHookCustomHeaderModel `tfsdk:"custom_headers"`
}

type gitlabProjectHookResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectHookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_hook"
}

func (r *gitlabProjectHookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getSchema()
}

func (r *gitlabProjectHookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabProjectHookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectHookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectHookResourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.AddProjectHookOptions{
		URL:                      data.URL.ValueStringPointer(),
		PushEvents:               data.PushEvents.ValueBoolPointer(),
		PushEventsBranchFilter:   data.PushEventsBranchFilter.ValueStringPointer(),
		IssuesEvents:             data.IssuesEvents.ValueBoolPointer(),
		ConfidentialIssuesEvents: data.ConfidentialIssuesEvents.ValueBoolPointer(),
		MergeRequestsEvents:      data.MergeRequestsEvents.ValueBoolPointer(),
		TagPushEvents:            data.TagPushEvents.ValueBoolPointer(),
		NoteEvents:               data.NoteEvents.ValueBoolPointer(),
		ConfidentialNoteEvents:   data.ConfidentialNoteEvents.ValueBoolPointer(),
		JobEvents:                data.JobEvents.ValueBoolPointer(),
		PipelineEvents:           data.PipelineEvents.ValueBoolPointer(),
		WikiPageEvents:           data.WikiPageEvents.ValueBoolPointer(),
		DeploymentEvents:         data.DeploymentEvents.ValueBoolPointer(),
		ReleasesEvents:           data.ReleasesEvents.ValueBoolPointer(),
		EnableSSLVerification:    data.EnableSSLVerification.ValueBoolPointer(),
		CustomWebhookTemplate:    data.CustomWebhookTemplate.ValueStringPointer(),
	}

	if !data.Token.IsNull() {
		options.Token = data.Token.ValueStringPointer()
	}

	if len(data.CustomHeaders) > 0 {
		headers := []*gitlab.HookCustomHeader{}
		for _, header := range data.CustomHeaders {
			headers = append(headers, &gitlab.HookCustomHeader{
				Key:   header.Key.ValueString(),
				Value: header.Value.ValueString(),
			})
		}

		options.CustomHeaders = &headers
	}

	tflog.Debug(ctx, "creating gitlab project hook with details", map[string]interface{}{
		"project": data.Project,
		"url":     data.URL.ValueString(),
	})

	hook, _, err := r.client.Projects.AddProjectHook(data.Project.ValueString(), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitLab project hook", err.Error())
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(data.Project.ValueStringPointer(), gitlab.Ptr(strconv.Itoa(hook.ID))))
	data.modelToStateModel(hook)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *gitlabProjectHookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectHookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, hookId, err := data.ResourceGitlabProjectHookParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab project hook", err.Error())
		return
	}

	tflog.Debug(ctx, "reading gitlab project hook with details", map[string]interface{}{
		"project": project,
		"id":      hookId,
	})

	hook, _, err := r.client.Projects.GetProjectHook(project, hookId, gitlab.WithContext(ctx))
	if err != nil {
		// Project/Hook not found
		if api.Is404(err) {
			tflog.Debug(ctx, "gitlab project hook not found, removing from state", map[string]interface{}{
				"project": project,
				"id":      hookId,
			})
			resp.State.RemoveResource(ctx)
		} else {
			// It's a real error
			resp.Diagnostics.AddError("Error reading GitLab project hook", err.Error())
		}
		return
	}

	data.Project = types.StringValue(project)
	data.modelToStateModel(hook)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectHookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectHookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, hookId, err := data.ResourceGitlabProjectHookParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab project hook", err.Error())
		return
	}

	options := &gitlab.EditProjectHookOptions{
		URL:                      data.URL.ValueStringPointer(),
		PushEvents:               data.PushEvents.ValueBoolPointer(),
		PushEventsBranchFilter:   data.PushEventsBranchFilter.ValueStringPointer(),
		IssuesEvents:             data.IssuesEvents.ValueBoolPointer(),
		ConfidentialIssuesEvents: data.ConfidentialIssuesEvents.ValueBoolPointer(),
		MergeRequestsEvents:      data.MergeRequestsEvents.ValueBoolPointer(),
		TagPushEvents:            data.TagPushEvents.ValueBoolPointer(),
		NoteEvents:               data.NoteEvents.ValueBoolPointer(),
		ConfidentialNoteEvents:   data.ConfidentialNoteEvents.ValueBoolPointer(),
		JobEvents:                data.JobEvents.ValueBoolPointer(),
		PipelineEvents:           data.PipelineEvents.ValueBoolPointer(),
		WikiPageEvents:           data.WikiPageEvents.ValueBoolPointer(),
		DeploymentEvents:         data.DeploymentEvents.ValueBoolPointer(),
		ReleasesEvents:           data.ReleasesEvents.ValueBoolPointer(),
		EnableSSLVerification:    data.EnableSSLVerification.ValueBoolPointer(),
		CustomWebhookTemplate:    data.CustomWebhookTemplate.ValueStringPointer(),
	}

	if !data.Token.IsNull() {
		options.Token = data.Token.ValueStringPointer()
	}

	if len(data.CustomHeaders) > 0 {
		headers := []*gitlab.HookCustomHeader{}
		for _, header := range data.CustomHeaders {
			headers = append(headers, &gitlab.HookCustomHeader{
				Key:   header.Key.ValueString(),
				Value: header.Value.ValueString(),
			})
		}

		options.CustomHeaders = &headers
	}

	tflog.Debug(ctx, "updating gitlab project hook with details", map[string]interface{}{
		"project": data.Project,
		"url":     data.URL.ValueString(),
	})

	hook, _, err := r.client.Projects.EditProjectHook(project, hookId, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitLab project hook", err.Error())
		return
	}

	data.modelToStateModel(hook)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectHookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectHookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, hookId, err := data.ResourceGitlabProjectHookParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab project hook", err.Error())
		return
	}

	_, err = r.client.Projects.DeleteProjectHook(project, hookId, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting GitLab project hook", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (d *gitlabProjectHookResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: gitlab.Ptr(d.getSchema()),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data *gitlabProjectHookResourceModel
				// Read Terraform plan data into the model
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				data.v0StateUpgrade(ctx)

				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]interface{}{"v1-id": data.ID.ValueString()})
			},
		},
	}
}

// Retrieve the attributes for the schema. Separated out from the rest of the schema
// so that the migration can refer to it more easily
func (d *gitlabProjectHookResource) getSchema() schema.Schema {
	return schema.Schema{
		Version: 1,
		MarkdownDescription: `The ` + "`" + `gitlab_project_hook` + "`" + ` resource allows to manage the lifecycle of a project hook.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/projects.html#hooks)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The id of the project hook. In the format of "project:hook_id"`,
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the hook to.",
				Required:            true,
			},
			"project_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the project for the hook.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"hook_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the project hook.",
				Computed:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The url of the hook to invoke. Forces re-creation to preserve `token`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "A token to present when invoking the hook. The token is not available for imported resources.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"push_events": schema.BoolAttribute{
				Description: "Invoke the hook for push events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"push_events_branch_filter": schema.StringAttribute{
				Description: "Invoke the hook for push events on matching branches only.",
				Optional:    true,
				Computed:    true,
			},
			"issues_events": schema.BoolAttribute{
				Description: "Invoke the hook for issues events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"confidential_issues_events": schema.BoolAttribute{
				Description: "Invoke the hook for confidential issues events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"merge_requests_events": schema.BoolAttribute{
				Description: "Invoke the hook for merge requests events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"tag_push_events": schema.BoolAttribute{
				Description: "Invoke the hook for tag push events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"note_events": schema.BoolAttribute{
				Description: "Invoke the hook for note events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"confidential_note_events": schema.BoolAttribute{
				Description: "Invoke the hook for confidential note events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"job_events": schema.BoolAttribute{
				Description: "Invoke the hook for job events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"pipeline_events": schema.BoolAttribute{
				Description: "Invoke the hook for pipeline events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"wiki_page_events": schema.BoolAttribute{
				Description: "Invoke the hook for wiki page events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"deployment_events": schema.BoolAttribute{
				Description: "Invoke the hook for deployment events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"releases_events": schema.BoolAttribute{
				Description: "Invoke the hook for release events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enable_ssl_verification": schema.BoolAttribute{
				Description: "Enable SSL verification when invoking the hook.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"custom_webhook_template": schema.StringAttribute{
				Description: "Custom webhook template.",
				Optional:    true,
				Computed:    true,
			},
			"custom_headers": schema.ListNestedAttribute{
				Description: "Custom headers for the project webhook.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "Key of the custom header.",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Required:      true,
							Description:   "Value of the custom header. This value cannot be imported.",
							Sensitive:     true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
					},
				},
			},
		},
	}
}

// Save the model data from a gitlab.ProjectHook object
func (d *gitlabProjectHookResourceModel) modelToStateModel(a *gitlab.ProjectHook) {
	d.URL = types.StringValue(a.URL)
	d.ProjectID = types.Int64Value(int64(a.ProjectID))
	d.HookID = types.Int64Value(int64(a.ID))

	d.PushEvents = types.BoolValue(a.PushEvents)
	d.PushEventsBranchFilter = types.StringValue(a.PushEventsBranchFilter)

	d.IssuesEvents = types.BoolValue(a.IssuesEvents)
	d.ConfidentialIssuesEvents = types.BoolValue(a.ConfidentialIssuesEvents)
	d.MergeRequestsEvents = types.BoolValue(a.MergeRequestsEvents)
	d.TagPushEvents = types.BoolValue(a.TagPushEvents)
	d.NoteEvents = types.BoolValue(a.NoteEvents)
	d.ConfidentialNoteEvents = types.BoolValue(a.ConfidentialNoteEvents)
	d.JobEvents = types.BoolValue(a.JobEvents)
	d.PipelineEvents = types.BoolValue(a.PipelineEvents)
	d.WikiPageEvents = types.BoolValue(a.WikiPageEvents)
	d.DeploymentEvents = types.BoolValue(a.DeploymentEvents)
	d.ReleasesEvents = types.BoolValue(a.ReleasesEvents)
	d.EnableSSLVerification = types.BoolValue(a.EnableSSLVerification)
	d.CustomWebhookTemplate = types.StringValue(a.CustomWebhookTemplate)

	// create a map of key/value data from state currently, so we don't overwrite
	// values in state when we can't read the values
	currentHeaderValues := map[string]string{}
	for _, v := range d.CustomHeaders {
		currentHeaderValues[v.Key.ValueString()] = v.Value.ValueString()
	}

	// Iterate through the headers that came back on the hook object, and
	// add them to state using the value that already exists in state previously.
	// Without this logic, the value would be lost in state with every plan/apply
	headers := []*gitlabHookCustomHeaderModel{}
	for _, v := range a.CustomHeaders {
		head := &gitlabHookCustomHeaderModel{}
		head.Key = types.StringValue(v.Key)

		// Value doesn't come back on read requests, so if it's "", we grab the value from
		// the current data state instead of the hook, so we don't "lose" the value.
		if v.Value != "" {
			head.Value = types.StringValue(v.Value)
		} else {
			head.Value = types.StringValue(currentHeaderValues[v.Key])
		}
		// Add the header to the list
		headers = append(headers, head)
	}

	// Only set the headers to state if there is at least 1 present. Otherwise,
	// the resource would break if there were no custom headers set.
	if len(headers) > 0 {
		d.CustomHeaders = headers
	}
}

// Updates the ID from just `hook_id` to `project:hook_id`
// Separated out so it can be tested individually
func (d *gitlabProjectHookResourceModel) v0StateUpgrade(ctx context.Context) {
	// The old ID was just the hook ID, and didn't contain the project
	oldIdValue := d.ID.ValueStringPointer()
	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]interface{}{"project": d.Project.ValueString(), "v0-id": oldIdValue})

	// Update the ID format and save that back into `data` as the ID
	d.ID = types.StringValue(utils.BuildTwoPartID(d.Project.ValueStringPointer(), oldIdValue))
}

func (d *gitlabProjectHookResourceModel) ResourceGitlabProjectHookParseId(id string) (string, int, error) {
	project, rawHookId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	hookId, err := strconv.Atoi(rawHookId)
	if err != nil {
		return "", 0, err
	}

	return project, hookId, nil
}
