package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	_ datasource.DataSource              = &gitlabProjectHookDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectHookDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectHookDataSource)
}

func NewGitlabProjectHookDataSource() datasource.DataSource {
	return &gitlabProjectHookDataSource{}
}

type gitlabProjectHookDataSource struct {
	client *gitlab.Client
}

type gitlabProjectHookDataSourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Project                  types.String `tfsdk:"project"`
	ProjectID                types.Int64  `tfsdk:"project_id"`
	HookID                   types.Int64  `tfsdk:"hook_id"`
	URL                      types.String `tfsdk:"url"`
	Token                    types.String `tfsdk:"token"`
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
	CustomWebhookTemplate    types.String `tfsdk:"custom_webhook_template"`
}

func (d *gitlabProjectHookDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_hook"
}

func (d *gitlabProjectHookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_hook`" + ` data source retrieves details about a hook in a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_webhooks/#get-a-project-webhook)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project>:<hook-id>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the hook to.",
				Required:            true,
			},
			"project_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the project for the hook.",
				Computed:            true,
			},
			"hook_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the project hook.",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The url of the hook to invoke.",
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "A token to present when invoking the hook. The token is only available on resource creation, not in this datasource. It will always be blank.",
				Computed:            true,
				Sensitive:           true,
				DeprecationMessage:  "The token is only available on resource creation, not in this datasource. It will always be blank.",
			},
			"push_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for push events.",
				Computed:            true,
			},
			"push_events_branch_filter": schema.StringAttribute{
				MarkdownDescription: "Invoke the hook for push events on matching branches only.",
				Computed:            true,
			},
			"issues_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for issues events.",
				Computed:            true,
			},
			"confidential_issues_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for confidential issues events.",
				Computed:            true,
			},
			"merge_requests_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for merge requests.",
				Computed:            true,
			},
			"tag_push_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for tag push events.",
				Computed:            true,
			},
			"note_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for notes events.",
				Computed:            true,
			},
			"confidential_note_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for confidential notes events.",
				Computed:            true,
			},
			"job_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for job events.",
				Computed:            true,
			},
			"pipeline_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for pipeline events.",
				Computed:            true,
			},
			"wiki_page_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for wiki page events.",
				Computed:            true,
			},
			"deployment_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for deployment events.",
				Computed:            true,
			},
			"releases_events": schema.BoolAttribute{
				MarkdownDescription: "Invoke the hook for releases events.",
				Computed:            true,
			},
			"enable_ssl_verification": schema.BoolAttribute{
				MarkdownDescription: "Enable ssl verification when invoking the hook.",
				Computed:            true,
			},
			"custom_webhook_template": schema.StringAttribute{
				MarkdownDescription: "Set a custom webhook template.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabProjectHookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectHookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectHookDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()
	hookID := data.HookID.ValueInt64()

	hook, _, err := d.client.Projects.GetProjectHook(project, hookID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project hook: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d", project, hookID))
	data.Project = types.StringValue(project)
	data.ProjectID = types.Int64Value(int64(hook.ProjectID))
	data.HookID = types.Int64Value(int64(hook.ID))
	data.URL = types.StringValue(hook.URL)
	data.PushEvents = types.BoolValue(hook.PushEvents)
	data.PushEventsBranchFilter = types.StringValue(hook.PushEventsBranchFilter)
	data.IssuesEvents = types.BoolValue(hook.IssuesEvents)
	data.ConfidentialIssuesEvents = types.BoolValue(hook.ConfidentialIssuesEvents)
	data.MergeRequestsEvents = types.BoolValue(hook.MergeRequestsEvents)
	data.TagPushEvents = types.BoolValue(hook.TagPushEvents)
	data.NoteEvents = types.BoolValue(hook.NoteEvents)
	data.ConfidentialNoteEvents = types.BoolValue(hook.ConfidentialNoteEvents)
	data.JobEvents = types.BoolValue(hook.JobEvents)
	data.PipelineEvents = types.BoolValue(hook.PipelineEvents)
	data.WikiPageEvents = types.BoolValue(hook.WikiPageEvents)
	data.DeploymentEvents = types.BoolValue(hook.DeploymentEvents)
	data.ReleasesEvents = types.BoolValue(hook.ReleasesEvents)
	data.EnableSSLVerification = types.BoolValue(hook.EnableSSLVerification)
	data.CustomWebhookTemplate = types.StringValue(hook.CustomWebhookTemplate)
	data.Token = types.StringValue("") // Token is not available in API response for security reasons
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
