package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/gitlab-org/api/client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabPipelineSchedulesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabPipelineSchedulesDataSource{}
)

func init() {
	registerDataSource(NewGitLabPipelineSchedulesDataSource)
}

// NewGitLabPipelineSchedulesDataSource is a helper function to simplify the provider implementation.
func NewGitLabPipelineSchedulesDataSource() datasource.DataSource {
	return &gitlabPipelineSchedulesDataSource{}
}

// gitlabPipelineSchedulesDataSource is the data source implementation.
type gitlabPipelineSchedulesDataSource struct {
	client *gitlab.Client
}

// gitLabPipelineSchedulesDataSourceModel describes the data source data model.
type gitLabPipelineSchedulesDataSourceModel struct {
	ID                types.String              `tfsdk:"id"`
	Project           types.String              `tfsdk:"project"`
	PipelineSchedules []*gitlabPipelineSchedule `tfsdk:"pipeline_schedules"`
}

type gitlabPipelineSchedule struct {
	ID           types.Int64                  `tfsdk:"id"`
	Description  types.String                 `tfsdk:"description"`
	Ref          types.String                 `tfsdk:"ref"`
	Cron         types.String                 `tfsdk:"cron"`
	CronTimezone types.String                 `tfsdk:"cron_timezone"`
	NextRunAt    types.String                 `tfsdk:"next_run_at"`
	Active       types.Bool                   `tfsdk:"active"`
	CreatedAt    types.String                 `tfsdk:"created_at"`
	UpdatedAt    types.String                 `tfsdk:"updated_at"`
	Owner        *gitlabPipelineScheduleOwner `tfsdk:"owner"`
}

// Metadata returns the data source type name.
func (d *gitlabPipelineSchedulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_schedules"
}

// Schema defines the schema for the data source.
func (d *gitlabPipelineSchedulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_pipeline_schedule`" + ` data source retrieves information about a gitlab pipeline schedule for a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/pipeline_schedules.html)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the schedule to.",
				Required:            true,
			},
			"pipeline_schedules": schema.ListNestedAttribute{
				MarkdownDescription: "The list of pipeline schedules.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The pipeline schedule id.",
							Required:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "The description of the pipeline schedule.",
							Computed:            true,
						},
						"ref": schema.StringAttribute{
							MarkdownDescription: "The branch/tag name to be triggered. This will be the full branch reference, for example: `refs/heads/main`, not `main`.",
							Computed:            true,
						},
						"cron": schema.StringAttribute{
							MarkdownDescription: "The cron (e.g. `0 1 * * *`).",
							Computed:            true,
						},
						"cron_timezone": schema.StringAttribute{
							MarkdownDescription: "The timezone.",
							Optional:            true,
							Computed:            true,
						},
						"next_run_at": schema.StringAttribute{
							MarkdownDescription: "The datetime of when the schedule will next run.",
							Computed:            true,
						},
						"active": schema.BoolAttribute{
							MarkdownDescription: "The activation status of pipeline schedule.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "The datetime of when the schedule was created.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "The datetime of when the schedule was last updated.",
							Computed:            true,
						},
						"owner": schema.SingleNestedAttribute{
							MarkdownDescription: "The details of the pipeline schedule owner.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"id": schema.Int64Attribute{
									MarkdownDescription: "The user ID.",
									Computed:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "Name.",
									Computed:            true,
								},
								"username": schema.StringAttribute{
									MarkdownDescription: "Username.",
									Computed:            true,
								},
								"state": schema.StringAttribute{
									MarkdownDescription: "User's state, one of: active, blocked.",
									Computed:            true,
								},
								"avatar_url": schema.StringAttribute{
									MarkdownDescription: "Image URL for the user's avatar.",
									Computed:            true,
								},
								"web_url": schema.StringAttribute{
									MarkdownDescription: "URL to the user's profile.",
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

// Configure adds the provider configured client to the data source.
func (d *gitlabPipelineSchedulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabPipelineSchedulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabPipelineSchedulesDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Make API call to read pipeline schedules
	schedules, _, err := d.client.PipelineSchedules.ListPipelineSchedules(state.Project.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read pipeline schedules: %s", err.Error()))
		return
	}

	for _, schedule := range schedules {
		state.PipelineSchedules = append(state.PipelineSchedules, &gitlabPipelineSchedule{
			ID:           types.Int64Value(int64(schedule.ID)),
			Description:  types.StringValue(schedule.Description),
			Ref:          types.StringValue(schedule.Ref),
			Cron:         types.StringValue(schedule.Cron),
			CronTimezone: types.StringValue(schedule.CronTimezone),
			NextRunAt:    types.StringValue(schedule.NextRunAt.Format(time.RFC3339)),
			Active:       types.BoolValue(schedule.Active),
			CreatedAt:    types.StringValue(schedule.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:    types.StringValue(schedule.UpdatedAt.Format(time.RFC3339)),
			Owner: &gitlabPipelineScheduleOwner{
				ID:        types.Int64Value(int64(schedule.Owner.ID)),
				Name:      types.StringValue(schedule.Owner.Name),
				Username:  types.StringValue(schedule.Owner.Username),
				State:     types.StringValue(schedule.Owner.State),
				AvatarUrl: types.StringValue(schedule.Owner.AvatarURL),
				WebUrl:    types.StringValue(schedule.Owner.WebURL),
			},
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
