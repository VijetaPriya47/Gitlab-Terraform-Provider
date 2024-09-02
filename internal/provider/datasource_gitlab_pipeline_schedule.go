package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabPipelineScheduleDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabPipelineScheduleDataSource{}
)

func init() {
	registerDataSource(NewGitLabPipelineScheduleDataSource)
}

// NewGitLabPipelineScheduleDataSource is a helper function to simplify the provider implementation.
func NewGitLabPipelineScheduleDataSource() datasource.DataSource {
	return &gitlabPipelineScheduleDataSource{}
}

// gitlabPipelineScheduleDataSource is the data source implementation.
type gitlabPipelineScheduleDataSource struct {
	client *gitlab.Client
}

// gitLabPipelineScheduleDataSourceModel describes the data source data model.
type gitLabPipelineScheduleDataSourceModel struct {
	ID                 types.String                        `tfsdk:"id"`
	PipelineScheduleID types.Int64                         `tfsdk:"pipeline_schedule_id"`
	Project            types.String                        `tfsdk:"project"`
	Description        types.String                        `tfsdk:"description"`
	Ref                types.String                        `tfsdk:"ref"`
	Cron               types.String                        `tfsdk:"cron"`
	CronTimezone       types.String                        `tfsdk:"cron_timezone"`
	NextRunAt          types.String                        `tfsdk:"next_run_at"`
	Active             types.Bool                          `tfsdk:"active"`
	CreatedAt          types.String                        `tfsdk:"created_at"`
	UpdatedAt          types.String                        `tfsdk:"updated_at"`
	LastPipeline       *gitlabPipelineScheduleLastPipeline `tfsdk:"last_pipeline"`
	Owner              *gitlabPipelineScheduleOwner        `tfsdk:"owner"`
	Variables          []*gitlabPipelineScheduleVariable   `tfsdk:"variables"`
}

// The details of the last pipeline run by the schedule
type gitlabPipelineScheduleLastPipeline struct {
	ID     types.Int64  `tfsdk:"id"`
	Sha    types.String `tfsdk:"sha"`
	Ref    types.String `tfsdk:"ref"`
	Status types.String `tfsdk:"status"`
}

// The details of the pipeline schedule owner
type gitlabPipelineScheduleOwner struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Username  types.String `tfsdk:"username"`
	State     types.String `tfsdk:"state"`
	AvatarUrl types.String `tfsdk:"avatar_url"`
	WebUrl    types.String `tfsdk:"web_url"`
}

// The details of a pipeline schedule variable
type gitlabPipelineScheduleVariable struct {
	Key          types.String `tfsdk:"key"`
	VariableType types.String `tfsdk:"variable_type"`
	Value        types.String `tfsdk:"value"`
}

// Metadata returns the data source type name.
func (d *gitlabPipelineScheduleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_schedule"
}

// Schema defines the schema for the data source.
func (d *gitlabPipelineScheduleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_pipeline_schedule`" + ` data source retrieves information about a gitlab pipeline schedule for a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/pipeline_schedules.html)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id>:<pipeline-schedule-id>`.",
				Computed:            true,
			},
			"pipeline_schedule_id": schema.Int64Attribute{
				MarkdownDescription: "The pipeline schedule id.",
				Required:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the schedule to.",
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
			"last_pipeline": schema.SingleNestedAttribute{
				MarkdownDescription: "The details of the last pipeline run by the schedule.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						MarkdownDescription: "The pipeline ID.",
						Computed:            true,
					},
					"sha": schema.StringAttribute{
						MarkdownDescription: "The SHA of the pipeline.",
						Computed:            true,
					},
					"ref": schema.StringAttribute{
						MarkdownDescription: "The ref of the pipeline.",
						Computed:            true,
					},
					"status": schema.StringAttribute{
						MarkdownDescription: "The status of pipelines, one of: created, waiting_for_resource, preparing, pending, running, success, failed, canceled, skipped, manual, scheduled.",
						Computed:            true,
					},
				},
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
			"variables": schema.ListNestedAttribute{
				MarkdownDescription: "The list of the pipeline schedule variables.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							MarkdownDescription: "The key of a variable.",
							Computed:            true,
						},
						"variable_type": schema.StringAttribute{
							MarkdownDescription: "The type of a variable, one of: env_var and file.",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "The value of a variable.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabPipelineScheduleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*gitlab.Client)
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabPipelineScheduleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabPipelineScheduleDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Make API call to read pipeline schedules
	schedule, _, err := d.client.PipelineSchedules.GetPipelineSchedule(state.Project.ValueString(), int(state.PipelineScheduleID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read pipeline schedule details: %s", err.Error()))
		return
	}

	state.ID = types.StringValue(utils.BuildTwoPartID(state.Project.ValueStringPointer(), gitlab.Ptr(strconv.Itoa(int(state.PipelineScheduleID.ValueInt64())))))
	state.Description = types.StringValue(schedule.Description)
	state.Ref = types.StringValue(schedule.Ref)
	state.Cron = types.StringValue(schedule.Cron)
	state.CronTimezone = types.StringValue(schedule.CronTimezone)
	state.NextRunAt = types.StringValue(schedule.NextRunAt.Format(time.RFC3339))
	state.Active = types.BoolValue(schedule.Active)
	state.CreatedAt = types.StringValue(schedule.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(schedule.UpdatedAt.Format(time.RFC3339))

	if schedule.LastPipeline != nil {
		state.LastPipeline = &gitlabPipelineScheduleLastPipeline{
			ID:     types.Int64Value(int64(schedule.LastPipeline.ID)),
			Sha:    types.StringValue(schedule.LastPipeline.SHA),
			Ref:    types.StringValue(schedule.LastPipeline.Ref),
			Status: types.StringValue(schedule.LastPipeline.Status),
		}
	}

	state.Owner = &gitlabPipelineScheduleOwner{
		ID:        types.Int64Value(int64(schedule.Owner.ID)),
		Name:      types.StringValue(schedule.Owner.Name),
		Username:  types.StringValue(schedule.Owner.Username),
		State:     types.StringValue(schedule.Owner.State),
		AvatarUrl: types.StringValue(schedule.Owner.AvatarURL),
		WebUrl:    types.StringValue(schedule.Owner.WebURL),
	}

	for _, variable := range schedule.Variables {
		state.Variables = append(state.Variables, &gitlabPipelineScheduleVariable{
			Key:          types.StringValue(variable.Key),
			VariableType: types.StringValue(string(variable.VariableType)),
			Value:        types.StringValue(variable.Value),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
