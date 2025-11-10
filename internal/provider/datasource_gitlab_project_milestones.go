package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabProjectMilestonesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectMilestonesDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectMilestonesDataSource)
}

func NewGitlabProjectMilestonesDataSource() datasource.DataSource {
	return &gitlabProjectMilestonesDataSource{}
}

type gitlabProjectMilestonesDataSource struct {
	client *gitlab.Client
}

type gitlabProjectMilestonesDataSourceModel struct {
	ID                      types.String                                       `tfsdk:"id"`
	Project                 types.String                                       `tfsdk:"project"`
	IIDs                    types.List                                         `tfsdk:"iids"`
	State                   types.String                                       `tfsdk:"state"`
	Title                   types.String                                       `tfsdk:"title"`
	Search                  types.String                                       `tfsdk:"search"`
	IncludeParentMilestones types.Bool                                         `tfsdk:"include_parent_milestones"`
	Milestones              []gitlabProjectMilestonesIndividualDataSourceModel `tfsdk:"milestones"`
}

type gitlabProjectMilestonesIndividualDataSourceModel struct {
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

func (d *gitlabProjectMilestonesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_milestones"
}

func (d *gitlabProjectMilestonesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_milestones`" + ` data source allows get details of a project milestones.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/milestones/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this datasource. In the format `<project:options-hash>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project owned by the authenticated user.",
				Required:            true,
			},
			"iids": schema.ListAttribute{
				MarkdownDescription: "Return only the milestones having the given `iid` (Note: ignored if `include_parent_milestones` is set as `true`).",
				ElementType:         types.Int64Type,
				Optional:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Return only `active` or `closed` milestones.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf(validMilestoneStates...)},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Return only the milestones having the given `title`.",
				Optional:            true,
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Return only milestones with a title or description matching the provided string.",
				Optional:            true,
			},
			"include_parent_milestones": schema.BoolAttribute{
				MarkdownDescription: "Include group milestones from parent group and its ancestors.",
				Optional:            true,
			},
			"milestones": schema.ListNestedAttribute{
				MarkdownDescription: "List of milestones from a project.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project": schema.StringAttribute{
							MarkdownDescription: "The ID or URL-encoded path of the project owned by the authenticated user.",
							Computed:            true,
						},
						"milestone_id": schema.Int64Attribute{
							MarkdownDescription: "The instance-wide ID of the project's milestone.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *gitlabProjectMilestonesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectMilestonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectMilestonesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	options := gitlab.ListMilestonesOptions{}
	var optionsHash strings.Builder

	if !data.IIDs.IsNull() && !data.IIDs.IsUnknown() {
		var IIDs []int
		data.IIDs.ElementsAs(ctx, &IIDs, true)
		options.IIDs = &IIDs
		optionsHash.WriteString(fmt.Sprint(IIDs))
	}

	if !data.Title.IsNull() && !data.Title.IsUnknown() {
		options.Title = data.Title.ValueStringPointer()
		optionsHash.WriteString(data.Title.ValueString())
	}

	if !data.State.IsNull() && !data.State.IsUnknown() {
		options.State = data.State.ValueStringPointer()
		optionsHash.WriteString(data.State.ValueString())
	}

	if !data.Search.IsNull() && !data.Search.IsUnknown() {
		options.Search = data.Search.ValueStringPointer()
		optionsHash.WriteString(data.Search.ValueString())
	}

	if !data.IncludeParentMilestones.IsNull() && !data.IncludeParentMilestones.IsUnknown() {
		options.IncludeParentMilestones = data.IncludeParentMilestones.ValueBoolPointer()     //nolint:staticcheck
		optionsHash.WriteString(strconv.FormatBool(data.IncludeParentMilestones.ValueBool())) //nolint:staticcheck
	}

	milestones, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Milestone, *gitlab.Response, error) {
		return d.client.Milestones.ListMilestones(project, &options, p, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project milestones: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("get gitlab milestones from project: %s", project))
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", project, optionsHash.String()))

	data.Milestones = []gitlabProjectMilestonesIndividualDataSourceModel{}
	for _, milestone := range milestones {
		modelMilestone := gitlabProjectMilestonesIndividualDataSourceModel{
			IID:         types.Int64Value(int64(milestone.IID)),
			MilestoneID: types.Int64Value(int64(milestone.ID)),
			Project:     types.StringValue(project),
			ProjectID:   types.Int64Value(int64(milestone.ProjectID)),
			Title:       types.StringValue(milestone.Title),
			Description: types.StringValue(milestone.Description),
			State:       types.StringValue(milestone.State),
			WebURL:      types.StringValue(milestone.WebURL),
		}
		if milestone.DueDate == nil {
			modelMilestone.DueDate = types.StringNull()
		} else {
			modelMilestone.DueDate = types.StringValue(milestone.DueDate.String())
		}
		if milestone.StartDate == nil {
			modelMilestone.StartDate = types.StringNull()
		} else {
			modelMilestone.StartDate = types.StringValue(milestone.StartDate.String())
		}
		if milestone.UpdatedAt == nil {
			modelMilestone.UpdatedAt = types.StringNull()
		} else {
			modelMilestone.UpdatedAt = types.StringValue(milestone.UpdatedAt.String())
		}
		if milestone.CreatedAt == nil {
			modelMilestone.CreatedAt = types.StringNull()
		} else {
			modelMilestone.CreatedAt = types.StringValue(milestone.CreatedAt.String())
		}
		if milestone.Expired == nil {
			modelMilestone.Expired = types.BoolValue(false)
		} else {
			modelMilestone.Expired = types.BoolValue(*milestone.Expired)
		}
		data.Milestones = append(data.Milestones, modelMilestone)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
