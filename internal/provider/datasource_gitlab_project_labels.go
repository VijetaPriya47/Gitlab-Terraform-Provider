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
	_ datasource.DataSource              = &gitlabProjectLabelsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectLabelsDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectLabelsDataSource)
}

func NewGitlabProjectLabelsDataSource() datasource.DataSource {
	return &gitlabProjectLabelsDataSource{}
}

type gitlabProjectLabelsDataSource struct {
	client *gitlab.Client
}

func (d *gitlabProjectLabelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_labels"
}

type gitlabProjectLabelsDataSourceModel struct {
	ID      types.String                              `tfsdk:"id"`
	Project types.String                              `tfsdk:"project"`
	Labels  []gitlabProjectLabelsDataSourceLabelModel `tfsdk:"labels"`
}

type gitlabProjectLabelsDataSourceLabelModel struct {
	ID                     types.Int64  `tfsdk:"id"`
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

func (d *gitlabProjectLabelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_labels`" + ` data source retrieves a list of labels for a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/labels/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
			},
			"labels": schema.ListNestedAttribute{
				MarkdownDescription: "The list of labels in the project.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the label.",
							Computed:            true,
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the datasource.
func (d *gitlabProjectLabelsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *gitlabProjectLabelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabProjectLabelsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()

	// Get all labels from GitLab API with pagination
	labels, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
		return d.client.Labels.ListLabels(project, nil, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get project labels", fmt.Sprintf("Error retrieving labels from project '%s': %s", project, err.Error()))
		return
	}

	// Set the ID to the project identifier
	data.ID = types.StringValue(project)

	// Map the API response to the model
	data.Labels = make([]gitlabProjectLabelsDataSourceLabelModel, len(labels))
	for i, label := range labels {
		labelModel := gitlabProjectLabelsDataSourceLabelModel{
			ID:                     types.Int64Value(int64(label.ID)),
			Name:                   types.StringValue(label.Name),
			Color:                  types.StringValue(label.Color),
			TextColor:              types.StringValue(label.TextColor),
			Description:            types.StringValue(label.Description),
			OpenIssuesCount:        types.Int64Value(int64(label.OpenIssuesCount)),
			ClosedIssuesCount:      types.Int64Value(int64(label.ClosedIssuesCount)),
			OpenMergeRequestsCount: types.Int64Value(int64(label.OpenMergeRequestsCount)),
			Subscribed:             types.BoolValue(label.Subscribed),
			Priority:               types.Int64Value(int64(label.Priority)),
			IsProjectLabel:         types.BoolValue(label.IsProjectLabel),
		}

		data.Labels[i] = labelModel
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
