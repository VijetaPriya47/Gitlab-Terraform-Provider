package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ resource.Resource = &gitlabGroupIssueBoardResource{}
var _ resource.ResourceWithConfigure = &gitlabGroupIssueBoardResource{}
var _ resource.ResourceWithImportState = &gitlabGroupIssueBoardResource{}

func init() {
	registerResource(NewGitLabGroupIssueBoardResource)
}

func NewGitLabGroupIssueBoardResource() resource.Resource {
	return &gitlabGroupIssueBoardResource{}
}

type gitlabGroupIssueBoardResource struct {
	client *gitlab.Client // This is required for making calls to GitLab later
}

type gitlabGroupIssueBoardResourceModel struct {
	Id          types.String                     `tfsdk:"id"`
	Group       types.String                     `tfsdk:"group"`
	Name        types.String                     `tfsdk:"name"`
	MilestoneId types.Int64                      `tfsdk:"milestone_id"`
	Labels      types.Set                        `tfsdk:"labels"`
	Lists       []gitlabGroupIssueBoardListModel `tfsdk:"lists"`
}

type gitlabGroupIssueBoardListModel struct {
	Id       types.Int64 `tfsdk:"id"`
	LabelId  types.Int64 `tfsdk:"label_id"`
	Position types.Int64 `tfsdk:"position"`
}

func (r *gitlabGroupIssueBoardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_issue_board"
}

func (r *gitlabGroupIssueBoardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_issue_board`" + ` resource allows to manage the lifecycle of a issue board in a group.

~> Multiple issue boards on one group requires a GitLab Premium or above License.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/group_boards.html)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group-id>:<issue-board-id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the group owned by the authenticated user.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the board.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"milestone_id": schema.Int64Attribute{
				MarkdownDescription: "The milestone the board should be scoped to.",
				Optional:            true,
			},
			"labels": schema.SetAttribute{
				MarkdownDescription: "The list of label names which the board should be scoped to.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"lists": schema.SetNestedBlock{
				MarkdownDescription: "The list of issue board lists.",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace(), setplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the list.",
							Computed:            true,
						},
						"label_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the label the list should be scoped to.",
							Optional:            true,
						},
						"position": schema.Int64Attribute{
							MarkdownDescription: "The explicit position of the list within the board, zero based.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupIssueBoardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupIssueBoardResource) groupIssueBoardToStateModel(groupID string, groupIssueBoard *gitlab.GroupIssueBoard, data *gitlabGroupIssueBoardResourceModel) {
	data.Group = types.StringValue(groupID)
	data.Name = types.StringValue(groupIssueBoard.Name)
	if groupIssueBoard.Milestone != nil {
		data.MilestoneId = types.Int64Value(int64(groupIssueBoard.Milestone.ID))
	}

	listsData := make([]gitlabGroupIssueBoardListModel, len(groupIssueBoard.Lists))
	for i, v := range groupIssueBoard.Lists {
		listData := gitlabGroupIssueBoardListModel{}
		listData.Id = types.Int64Value(int64(v.ID))
		listData.LabelId = types.Int64Value(int64(v.Label.ID))
		listData.Position = types.Int64Value(int64(v.Position))

		listsData[i] = listData
	}
	data.Lists = listsData
}

func (r *gitlabGroupIssueBoardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupIssueBoardResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	groupID, boardID, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<group-id>:<board-id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	boardId, err := strconv.Atoi(boardID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid board ID provided, board ID should be an Int",
			fmt.Sprintf("Unable to convert board id to int: %s", err.Error()),
		)
		return
	}

	// Read environment protection
	groupIssueBoard, _, err := r.client.GroupIssueBoards.GetGroupIssueBoard(groupID, boardId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "group issue board does not exist, removing from state", map[string]interface{}{
				"group": groupID, "environment": boardID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read gitlab group issue board details: %s", err.Error()))
		return
	}

	// persist API response in state model
	data.Id = types.StringValue(utils.BuildTwoPartID(&groupID, &boardID))
	r.groupIssueBoardToStateModel(groupID, groupIssueBoard, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupIssueBoardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupIssueBoardResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupID, boardID, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Update. It should be '<group-id>:<board-id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	boardId, err := strconv.Atoi(boardID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Internal provider error",
			fmt.Sprintf("Unable to convert board id to int: %s", err.Error()),
		)
		return
	}

	// local copies of plan arguments
	// read all information for refresh from resource id
	boardName := data.Name.ValueString()

	optionsUpdate := &gitlab.UpdateGroupIssueBoardOptions{
		Name: gitlab.Ptr(boardName),
	}

	if !data.MilestoneId.IsNull() && !data.MilestoneId.IsUnknown() {
		optionsUpdate.MilestoneID = gitlab.Ptr(int(data.MilestoneId.ValueInt64()))
	}

	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		// convert the Set to a []string and pass it in
		var labels []string
		data.Labels.ElementsAs(ctx, &labels, true)
		gitlabLabels := gitlab.Labels(labels)
		optionsUpdate.Labels = &gitlabLabels
	}

	issueBoard, _, err := r.client.GroupIssueBoards.UpdateIssueBoard(groupID, boardId, optionsUpdate, gitlab.WithContext(ctx))
	if err != nil {
		// persist API response in state model
		data.Id = types.StringValue(fmt.Sprintf("%s:%d", groupID, issueBoard.ID))
		r.groupIssueBoardToStateModel(groupID, issueBoard, data)
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The group issue board feature is not available on this group. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update issue board: %s", err.Error()))
		return
	}

	for _, d := range issueBoard.Lists {
		_, err := r.client.GroupIssueBoards.DeleteGroupIssueBoardList(groupID, issueBoard.ID, d.ID, gitlab.WithContext(ctx))
		if err != nil {
			// persist API response in state model
			data.Id = types.StringValue(fmt.Sprintf("%s:%d", groupID, issueBoard.ID))
			r.groupIssueBoardToStateModel(groupID, issueBoard, data)
			// Save updated data into Terraform state
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError("GitLab API error occurred", "failed to delete list ")
		}

	}
	listsData := make([]*gitlab.BoardList, len(data.Lists))
	// Sort data.Lists based on list Position
	sort.Slice(data.Lists, func(i, j int) bool {
		labelPosI := data.Lists[i].Position
		labelPosJ := data.Lists[j].Position
		// Handle nil values
		if (labelPosI.IsNull() || labelPosI.IsUnknown()) && (labelPosJ.IsNull() || labelPosJ.IsUnknown()) {
			return false // Treat two nils as equal
		} else if labelPosI.IsNull() || labelPosI.IsUnknown() {
			return true // Nil should come after non-nil
		} else if labelPosJ.IsNull() || labelPosJ.IsUnknown() {
			return false // Non-nil should come before nil
		}
		return labelPosI.ValueInt64() < labelPosJ.ValueInt64()
	})
	for i, v := range data.Lists {
		listOptions := &gitlab.CreateGroupIssueBoardListOptions{}
		listOptions.LabelID = gitlab.Ptr(int(v.LabelId.ValueInt64()))
		issueBoardList, _, err := r.client.GroupIssueBoards.CreateGroupIssueBoardList(groupID, issueBoard.ID, listOptions, gitlab.WithContext(ctx))
		if err != nil {
			if api.Is404(err) {
				resp.Diagnostics.AddError(
					"GitLab Feature not available",
					fmt.Sprintf("The group issue board feature is not available on this group. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
				)
				return
			}
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update issue board lists. Issue board may still have been updated: %s", err.Error()))
			return
		}
		listsData[i] = issueBoardList
	}
	issueBoard.Lists = listsData

	// persist API response in state model
	data.Id = types.StringValue(fmt.Sprintf("%s:%d", groupID, issueBoard.ID))
	r.groupIssueBoardToStateModel(groupID, issueBoard, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "updated a group issue board", map[string]interface{}{
		"group": groupID, "board": issueBoard.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupIssueBoardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupIssueBoardResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	groupID, boardID, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Delete. It should be '<group-id>:<board-id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	boardId, err := strconv.Atoi(boardID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Internal provider error",
			fmt.Sprintf("Unable to convert board id to int: %s", err.Error()),
		)
		return
	}

	if _, err = r.client.GroupIssueBoards.DeleteIssueBoard(groupID, boardId, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete group issue board: %s", err.Error()),
		)
	}
}

func (r *gitlabGroupIssueBoardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

func (r *gitlabGroupIssueBoardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data *gitlabGroupIssueBoardResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	// read all information for refresh from resource id
	groupID := data.Group.ValueString()
	boardName := data.Name.ValueString()

	// configure GitLab API call
	options := &gitlab.CreateGroupIssueBoardOptions{
		Name: gitlab.Ptr(boardName),
	}

	optionsUpdate := &gitlab.UpdateGroupIssueBoardOptions{
		Name: gitlab.Ptr(boardName),
	}

	if !data.MilestoneId.IsNull() && !data.MilestoneId.IsUnknown() {
		optionsUpdate.MilestoneID = gitlab.Ptr(int(data.MilestoneId.ValueInt64()))
	}

	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		// convert the Set to a []string and pass it in
		var labels []string
		data.Labels.ElementsAs(ctx, &labels, true)
		gitlabLabels := gitlab.Labels(labels)
		optionsUpdate.Labels = &gitlabLabels
	}

	issueBoard, _, err := r.client.GroupIssueBoards.CreateGroupIssueBoard(groupID, options, gitlab.WithContext(ctx))
	if err != nil {
		// persist API response in state model
		data.Id = types.StringValue(fmt.Sprintf("%s:%d", groupID, issueBoard.ID))
		r.groupIssueBoardToStateModel(groupID, issueBoard, data)
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The group issue board feature is not available on this group. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create issue board: %s", err.Error()))
		return
	}

	issueBoard, _, err = r.client.GroupIssueBoards.UpdateIssueBoard(groupID, issueBoard.ID, optionsUpdate, gitlab.WithContext(ctx))
	if err != nil {
		// persist API response in state model
		data.Id = types.StringValue(fmt.Sprintf("%s:%d", groupID, issueBoard.ID))
		r.groupIssueBoardToStateModel(groupID, issueBoard, data)
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The group issue board feature is not available on this group. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create issue board: %s", err.Error()))
		return
	}

	listsData := make([]*gitlab.BoardList, len(data.Lists))
	// Sort data.Lists based on list Position
	sort.Slice(data.Lists, func(i, j int) bool {
		labelPosI := data.Lists[i].Position
		labelPosJ := data.Lists[j].Position
		// Handle nil values
		if (labelPosI.IsNull() || labelPosI.IsUnknown()) && (labelPosJ.IsNull() || labelPosJ.IsUnknown()) {
			return false // Treat two nils as equal
		} else if labelPosI.IsNull() || labelPosI.IsUnknown() {
			return true // Nil should come after non-nil
		} else if labelPosJ.IsNull() || labelPosJ.IsUnknown() {
			return false // Non-nil should come before nil
		}
		return labelPosI.ValueInt64() < labelPosJ.ValueInt64()
	})
	for i, v := range data.Lists {
		listOptions := &gitlab.CreateGroupIssueBoardListOptions{}
		listOptions.LabelID = gitlab.Ptr(int(v.LabelId.ValueInt64()))
		issueBoardList, _, err := r.client.GroupIssueBoards.CreateGroupIssueBoardList(groupID, issueBoard.ID, listOptions, gitlab.WithContext(ctx))
		if err != nil {
			if api.Is404(err) {
				resp.Diagnostics.AddError(
					"GitLab Feature not available",
					fmt.Sprintf("The group issue board feature is not available on this group. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
				)
				return
			}
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create issue board: %s", err.Error()))
			return
		}
		listsData[i] = issueBoardList
	}
	issueBoard.Lists = listsData

	// persist API response in state model
	boardID := fmt.Sprintf("%d", issueBoard.ID)
	data.Id = types.StringValue(utils.BuildTwoPartID(&groupID, &boardID))
	r.groupIssueBoardToStateModel(groupID, issueBoard, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "created a group issue board", map[string]interface{}{
		"group": groupID, "board": issueBoard.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
