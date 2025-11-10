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
	_ datasource.DataSource              = &gitlabGroupHooksDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabGroupHooksDataSource{}
)

func init() {
	registerDataSource(NewGitlabGroupHooksDataSource)
}

func NewGitlabGroupHooksDataSource() datasource.DataSource {
	return &gitlabGroupHooksDataSource{}
}

type gitlabGroupHooksDataSource struct {
	client *gitlab.Client
}

type gitlabGroupHooksDataSourceModel struct {
	ID    types.String                                `tfsdk:"id"`
	Group types.String                                `tfsdk:"group"`
	Hooks []gitlabGroupHooksIndividualDataSourceModel `tfsdk:"hooks"`
}

type gitlabGroupHooksIndividualDataSourceModel struct {
	Group                    types.String `tfsdk:"group"`
	GroupID                  types.Int64  `tfsdk:"group_id"`
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
	SubGroupEvents           types.Bool   `tfsdk:"subgroup_events"`
	EmojiEvents              types.Bool   `tfsdk:"emoji_events"`
	EnableSSLVerification    types.Bool   `tfsdk:"enable_ssl_verification"`
	CustomWebhookTemplate    types.String `tfsdk:"custom_webhook_template"`
}

func (d *gitlabGroupHooksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_hooks"
}

func (d *gitlabGroupHooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_hooks`" + ` data source retrieves details about hooks in a group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_webhooks/#list-group-hooks)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source. In the format `<group>`.",
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the group.",
				Required:            true,
			},
			"hooks": schema.ListNestedAttribute{
				MarkdownDescription: "The list of hooks.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group": schema.StringAttribute{
							MarkdownDescription: "The ID or full path of the group.",
							Computed:            true,
						},
						"group_id": schema.Int64Attribute{
							MarkdownDescription: "The id of the group for the hook.",
							Computed:            true,
						},
						"hook_id": schema.Int64Attribute{
							MarkdownDescription: "The id of the group hook.",
							Computed:            true,
						},
						"url": schema.StringAttribute{
							MarkdownDescription: "The url of the hook to invoke.",
							Computed:            true,
						},
						"token": schema.StringAttribute{
							MarkdownDescription: "A token to present when invoking the hook. The token is only available on resource creation, not in this datasource. It will always be blank. To be removed in 19.0.",
							Computed:            true,
							DeprecationMessage:  "The token is only available on resource creation, not in this datasource. It will always be blank. To be removed in 19.0.",
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
						"subgroup_events": schema.BoolAttribute{
							MarkdownDescription: "Invoke the hook for subgroup events.",
							Computed:            true,
						},
						"emoji_events": schema.BoolAttribute{
							MarkdownDescription: "Invoke the hook for emoji events.",
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

func (d *gitlabGroupHooksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabGroupHooksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabGroupHooksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := data.Group.ValueString()
	options := gitlab.ListGroupHooksOptions{}
	hooks, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.GroupHook, *gitlab.Response, error) {
		return d.client.Groups.ListGroupHooks(group, &options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read group hooks: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(group)

	data.Hooks = []gitlabGroupHooksIndividualDataSourceModel{}
	for _, hook := range hooks {
		modelHook := gitlabGroupHooksIndividualDataSourceModel{
			Group:                    types.StringValue(group),
			GroupID:                  types.Int64Value(int64(hook.GroupID)),
			HookID:                   types.Int64Value(int64(hook.ID)),
			URL:                      types.StringValue(hook.URL),
			Token:                    types.StringValue(""), // Always blank as noted in deprecation message
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
			SubGroupEvents:           types.BoolValue(hook.SubGroupEvents),
			EmojiEvents:              types.BoolValue(hook.EmojiEvents),
			EnableSSLVerification:    types.BoolValue(hook.EnableSSLVerification),
			CustomWebhookTemplate:    types.StringValue(hook.CustomWebhookTemplate),
		}
		data.Hooks = append(data.Hooks, modelHook)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
