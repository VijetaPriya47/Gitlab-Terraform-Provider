package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// groupMattermostIntegrationResponse is a custom struct that correctly matches
// the API response structure. The client-go GroupMattermostIntegration struct
// has notify_only_broken_pipelines, branches_to_be_notified, labels_to_be_notified,
// and labels_to_be_notified_behavior at the top level, but the API returns them
// inside the properties object.
// TODO: Remove this when client-go is fixed to match the actual API response.
type groupMattermostIntegrationResponse struct {
	ID                       int64                                         `json:"id"`
	Active                   bool                                          `json:"active"`
	PushEvents               bool                                          `json:"push_events"`
	IssuesEvents             bool                                          `json:"issues_events"`
	ConfidentialIssuesEvents bool                                          `json:"confidential_issues_events"`
	MergeRequestsEvents      bool                                          `json:"merge_requests_events"`
	TagPushEvents            bool                                          `json:"tag_push_events"`
	NoteEvents               bool                                          `json:"note_events"`
	ConfidentialNoteEvents   bool                                          `json:"confidential_note_events"`
	PipelineEvents           bool                                          `json:"pipeline_events"`
	WikiPageEvents           bool                                          `json:"wiki_page_events"`
	Properties               *groupMattermostIntegrationPropertiesResponse `json:"properties"`
}

type groupMattermostIntegrationPropertiesResponse struct {
	Username                   string `json:"username"`
	Channel                    string `json:"channel"`
	NotifyOnlyBrokenPipelines  bool   `json:"notify_only_broken_pipelines"`
	BranchesToBeNotified       string `json:"branches_to_be_notified"`
	LabelsToBeNotified         string `json:"labels_to_be_notified"`
	LabelsToBeNotifiedBehavior string `json:"labels_to_be_notified_behavior"`
	PushChannel                string `json:"push_channel"`
	IssueChannel               string `json:"issue_channel"`
	ConfidentialIssueChannel   string `json:"confidential_issue_channel"`
	MergeRequestChannel        string `json:"merge_request_channel"`
	NoteChannel                string `json:"note_channel"`
	ConfidentialNoteChannel    string `json:"confidential_note_channel"`
	TagPushChannel             string `json:"tag_push_channel"`
	PipelineChannel            string `json:"pipeline_channel"`
	WikiPageChannel            string `json:"wiki_page_channel"`
}

var (
	_ resource.Resource                = &gitlabGroupIntegrationMattermostResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupIntegrationMattermostResource{}
	_ resource.ResourceWithImportState = &gitlabGroupIntegrationMattermostResource{}
)

func init() {
	registerResource(NewGitlabGroupIntegrationMattermostResource)
}

func NewGitlabGroupIntegrationMattermostResource() resource.Resource {
	return &gitlabGroupIntegrationMattermostResource{}
}

type gitlabGroupIntegrationMattermostResource struct {
	client *gitlab.Client
}

func (r *gitlabGroupIntegrationMattermostResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_integration_mattermost"
}

func (r *gitlabGroupIntegrationMattermostResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_integration_mattermost`" + ` resource manages the lifecycle of a group integration with Mattermost.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_integrations/#mattermost-notifications)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the group.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the group to integrate with Mattermost.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"webhook": schema.StringAttribute{
				MarkdownDescription: "Mattermost notifications webhook (for example, http://mattermost.example.com/hooks/...).",
				Required:            true,
				Sensitive:           true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Mattermost notifications username.",
				Optional:            true,
				Computed:            true,
			},
			"channel": schema.StringAttribute{
				MarkdownDescription: "Default channel to use if no other channel is configured.",
				Optional:            true,
				Computed:            true,
			},
			"use_inherited_settings": schema.BoolAttribute{
				MarkdownDescription: "Inherit settings from parent group.",
				Optional:            true,
			},
			"labels_to_be_notified": schema.StringAttribute{
				MarkdownDescription: "Labels to send notifications for. Leave blank to receive notifications for all events.",
				Optional:            true,
				Computed:            true,
			},
			"labels_to_be_notified_behavior": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Labels to be notified for. Valid values are %s.", utils.RenderValueListForDocs(api.ValidLabelsToBeNotifiedBehavior)),
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidLabelsToBeNotifiedBehavior...)},
			},
			"notify_only_broken_pipelines": schema.BoolAttribute{
				MarkdownDescription: "Send notifications for broken pipelines.",
				Optional:            true,
				Computed:            true,
			},
			"branches_to_be_notified": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Branches to send notifications for. Valid values are %s.", utils.RenderValueListForDocs(api.ValidBranchesToBeNotified)),
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidBranchesToBeNotified...)},
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for push events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for issues events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"confidential_issues_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential issues events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for merge requests events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for tag push events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for note events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"confidential_note_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for confidential note events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"pipeline_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for pipeline events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"wiki_page_events": schema.BoolAttribute{
				MarkdownDescription: "Enable notifications for wiki page events.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"push_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive push events notifications.",
				Optional:            true,
				Computed:            true,
			},
			"issue_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive issue events notifications.",
				Optional:            true,
				Computed:            true,
			},
			"confidential_issue_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive confidential issue events notifications.",
				Optional:            true,
				Computed:            true,
			},
			"merge_request_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive merge request events notifications.",
				Optional:            true,
				Computed:            true,
			},
			"note_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive note events notifications.",
				Optional:            true,
				Computed:            true,
			},
			"confidential_note_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive confidential note events notifications.",
				Optional:            true,
				Computed:            true,
			},
			"tag_push_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive tag push events notifications.",
				Optional:            true,
				Computed:            true,
			},
			"pipeline_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive pipeline events notifications.",
				Optional:            true,
				Computed:            true,
			},
			"wiki_page_channel": schema.StringAttribute{
				MarkdownDescription: "The name of the channel to receive wiki page events notifications.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

type gitlabGroupIntegrationMattermostResourceModel struct {
	Id                         types.String `tfsdk:"id"`
	Group                      types.String `tfsdk:"group"`
	Webhook                    types.String `tfsdk:"webhook"`
	Username                   types.String `tfsdk:"username"`
	Channel                    types.String `tfsdk:"channel"`
	UseInheritedSettings       types.Bool   `tfsdk:"use_inherited_settings"`
	LabelsToBeNotified         types.String `tfsdk:"labels_to_be_notified"`
	LabelsToBeNotifiedBehavior types.String `tfsdk:"labels_to_be_notified_behavior"`
	NotifyOnlyBrokenPipelines  types.Bool   `tfsdk:"notify_only_broken_pipelines"`
	BranchesToBeNotified       types.String `tfsdk:"branches_to_be_notified"`
	PushEvents                 types.Bool   `tfsdk:"push_events"`
	IssuesEvents               types.Bool   `tfsdk:"issues_events"`
	ConfidentialIssuesEvents   types.Bool   `tfsdk:"confidential_issues_events"`
	MergeRequestsEvents        types.Bool   `tfsdk:"merge_requests_events"`
	TagPushEvents              types.Bool   `tfsdk:"tag_push_events"`
	NoteEvents                 types.Bool   `tfsdk:"note_events"`
	ConfidentialNoteEvents     types.Bool   `tfsdk:"confidential_note_events"`
	PipelineEvents             types.Bool   `tfsdk:"pipeline_events"`
	WikiPageEvents             types.Bool   `tfsdk:"wiki_page_events"`
	PushChannel                types.String `tfsdk:"push_channel"`
	IssueChannel               types.String `tfsdk:"issue_channel"`
	ConfidentialIssueChannel   types.String `tfsdk:"confidential_issue_channel"`
	MergeRequestChannel        types.String `tfsdk:"merge_request_channel"`
	NoteChannel                types.String `tfsdk:"note_channel"`
	ConfidentialNoteChannel    types.String `tfsdk:"confidential_note_channel"`
	TagPushChannel             types.String `tfsdk:"tag_push_channel"`
	PipelineChannel            types.String `tfsdk:"pipeline_channel"`
	WikiPageChannel            types.String `tfsdk:"wiki_page_channel"`
}

func (r *gitlabGroupIntegrationMattermostResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabGroupIntegrationMattermostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabGroupIntegrationMattermostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateMattermostIntegration(ctx, &data, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = data.Group
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupIntegrationMattermostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabGroupIntegrationMattermostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := data.Id.ValueString()

	// Use raw HTTP GET because client-go's GroupMattermostIntegration struct has
	// a bug where fields like notify_only_broken_pipelines are expected at top level,
	// but the API returns them inside the properties object.
	service, err := r.getGroupMattermostIntegration(ctx, groupID)
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "mattermost integration doesn't exist, removing from state", map[string]any{
				"group": groupID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error reading mattermost integration for group %s: %s", groupID, err.Error()))
		return
	}

	r.apiToStateModel(service, &data)

	// Set group from ID if not already set (for import)
	if data.Group.IsNull() || data.Group.ValueString() == "" {
		data.Group = types.StringValue(groupID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupIntegrationMattermostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabGroupIntegrationMattermostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateMattermostIntegration(ctx, &data, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupIntegrationMattermostResource) updateMattermostIntegration(ctx context.Context, data *gitlabGroupIntegrationMattermostResourceModel, diags diag.Diagnostics) {
	groupID := data.Group.ValueString()

	opts := r.stateModelToApiOptions(data)
	if _, _, err := r.client.Integrations.SetGroupMattermostIntegration(groupID, opts, gitlab.WithContext(ctx)); err != nil {
		diags.AddError("GitLab API error occurred", fmt.Sprintf("Error setting mattermost integration for group %s: %v", groupID, err))
		return
	}

	service, err := r.getGroupMattermostIntegration(ctx, groupID)
	if err != nil {
		diags.AddError("GitLab API error occurred", fmt.Sprintf("Error reading mattermost integration for group %s: %v", groupID, err))
		return
	}

	r.apiToStateModel(service, data)
}

func (r *gitlabGroupIntegrationMattermostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabGroupIntegrationMattermostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := data.Id.ValueString()

	_, err := r.client.Integrations.DeleteGroupMattermostIntegration(groupID, gitlab.WithContext(ctx))
	if err != nil && !api.Is404(err) {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Error deleting mattermost integration for group %s: %v", groupID, err))
	}
}

func (r *gitlabGroupIntegrationMattermostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// stateModelToApiOptions converts the Terraform state model to GitLab API options.
func (r *gitlabGroupIntegrationMattermostResource) stateModelToApiOptions(data *gitlabGroupIntegrationMattermostResourceModel) *gitlab.GroupMattermostIntegrationOptions {
	opts := &gitlab.GroupMattermostIntegrationOptions{
		WebHook:                    data.Webhook.ValueStringPointer(),
		Username:                   data.Username.ValueStringPointer(),
		UseInheritedSettings:       data.UseInheritedSettings.ValueBoolPointer(),
		LabelsToBeNotified:         data.LabelsToBeNotified.ValueStringPointer(),
		LabelsToBeNotifiedBehavior: data.LabelsToBeNotifiedBehavior.ValueStringPointer(),
		NotifyOnlyBrokenPipelines:  data.NotifyOnlyBrokenPipelines.ValueBoolPointer(),
		BranchesToBeNotified:       data.BranchesToBeNotified.ValueStringPointer(),
		PushEvents:                 data.PushEvents.ValueBoolPointer(),
		IssuesEvents:               data.IssuesEvents.ValueBoolPointer(),
		ConfidentialIssuesEvents:   data.ConfidentialIssuesEvents.ValueBoolPointer(),
		MergeRequestsEvents:        data.MergeRequestsEvents.ValueBoolPointer(),
		TagPushEvents:              data.TagPushEvents.ValueBoolPointer(),
		NoteEvents:                 data.NoteEvents.ValueBoolPointer(),
		ConfidentialNoteEvents:     data.ConfidentialNoteEvents.ValueBoolPointer(),
		PipelineEvents:             data.PipelineEvents.ValueBoolPointer(),
		WikiPageEvents:             data.WikiPageEvents.ValueBoolPointer(),
		PushChannel:                data.PushChannel.ValueStringPointer(),
		IssueChannel:               data.IssueChannel.ValueStringPointer(),
		ConfidentialIssueChannel:   data.ConfidentialIssueChannel.ValueStringPointer(),
		MergeRequestChannel:        data.MergeRequestChannel.ValueStringPointer(),
		NoteChannel:                data.NoteChannel.ValueStringPointer(),
		ConfidentialNoteChannel:    data.ConfidentialNoteChannel.ValueStringPointer(),
		TagPushChannel:             data.TagPushChannel.ValueStringPointer(),
		PipelineChannel:            data.PipelineChannel.ValueStringPointer(),
		WikiPageChannel:            data.WikiPageChannel.ValueStringPointer(),
	}

	if !data.Channel.IsNull() {
		opts.Channel = gitlab.Ptr(data.Channel.ValueString())
	}

	return opts
}

// getGroupMattermostIntegration fetches the Mattermost integration for a group using raw HTTP.
// This is necessary because client-go's GroupMattermostIntegration struct returns a generic Integration struct
// instead of a more detailed MattermostIntegration struct, so values inside the `properties` block aren't returned.
// This will be fixed in client-go 2.0 and this should be updated then.
//
// TODO: Remove this when client-go is fixed and use r.client.Integrations.GetGroupMattermostIntegration instead.
func (r *gitlabGroupIntegrationMattermostResource) getGroupMattermostIntegration(ctx context.Context, groupID string) (*groupMattermostIntegrationResponse, error) {
	path := fmt.Sprintf("groups/%s/integrations/mattermost", groupID)

	req, err := r.client.NewRequest(http.MethodGet, path, nil, []gitlab.RequestOptionFunc{gitlab.WithContext(ctx)})
	if err != nil {
		return nil, err
	}

	var service groupMattermostIntegrationResponse
	_, err = r.client.Do(req, &service)
	if err != nil {
		return nil, err
	}

	return &service, nil
}

// apiToStateModel maps the GitLab API response to the Terraform state model.
func (r *gitlabGroupIntegrationMattermostResource) apiToStateModel(service *groupMattermostIntegrationResponse, data *gitlabGroupIntegrationMattermostResourceModel) {
	data.PushEvents = types.BoolValue(service.PushEvents)
	data.IssuesEvents = types.BoolValue(service.IssuesEvents)
	data.ConfidentialIssuesEvents = types.BoolValue(service.ConfidentialIssuesEvents)
	data.MergeRequestsEvents = types.BoolValue(service.MergeRequestsEvents)
	data.TagPushEvents = types.BoolValue(service.TagPushEvents)
	data.NoteEvents = types.BoolValue(service.NoteEvents)
	data.ConfidentialNoteEvents = types.BoolValue(service.ConfidentialNoteEvents)
	data.PipelineEvents = types.BoolValue(service.PipelineEvents)
	data.WikiPageEvents = types.BoolValue(service.WikiPageEvents)

	if service.Properties != nil {
		data.Username = types.StringValue(service.Properties.Username)
		data.Channel = types.StringValue(service.Properties.Channel)
		data.NotifyOnlyBrokenPipelines = types.BoolValue(service.Properties.NotifyOnlyBrokenPipelines)
		data.BranchesToBeNotified = types.StringValue(service.Properties.BranchesToBeNotified)
		data.LabelsToBeNotified = types.StringValue(service.Properties.LabelsToBeNotified)
		data.LabelsToBeNotifiedBehavior = types.StringValue(service.Properties.LabelsToBeNotifiedBehavior)
		data.PushChannel = types.StringValue(service.Properties.PushChannel)
		data.IssueChannel = types.StringValue(service.Properties.IssueChannel)
		data.ConfidentialIssueChannel = types.StringValue(service.Properties.ConfidentialIssueChannel)
		data.MergeRequestChannel = types.StringValue(service.Properties.MergeRequestChannel)
		data.NoteChannel = types.StringValue(service.Properties.NoteChannel)
		data.ConfidentialNoteChannel = types.StringValue(service.Properties.ConfidentialNoteChannel)
		data.TagPushChannel = types.StringValue(service.Properties.TagPushChannel)
		data.PipelineChannel = types.StringValue(service.Properties.PipelineChannel)
		data.WikiPageChannel = types.StringValue(service.Properties.WikiPageChannel)
	}
}
