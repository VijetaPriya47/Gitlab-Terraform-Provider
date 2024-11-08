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
	_ resource.Resource                = &gitlabGroupHookResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupHookResource{}
	_ resource.ResourceWithImportState = &gitlabGroupHookResource{}
)

func init() {
	registerResource(NewGitLabGroupHookResource)
}

// NewGitLabProjectHookResource is a helper function to simplify the provider implementation.
func NewGitLabGroupHookResource() resource.Resource {
	return &gitlabGroupHookResource{}
}

type gitlabGroupHookResourceModel struct {
	ID types.String `tfsdk:"id"`

	Group   types.String `tfsdk:"group"`
	GroupID types.Int64  `tfsdk:"group_id"`
	HookID  types.Int64  `tfsdk:"hook_id"`
	URL     types.String `tfsdk:"url"`
	Token   types.String `tfsdk:"token"`

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
	SubGroupEvents           types.Bool   `tfsdk:"subgroup_events"`

	EnableSSLVerification types.Bool                     `tfsdk:"enable_ssl_verification"`
	CustomWebhookTemplate types.String                   `tfsdk:"custom_webhook_template"`
	CustomHeaders         []*gitlabHookCustomHeaderModel `tfsdk:"custom_headers"`
}

type gitlabHookCustomHeaderModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type gitlabGroupHookResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupHookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_hook"
}

func (r *gitlabGroupHookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getSchema()
}

func (r *gitlabGroupHookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupHookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupHookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupHookResourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.AddGroupHookOptions{
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
		SubGroupEvents:           data.SubGroupEvents.ValueBoolPointer(),
		EnableSSLVerification:    data.EnableSSLVerification.ValueBoolPointer(),
		CustomWebhookTemplate:    data.CustomWebhookTemplate.ValueStringPointer(),
	}

	if !data.Token.IsNull() {
		options.Token = data.Token.ValueStringPointer()
	}

	if len(data.CustomHeaders) > 0 {
		headers := make([]*gitlab.HookCustomHeader, 0, len(data.CustomHeaders))
		for _, header := range data.CustomHeaders {
			headers = append(headers, &gitlab.HookCustomHeader{
				Key:   header.Key.ValueString(),
				Value: header.Value.ValueString(),
			})
		}

		options.CustomHeaders = &headers
	}

	tflog.Debug(ctx, "creating gitlab group hook with details", map[string]interface{}{
		"group": data.Group,
		"url":   data.URL.ValueString(),
	})

	hook, _, err := r.client.Groups.AddGroupHook(data.Group.ValueString(), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitLab group hook", err.Error())
		return
	}

	data.ID = types.StringValue(utils.BuildTwoPartID(data.Group.ValueStringPointer(), gitlab.Ptr(strconv.Itoa(hook.ID))))
	data.modelToStateModel(hook)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *gitlabGroupHookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupHookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, hookId, err := data.ResourceGitlabGroupHookParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab Group hook", err.Error())
		return
	}

	tflog.Debug(ctx, "reading gitlab Group hook with details", map[string]interface{}{
		"group": group,
		"id":    hookId,
	})

	hook, _, err := r.client.Groups.GetGroupHook(group, hookId, gitlab.WithContext(ctx))
	if err != nil {
		// Group/Hook not found
		if api.Is404(err) {
			tflog.Debug(ctx, "gitlab Group hook not found, removing from state", map[string]interface{}{
				"group": group,
				"id":    hookId,
			})
			resp.State.RemoveResource(ctx)
		} else {
			// It's a real error
			resp.Diagnostics.AddError("Error reading GitLab Group hook", err.Error())
		}
		return
	}

	data.Group = types.StringValue(group)
	data.modelToStateModel(hook)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupHookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupHookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, hookId, err := data.ResourceGitlabGroupHookParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab project hook", err.Error())
		return
	}

	options := &gitlab.EditGroupHookOptions{
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
		SubGroupEvents:           data.SubGroupEvents.ValueBoolPointer(),
		EnableSSLVerification:    data.EnableSSLVerification.ValueBoolPointer(),
		CustomWebhookTemplate:    data.CustomWebhookTemplate.ValueStringPointer(),
	}

	if !data.Token.IsNull() {
		options.Token = data.Token.ValueStringPointer()
	}

	if len(data.CustomHeaders) > 0 {
		headers := make([]*gitlab.HookCustomHeader, 0, len(data.CustomHeaders))
		for _, header := range data.CustomHeaders {
			headers = append(headers, &gitlab.HookCustomHeader{
				Key:   header.Key.ValueString(),
				Value: header.Value.ValueString(),
			})
		}

		options.CustomHeaders = &headers
	}

	tflog.Debug(ctx, "updating gitlab Group hook with details", map[string]interface{}{
		"group": data.Group,
		"url":   data.URL.ValueString(),
	})

	hook, _, err := r.client.Groups.EditGroupHook(project, hookId, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitLab Group hook", err.Error())
		return
	}

	data.modelToStateModel(hook)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupHookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupHookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, hookId, err := data.ResourceGitlabGroupHookParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab group hook", err.Error())
		return
	}

	_, err = r.client.Groups.DeleteGroupHook(group, hookId, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting GitLab group hook", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

// Retrieve the attributes for the schema. Separated out from the rest of the schema
// so that the migration can refer to it more easily
func (d *gitlabGroupHookResource) getSchema() schema.Schema {
	return schema.Schema{
		Version: 0,
		MarkdownDescription: `The ` + "`" + `gitlab_group_hook` + "`" + ` resource allows to manage the lifecycle of a group hook.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/groups.html#hooks)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The id of the group hook. In the format of "group:hook_id"`,
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The full path or id of the group to add the hook to.",
				Required:            true,
			},
			"group_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the group for the hook.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"hook_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the group hook.",
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
			"subgroup_events": schema.BoolAttribute{
				Description: "Invoke the hook for subgroup events.",
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

// Save the model data from a gitlab.GroupHook object
func (d *gitlabGroupHookResourceModel) modelToStateModel(a *gitlab.GroupHook) {
	d.URL = types.StringValue(a.URL)
	d.GroupID = types.Int64Value(int64(a.GroupID))
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
	d.SubGroupEvents = types.BoolValue(a.SubGroupEvents)
	d.EnableSSLVerification = types.BoolValue(a.EnableSSLVerification)
	d.CustomWebhookTemplate = types.StringValue(a.CustomWebhookTemplate)

	if len(a.CustomHeaders) > 0 || len(d.CustomHeaders) > 0 {
		// create a map of key/value data from state currently, so we don't overwrite
		// values in state when we can't read the values
		currentHeaderValues := map[string]string{}
		for _, v := range d.CustomHeaders {
			currentHeaderValues[v.Key.ValueString()] = v.Value.ValueString()
		}

		// Iterate through the headers that came back on the hook object, and
		// add them to state using the value that already exists in state previously.
		// Without this logic, the value would be lost in state with every plan/apply
		headers := make([]*gitlabHookCustomHeaderModel, 0, len(a.CustomHeaders))
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
			headers = append(headers, head)
		}

		d.CustomHeaders = headers
	}

}

// Not bound to the resource model because it's used in the tests, so this
// makes accessing it easier
func (d *gitlabGroupHookResourceModel) ResourceGitlabGroupHookParseID(id string) (string, int, error) {
	group, rawHookId, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	hookId, err := strconv.Atoi(rawHookId)
	if err != nil {
		return "", 0, err
	}

	return group, hookId, nil
}
