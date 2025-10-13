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
	_ datasource.DataSource              = &gitlabProjectHooksDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectHooksDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectHooksDataSource)
}

func NewGitlabProjectHooksDataSource() datasource.DataSource {
	return &gitlabProjectHooksDataSource{}
}

type gitlabProjectHooksDataSource struct {
	client *gitlab.Client
}

type gitlabProjectHooksDataSourceModel struct {
	ID      types.String                                  `tfsdk:"id"`
	Project types.String                                  `tfsdk:"project"`
	Hooks   []gitlabProjectHooksIndividualDataSourceModel `tfsdk:"hooks"`
}

type gitlabProjectHooksIndividualDataSourceModel struct {
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

func (d *gitlabProjectHooksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_hooks"
}

func (d *gitlabProjectHooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_hooks`" + ` data source allows to retrieve details about hooks in a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_webhooks/#list-webhooks-for-a-project)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project.",
				Required:            true,
			},
			"hooks": schema.ListNestedAttribute{
				MarkdownDescription: "The list of hooks.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project": schema.StringAttribute{
							MarkdownDescription: "The name or id of the project to add the hook to.",
							Computed:            true,
						},
						"project_id": schema.Int64Attribute{
							MarkdownDescription: "The id of the project for the hook.",
							Computed:            true,
						},
						"hook_id": schema.Int64Attribute{
							MarkdownDescription: "The id of the project hook.",
							Computed:            true,
						},
						"url": schema.StringAttribute{
							MarkdownDescription: "The url of the hook to invoke.",
							Computed:            true,
						},
						"token": schema.StringAttribute{
							MarkdownDescription: "A token to present when invoking the hook. The token is only available on resource creation, not in this datasource. It will always be blank. Will be removed in 19.0.",
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
				},
			},
		},
	}
}

func (d *gitlabProjectHooksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectHooksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectHooksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	options := gitlab.ListProjectHooksOptions{}

	hooks, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.ProjectHook, *gitlab.Response, error) {
		return d.client.Projects.ListProjectHooks(project, &options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project hooks: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(project)
	for _, hook := range hooks {
		modelHook := gitlabProjectHooksIndividualDataSourceModel{
			Project:                  types.StringValue(project),
			ProjectID:                types.Int64Value(int64(hook.ProjectID)),
			HookID:                   types.Int64Value(int64(hook.ID)),
			URL:                      types.StringValue(hook.URL),
			Token:                    types.StringValue(""),
			PushEvents:               types.BoolValue(hook.PushEvents),
			PushEventsBranchFilter:   types.StringValue(hook.PushEventsBranchFilter),
			IssuesEvents:             types.BoolValue(hook.IssuesEvents),
			ConfidentialIssuesEvents: types.BoolValue(hook.ConfidentialIssuesEvents),
			MergeRequestsEvents:      types.BoolValue(hook.MergeRequestsEvents),
			TagPushEvents:            types.BoolValue(hook.TagPushEvents),
			NoteEvents:               types.BoolValue(hook.NoteEvents),
			ConfidentialNoteEvents:   types.BoolValue(hook.ConfidentialNoteEvents),
			JobEvents:                types.BoolValue(hook.JobEvents),
			PipelineEvents:           types.BoolValue(hook.PipelineEvents),
			WikiPageEvents:           types.BoolValue(hook.WikiPageEvents),
			DeploymentEvents:         types.BoolValue(hook.DeploymentEvents),
			ReleasesEvents:           types.BoolValue(hook.ReleasesEvents),
			EnableSSLVerification:    types.BoolValue(hook.EnableSSLVerification),
			CustomWebhookTemplate:    types.StringValue(hook.CustomWebhookTemplate),
		}
		data.Hooks = append(data.Hooks, modelHook)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
