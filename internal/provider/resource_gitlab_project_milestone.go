package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	milestoneStateToStateEvent = map[string]string{
		"active": "activate",
		"closed": "close",
	}
	validStates = []string{"active", "closed"}
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabProjectMilestoneResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectMilestoneResource{}
	_ resource.ResourceWithImportState = &gitlabProjectMilestoneResource{}
)

func init() {
	registerResource(NewGitLabProjectMilestoneResource)
}

func NewGitLabProjectMilestoneResource() resource.Resource {
	return &gitlabProjectMilestoneResource{}
}

type gitlabProjectMilestoneResource struct {
	client *gitlab.Client
}

type gitlabProjectMilestoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Project     types.String `tfsdk:"project"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	DueDate     types.String `tfsdk:"due_date"`
	StartDate   types.String `tfsdk:"start_date"`
	State       types.String `tfsdk:"state"`
	IID         types.Int64  `tfsdk:"iid"`
	MilestoneID types.Int64  `tfsdk:"milestone_id"`
	ProjectID   types.Int64  `tfsdk:"project_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	WebURL      types.String `tfsdk:"web_url"`
	Expired     types.Bool   `tfsdk:"expired"`
}

// Metadata returns the resource name
func (r *gitlabProjectMilestoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_milestone"
}

