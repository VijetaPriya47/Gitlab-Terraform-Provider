package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	// "gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ datasource.DataSource              = &gitlabProjectIssueLabelEventsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectIssueLabelEventsDataSource{}
)

func NewGitLabProjectIssueLabelEventsDataSource() datasource.DataSource {
	return &gitlabProjectIssueLabelEventsDataSource{}
}

func init() {
	registerDataSource(NewGitLabProjectIssueLabelEventsDataSource)
}

// gitlabProjectIssueLabelEventsDataSource is the data source implementation.
type gitlabProjectIssueLabelEventsDataSource struct {
	client *gitlab.Client
}

// gitlabProjectIssueLabelEventsDataSourceModel describes the data source data model.
type gitlabProjectIssueLabelEventsDataSourceModel struct {
	Id            types.String      `tfsdk:"id"`
	Project       types.String      `tfsdk:"project"`
	IssueIID      types.Int64       `tfsdk:"issue_iid"`
	PagesReturned types.Int64       `tfsdk:"pages_returned"`
	Events        []labelEventModel `tfsdk:"events"`
}

// labelEventModel describes a single label event.
type labelEventModel struct {
	Id           types.Int64  `tfsdk:"id"`
	CreatedAt    types.String `tfsdk:"created_at"`
	Action       types.String `tfsdk:"action"`
	Label        labelModel   `tfsdk:"label"`
	User         userModel    `tfsdk:"user"`
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceId   types.Int64  `tfsdk:"resource_id"`
}

// labelModel describes a GitLab label.
type labelModel struct {
	Id          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	Description types.String `tfsdk:"description"`
}

// userModel describes a GitLab user.
type userModel struct {
	Id        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Username  types.String `tfsdk:"username"`
	AvatarUrl types.String `tfsdk:"avatar_url"`
	WebUrl    types.String `tfsdk:"web_url"`
	State     types.String `tfsdk:"state"`
}

// Metadata returns the data source type name.
func (d *gitlabProjectIssueLabelEventsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_issue_label_events"
}

// GetSchema defines the schema for the data source.
func (d *gitlabProjectIssueLabelEventsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_issue_label_events`" + ` data source retrieves label events for a specific GitLab project issue.

**Upstream API**: [GitLab Resource Label Events API docs](https://docs.gitlab.com/api/resource_label_events/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<issue_iid>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project.",
				Required:            true,
			},
			"issue_iid": schema.Int64Attribute{
				MarkdownDescription: "The internal ID of the issue.",
				Required:            true,
			},
			"pages_returned": schema.Int64Attribute{
				MarkdownDescription: "Number of pages to return. Default is 1.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"events": schema.ListNestedAttribute{
				MarkdownDescription: "List of label events for the issue.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the label event.",
							Computed:            true,
						},
						"resource_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the resource associated with the label event.",
							Computed:            true,
						},
						"resource_type": schema.StringAttribute{
							MarkdownDescription: "The type of the resource associated with the label event.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "The date and time when the label event was created.",
							Computed:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "The action performed on the label (add, remove).",
							Computed:            true,
						},
						"label": schema.SingleNestedAttribute{
							MarkdownDescription: "The label that was added or removed.",
							Computed:            true,
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
									MarkdownDescription: "The color of the label.",
									Computed:            true,
								},
								"description": schema.StringAttribute{
									MarkdownDescription: "The description of the label.",
									Computed:            true,
								},
							},
						},
						"user": schema.SingleNestedAttribute{
							MarkdownDescription: "The user who performed the action.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"id": schema.Int64Attribute{
									MarkdownDescription: "The ID of the user.",
									Computed:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "The name of the user.",
									Computed:            true,
								},
								"username": schema.StringAttribute{
									MarkdownDescription: "The username of the user.",
									Computed:            true,
								},
								"avatar_url": schema.StringAttribute{
									MarkdownDescription: "The avatar URL of the user.",
									Computed:            true,
								},
								"web_url": schema.StringAttribute{
									MarkdownDescription: "The web URL of the user.",
									Computed:            true,
								},
								"state": schema.StringAttribute{
									MarkdownDescription: "The state of the user.",
									Computed:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildLabelModelFromEventLabel(e *gitlab.LabelEvent) labelModel {
	if types.Int64Value(e.ID).IsNull() {
		return labelModel{} // no valid label
	}
	return labelModel{
		Id:          types.Int64Value(e.Label.ID),
		Name:        types.StringValue(e.Label.Name),
		Color:       types.StringValue(e.Label.Color),
		Description: types.StringValue(e.Label.Description),
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabProjectIssueLabelEventsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectIssueLabelEventsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabProjectIssueLabelEventsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	PagesReturned := int(data.PagesReturned.ValueInt64())
	project := data.Project.ValueString()
	issueIID := data.IssueIID.ValueInt64()
	if PagesReturned == 0 {
		PagesReturned = 1
	}

	opts := &gitlab.ListLabelEventsOptions{
		ListOptions: gitlab.ListOptions{},
	}

	var allEvents []*gitlab.LabelEvent
	var events []*gitlab.LabelEvent
	var respMeta *gitlab.Response
	var err error
	// Paginate through all label events
	for {
		events, respMeta, err = d.client.ResourceLabelEvents.ListIssueLabelEvents(project, issueIID, opts, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to read issue label events",
				fmt.Sprintf("Unable to read label events for project %q issue %d: %s", project, issueIID, err),
			)
			return
		}
		// Append current page of events to the total collection
		allEvents = append(allEvents, events...)
		// Break the loop if we've reached the last page
		if (opts.Page == 0 && len(events) == 0) || PagesReturned == 1 {
			break
		}

		// Move to the next page
		opts.Page = respMeta.NextPage
		PagesReturned--
	}

	data.Id = types.StringValue(fmt.Sprintf("%s:%d", project, issueIID))
	data.Project = types.StringValue(project)
	data.IssueIID = types.Int64Value(issueIID)

	// Convert API response to Terraform model
	data.Events = []labelEventModel{}
	for _, event := range allEvents {
		eventVariable := labelEventModel{
			Id:           types.Int64Value(event.ID),
			CreatedAt:    types.StringValue(event.CreatedAt.Format(time.RFC3339)),
			Action:       types.StringValue(event.Action),
			ResourceType: types.StringValue(event.ResourceType),
			ResourceId:   types.Int64Value(event.ResourceID),
			Label:        buildLabelModelFromEventLabel(event),
			User: userModel{
				Id:        types.Int64Value(event.User.ID),
				Name:      types.StringValue(event.User.Name),
				Username:  types.StringValue(event.User.Username),
				AvatarUrl: types.StringValue(event.User.AvatarURL),
				WebUrl:    types.StringValue(event.User.WebURL),
				State:     types.StringValue(event.User.State),
			},
		}
		data.Events = append(data.Events, eventVariable)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
