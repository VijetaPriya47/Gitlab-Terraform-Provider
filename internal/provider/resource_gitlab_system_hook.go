package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

var (
	_ resource.Resource                = &gitlabSystemHookResource{}
	_ resource.ResourceWithConfigure   = &gitlabSystemHookResource{}
	_ resource.ResourceWithImportState = &gitlabSystemHookResource{}
)

func init() {
	registerResource(NewGitlabSystemHookResource)
}

func NewGitlabSystemHookResource() resource.Resource {
	return &gitlabSystemHookResource{}
}

func (r *gitlabSystemHookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_hook"
}

type gitlabSystemHookResource struct {
	client *gitlab.Client
}

type gitlabSystemHookResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	URL                    types.String `tfsdk:"url"`
	Token                  types.String `tfsdk:"token"`
	PushEvents             types.Bool   `tfsdk:"push_events"`
	TagPushEvents          types.Bool   `tfsdk:"tag_push_events"`
	MergeRequestsEvents    types.Bool   `tfsdk:"merge_requests_events"`
	RepositoryUpdateEvents types.Bool   `tfsdk:"repository_update_events"`
	EnableSSLVerification  types.Bool   `tfsdk:"enable_ssl_verification"`
	CreatedAt              types.String `tfsdk:"created_at"`
}

func (r *gitlabSystemHookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_system_hook`" + ` resource allows to manage the lifecycle of a system hook.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/system_hooks/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this terraform resource. In the format `<hook-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The hook URL.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "Secret token to validate received payloads; this isn't returned in the response. This attribute is not available for imported resources.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "When true, the hook fires on push events.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "When true, the hook fires on new tags being pushed.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Trigger hook on merge requests events.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"repository_update_events": schema.BoolAttribute{
				MarkdownDescription: "Trigger hook on repository update events.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"enable_ssl_verification": schema.BoolAttribute{
				MarkdownDescription: "Do SSL verification when triggering the hook.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The date and time the hook was created in ISO8601 format.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabSystemHookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabSystemHookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabSystemHookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.AddHookOptions{
		URL: data.URL.ValueStringPointer(),
	}

	if !data.Token.IsNull() && !data.Token.IsUnknown() {
		options.Token = data.Token.ValueStringPointer()
	}

	if !data.PushEvents.IsNull() && !data.PushEvents.IsUnknown() {
		options.PushEvents = data.PushEvents.ValueBoolPointer()
	}

	if !data.TagPushEvents.IsNull() && !data.TagPushEvents.IsUnknown() {
		options.TagPushEvents = data.TagPushEvents.ValueBoolPointer()
	}

	if !data.MergeRequestsEvents.IsNull() && !data.MergeRequestsEvents.IsUnknown() {
		options.MergeRequestsEvents = data.MergeRequestsEvents.ValueBoolPointer()
	}

	if !data.RepositoryUpdateEvents.IsNull() && !data.RepositoryUpdateEvents.IsUnknown() {
		options.RepositoryUpdateEvents = data.RepositoryUpdateEvents.ValueBoolPointer()
	}

	if !data.EnableSSLVerification.IsNull() && !data.EnableSSLVerification.IsUnknown() {
		options.EnableSSLVerification = data.EnableSSLVerification.ValueBoolPointer()
	}

	tflog.Debug(ctx, fmt.Sprintf("create gitlab system hook %q", *options.URL))

	hook, _, err := r.client.SystemHooks.AddHook(options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create system hook: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", hook.ID))
	data.Token = types.StringValue(*options.Token)
	data.systemHookToStateModel(hook)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabSystemHookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabSystemHookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hookID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<hook-id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read gitlab system hook %d", hookID))

	hook, _, err := r.client.SystemHooks.GetHook(hookID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("gitlab system hook not found %d, removing from state", hookID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read system hook: %s", err.Error()))
		return
	}
	data.systemHookToStateModel(hook)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabSystemHookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

func (r *gitlabSystemHookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabSystemHookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hookID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<hook-id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Delete gitlab system hook %d", hookID))

	_, err = r.client.SystemHooks.DeleteHook(hookID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete gitlab system hook", err.Error())
		return
	}
}

func (r *gitlabSystemHookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabSystemHookResourceModel) systemHookToStateModel(hook *gitlab.Hook) {
	d.URL = types.StringValue(hook.URL)
	d.PushEvents = types.BoolValue(hook.PushEvents)
	d.TagPushEvents = types.BoolValue(hook.TagPushEvents)
	d.MergeRequestsEvents = types.BoolValue(hook.MergeRequestsEvents)
	d.RepositoryUpdateEvents = types.BoolValue(hook.RepositoryUpdateEvents)
	d.EnableSSLVerification = types.BoolValue(hook.EnableSSLVerification)
	d.CreatedAt = types.StringValue(hook.CreatedAt.Format(time.RFC3339))
}
