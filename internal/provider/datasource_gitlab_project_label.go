package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

var (
	_ datasource.DataSource              = &gitlabProjectLabelDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectLabelDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectLabelDataSource)
}

func NewGitlabProjectLabelDataSource() datasource.DataSource {
	return &gitlabProjectLabelDataSource{}
}

type gitlabProjectLabelDataSource struct {
	client *gitlab.Client
}

func (d *gitlabProjectLabelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_label"
}

type gitlabProjectLabelDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Project                types.String `tfsdk:"project"`
	LabelID                types.Int64  `tfsdk:"label_id"`
	Name                   types.String `tfsdk:"name"`
	Color                  types.String `tfsdk:"color"`
	TextColor              types.String `tfsdk:"text_color"`
	Description            types.String `tfsdk:"description"`
	OpenIssuesCount        types.Int64  `tfsdk:"open_issues_count"`
	ClosedIssuesCount      types.Int64  `tfsdk:"closed_issues_count"`
	OpenMergeRequestsCount types.Int64  `tfsdk:"open_merge_requests_count"`
	Subscribed             types.Bool   `tfsdk:"subscribed"`
	Priority               types.Int64  `tfsdk:"priority"`
	IsProjectLabel         types.Bool   `tfsdk:"is_project_label"`
}

func (d *gitlabProjectLabelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_label`" + ` data source retrieves details about a project label.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/labels/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the label in the format `project:label_id`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
			},
			"label_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the label.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the label.",
				Computed:            true,
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "The color of the label given in 6-digit hex notation with leading '#' sign.",
				Computed:            true,
			},
			"text_color": schema.StringAttribute{
				MarkdownDescription: "The text color of the label given in 6-digit hex notation with leading '#' sign.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the label.",
				Computed:            true,
			},
			"open_issues_count": schema.Int64Attribute{
				MarkdownDescription: "The number of open issues with this label.",
				Computed:            true,
			},
			"closed_issues_count": schema.Int64Attribute{
				MarkdownDescription: "The number of closed issues with this label.",
				Computed:            true,
			},
			"open_merge_requests_count": schema.Int64Attribute{
				MarkdownDescription: "The number of open merge requests with this label.",
				Computed:            true,
			},
			"subscribed": schema.BoolAttribute{
				MarkdownDescription: "Whether the authenticated user is subscribed to the label.",
				Computed:            true,
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "The priority of the label. Null if no priority is set.",
				Computed:            true,
			},
			"is_project_label": schema.BoolAttribute{
				MarkdownDescription: "Whether the label is a project label.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the datasource.
func (d *gitlabProjectLabelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*GitLabDatasourceData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *GitLabDatasourceData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = providerData.Client
}

func (d *gitlabProjectLabelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabProjectLabelDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	labelID := data.LabelID.ValueInt64()

	// Get the label from GitLab API using label ID
	label, _, err := d.client.Labels.GetLabel(project, fmt.Sprintf("%d", labelID), gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get project label", fmt.Sprintf("Error retrieving label with ID %d from project '%s': %s", labelID, project, err.Error()))
		return
	}

	// Build composite ID using the project value as provided by the user.
	// This can be either a numeric ID or a path - both work since ":" will be URL-encoded in paths.
	data.ID = types.StringValue(fmt.Sprintf("%s:%d", project, label.ID))
	data.Name = types.StringValue(label.Name)
	data.Color = types.StringValue(label.Color)
	data.TextColor = types.StringValue(label.TextColor)
	data.Description = types.StringValue(label.Description)
	data.OpenIssuesCount = types.Int64Value(int64(label.OpenIssuesCount))
	data.ClosedIssuesCount = types.Int64Value(int64(label.ClosedIssuesCount))
	data.OpenMergeRequestsCount = types.Int64Value(int64(label.OpenMergeRequestsCount))
	data.Subscribed = types.BoolValue(label.Subscribed)
	if label.Priority.IsSpecified() {
		if priority, err := label.Priority.Get(); err == nil {
			data.Priority = types.Int64Value(priority)
		} else {
			data.Priority = types.Int64Null()
		}
	} else {
		data.Priority = types.Int64Null()
	}
	data.IsProjectLabel = types.BoolValue(label.IsProjectLabel)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