func (r *gitlabProjectMilestoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_milestone`" + ` resource manages the lifecycle of a project milestone.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/milestones/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<milestone_id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project owned by the authenticated user.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of a milestone.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the milestone.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"due_date": schema.StringAttribute{
				MarkdownDescription: "The due date of the milestone. Date string in the format YYYY-MM-DD, for example 2016-03-11.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
				Validators:          []validator.String{utils.IsValidDate()},
			},
			"start_date": schema.StringAttribute{
				MarkdownDescription: "The start date of the milestone. Date string in the format YYYY-MM-DD, for example 2016-03-11.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
				Validators:          []validator.String{utils.IsValidDate()},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The state of the milestone. Valid values are: %s.", utils.RenderValueListForDocs(validStates)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("active"),
				Validators:          []validator.String{stringvalidator.OneOf(validStates...)},
			},
			"iid": schema.Int64Attribute{
				MarkdownDescription: "The ID of the project's milestone.",
				Computed:            true,
			},
			"milestone_id": schema.Int64Attribute{
				MarkdownDescription: "The instance-wide ID of the project's milestone.",
				Computed:            true,
			},
			"project_id": schema.Int64Attribute{
				MarkdownDescription: "The project ID of milestone.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time of creation of the milestone. Date time string, ISO 8601 formatted, for example 2016-03-11T03:45:40Z.",
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
			"expired": schema.BoolAttribute{
				MarkdownDescription: "Bool, true if milestone expired.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectMilestoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectMilestoneResource) milestoneToStateModel(project string, milestone *gitlab.Milestone, data *gitlabProjectMilestoneResourceModel) {
	data.Project = types.StringValue(project)
	data.IID = types.Int64Value(int64(milestone.IID))
	data.MilestoneID = types.Int64Value(int64(milestone.ID))
	data.ProjectID = types.Int64Value(int64(milestone.ProjectID))
	data.Title = types.StringValue(milestone.Title)
	data.Description = types.StringValue(milestone.Description)
	data.State = types.StringValue(milestone.State)
	data.WebURL = types.StringValue(milestone.WebURL)

	if milestone.DueDate != nil {
		data.DueDate = types.StringValue(milestone.DueDate.String())
	} else {
		data.DueDate = types.StringNull()
	}

	if milestone.StartDate != nil {
		data.StartDate = types.StringValue(milestone.StartDate.String())
	} else {
		data.StartDate = types.StringNull()
	}

	if milestone.CreatedAt != nil {
		data.CreatedAt = types.StringValue(milestone.CreatedAt.Format(time.RFC3339))
	} else {
		data.CreatedAt = types.StringNull()
	}

	if milestone.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(milestone.UpdatedAt.Format(time.RFC3339))
	} else {
		data.UpdatedAt = types.StringNull()
	}

	if milestone.Expired != nil {
		data.Expired = types.BoolValue(*milestone.Expired)
	} else {
		data.Expired = types.BoolValue(false)
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabProjectMilestoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectMilestoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectMilestoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	title := data.Title.ValueString()

	options := &gitlab.CreateMilestoneOptions{
		Title: &title,
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	if !data.StartDate.IsNull() && !data.StartDate.IsUnknown() {
		parsedStartDate, err := gitlab.ParseISOTime(data.StartDate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to parse start_date",
				fmt.Sprintf("Failed to parse start_date: %s. %v", data.StartDate.ValueString(), err),
			)
			return
		}
		options.StartDate = &parsedStartDate
	}

	if !data.DueDate.IsNull() && !data.DueDate.IsUnknown() {
		parsedDueDate, err := gitlab.ParseISOTime(data.DueDate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to parse due_date",
				fmt.Sprintf("Failed to parse due_date: %s. %v", data.DueDate.ValueString(), err),
			)
			return
		}
		options.DueDate = &parsedDueDate
	}

	tflog.Debug(ctx, "create project milestone", map[string]any{
		"project": project, "title": title,
	})
	milestone, _, err := r.client.Milestones.CreateMilestone(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create milestone: %s", err.Error()))
		return
	}

	// Handle state change if needed (state is not part of CREATE API, only UPDATE)
	if !data.State.IsNull() && !data.State.IsUnknown() && data.State.ValueString() != "active" {
		updateOptions := &gitlab.UpdateMilestoneOptions{
			StateEvent: gitlab.Ptr(milestoneStateToStateEvent[data.State.ValueString()]),
		}
		milestone, _, err = r.client.Milestones.UpdateMilestone(project, milestone.ID, updateOptions, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError(
				"GitLab API error occurred",
				fmt.Sprintf("Failed to update milestone ID %d in project %s right after creation: %v", milestone.ID, project, err),
			)
			return
		}
	}

	// Build ID and populate state
	milestoneIDStr := strconv.FormatInt(int64(milestone.ID), 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &milestoneIDStr))
	r.milestoneToStateModel(project, milestone, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "created a milestone", map[string]any{
		"project": project, "milestone": milestoneIDStr,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectMilestoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectMilestoneResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse ID to get project and milestone ID
	project, rawMilestoneID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<project>:<milestone_id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	milestoneID, err := strconv.ParseInt(rawMilestoneID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid milestone ID provided, milestone ID should be an Int",
			fmt.Sprintf("Unable to convert milestone id to int: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "read project milestone", map[string]any{
		"project": project, "milestone": milestoneID,
	})
	milestone, _, err := r.client.Milestones.GetMilestone(project, milestoneID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "milestone does not exist, removing from state", map[string]any{
				"project": project, "milestone": milestoneID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read gitlab milestone details: %s", err.Error()))
		return
	}

	r.milestoneToStateModel(project, milestone, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectMilestoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectMilestoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, rawMilestoneID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Update. It should be '<project>:<milestone_id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	milestoneID, err := strconv.ParseInt(rawMilestoneID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid milestone ID provided, milestone ID should be an Int",
			fmt.Sprintf("Unable to convert milestone id to int: %s", err.Error()),
		)
		return
	}

	options := &gitlab.UpdateMilestoneOptions{}

	if !data.Title.IsNull() && !data.Title.IsUnknown() {
		options.Title = gitlab.Ptr(data.Title.ValueString())
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	if !data.StartDate.IsNull() && !data.StartDate.IsUnknown() {
		parsedStartDate, err := gitlab.ParseISOTime(data.StartDate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to parse start_date",
				fmt.Sprintf("Failed to parse start_date: %s. %v", data.StartDate.ValueString(), err),
			)
			return
		}
		options.StartDate = &parsedStartDate
	}

	if !data.DueDate.IsNull() && !data.DueDate.IsUnknown() {
		parsedDueDate, err := gitlab.ParseISOTime(data.DueDate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to parse due_date",
				fmt.Sprintf("Failed to parse due_date: %s. %v", data.DueDate.ValueString(), err),
			)
			return
		}
		options.DueDate = &parsedDueDate
	}

	if !data.State.IsNull() && !data.State.IsUnknown() {
		options.StateEvent = gitlab.Ptr(milestoneStateToStateEvent[data.State.ValueString()])
	}

	tflog.Debug(ctx, "update project milestone", map[string]any{
		"project": project, "milestone": milestoneID,
	})
	milestone, _, err := r.client.Milestones.UpdateMilestone(project, milestoneID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update milestone: %s", err.Error()))
		return
	}

	r.milestoneToStateModel(project, milestone, data)

	// Log the update of the resource
	tflog.Debug(ctx, "updated a milestone", map[string]any{
		"project": project, "milestone": milestoneID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectMilestoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectMilestoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, rawMilestoneID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Delete. It should be '<project>:<milestone_id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	milestoneID, err := strconv.ParseInt(rawMilestoneID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid milestone ID provided, milestone ID should be an Int",
			fmt.Sprintf("Unable to convert milestone id to int: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "delete project milestone", map[string]any{
		"project": project, "milestone": milestoneID,
	})
	if _, err = r.client.Milestones.DeleteMilestone(project, milestoneID, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete milestone: %s", err.Error()),
		)
	}
}
