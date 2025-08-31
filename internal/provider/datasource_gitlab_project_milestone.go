package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabProjectMilestoneDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectMilestoneDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectMilestoneDataSource)
}

func NewGitlabProjectMilestoneDataSource() datasource.DataSource {
	return &gitlabProjectMilestoneDataSource{}
}

type gitlabProjectMilestoneDataSource struct {
	client *gitlab.Client
}

type gitlabProjectMilestoneDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Project     types.String `tfsdk:"project"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	DueDate     types.String `tfsdk:"due_date"`
	StartDate   types.String `tfsdk:"start_date"`
	State       types.String `tfsdk:"state"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Expired     types.Bool   `tfsdk:"expired"`
	IID         types.Int64  `tfsdk:"iid"`
	MilestoneID types.Int64  `tfsdk:"milestone_id"`
	ProjectID   types.Int64  `tfsdk:"project_id"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	WebURL      types.String `tfsdk:"web_url"`
}

func (d *gitlabProjectMilestoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_milestone"
}

func (d *gitlabProjectMilestoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	validMilestoneStates := []string{"active", "closed"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_milestone`" + ` data source allows get details of a project milestone.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/milestones/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:milestone-id>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project owned by the authenticated user.",
				Required:            true,
			},
			"milestone_id": schema.Int64Attribute{
				MarkdownDescription: "The instance-wide ID of the project’s milestone.",
				Required:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of a milestone.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the milestone.",
				Computed:            true,
			},
			"due_date": schema.StringAttribute{
				MarkdownDescription: "The due date of the milestone. Date time string in the format YYYY-MM-DD, for example 2016-03-11.",
				Computed:            true,
			},
			"start_date": schema.StringAttribute{
				MarkdownDescription: "The start date of the milestone. Date time string in the format YYYY-MM-DD, for example 2016-03-11.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The state of the milestone. Valid values are: %s.", utils.RenderValueListForDocs(validMilestoneStates)),
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time of creation of the milestone. Date time string, ISO 8601 formatted, for example 2016-03-11T03:45:40Z.",
				Computed:            true,
			},
			"expired": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if milestone expired.",
				Computed:            true,
			},
			"iid": schema.Int64Attribute{
				MarkdownDescription: "The ID of the project's milestone.",
				Computed:            true,
			},
			"project_id": schema.Int64Attribute{
				MarkdownDescription: "The project ID of milestone.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The last update time of the milestone. Date time string, ISO 8601 formatted, for example 2016-03-11T03:45:40Z.",
				Computed:            true,
			},
			"web_url": schema.StringAttribute{
				MarkdownDescription: "The web URL of the milestone.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabProjectMilestoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectMilestoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectMilestoneDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()
	milestoneID := int(data.MilestoneID.ValueInt64())

	milestone, _, err := d.client.Milestones.GetMilestone(project, milestoneID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project milestone: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d", project, milestone.ID))
	data.IID = types.Int64Value(int64(milestone.IID))
	data.MilestoneID = types.Int64Value(int64(milestone.ID))
	data.Project = types.StringValue(project)
	data.ProjectID = types.Int64Value(int64(milestone.ProjectID))
	data.Title = types.StringValue(milestone.Title)
	data.Description = types.StringValue(milestone.Description)
	data.State = types.StringValue(milestone.State)
	data.WebURL = types.StringValue(milestone.WebURL)

	if milestone.DueDate == nil {
		data.DueDate = types.StringNull()
	} else {
		data.DueDate = types.StringValue(milestone.DueDate.String())
	}
	if milestone.StartDate == nil {
		data.StartDate = types.StringNull()
	} else {
		data.StartDate = types.StringValue(milestone.StartDate.String())
	}
	if milestone.UpdatedAt == nil {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(milestone.UpdatedAt.Format(time.RFC3339))
	}
	if milestone.CreatedAt == nil {
		data.CreatedAt = types.StringNull()
	} else {
		data.CreatedAt = types.StringValue(milestone.CreatedAt.Format(time.RFC3339))
	}
	if milestone.Expired == nil {
		data.Expired = types.BoolValue(false)
	} else {
		data.Expired = types.BoolValue(*milestone.Expired)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
