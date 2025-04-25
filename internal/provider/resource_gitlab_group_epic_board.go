package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabGroupEpicBoardResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupEpicBoardResource{}
	_ resource.ResourceWithImportState = &gitlabGroupEpicBoardResource{}
)

func init() {
	registerResource(NewGitLabGroupEpicBoardResource)
}

func NewGitLabGroupEpicBoardResource() resource.Resource {
	return &gitlabGroupEpicBoardResource{}
}

type gitlabGroupEpicBoardResource struct {
	client *gitlab.Client
}

type gitlabGroupEpicBoardResourceModel struct {
	Id    types.String                    `tfsdk:"id"`
	Group types.String                    `tfsdk:"group"`
	Name  types.String                    `tfsdk:"name"`
	Lists []gitlabGroupEpicBoardListModel `tfsdk:"lists"`
}

type gitlabGroupEpicBoardListModel struct {
	Id       types.Int64 `tfsdk:"id"`
	Position types.Int64 `tfsdk:"position"`
	LabelId  types.Int64 `tfsdk:"label_id"`
}

func (r *gitlabGroupEpicBoardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_epic_board"
}

func (r *gitlabGroupEpicBoardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_epic_board`" + ` resource allows to manage the lifecycle of a epic board in a group.

~> Multiple epic boards on one group requires a GitLab Premium or above License.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/group_boards/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group-id>:<epic-board-id>`.",
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
			},
		},
		Blocks: map[string]schema.Block{
			"lists": schema.SetNestedBlock{
				MarkdownDescription: "The list of epic board lists.",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace(), setplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the list.",
							Computed:            true,
							PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						},
						"position": schema.Int64Attribute{
							MarkdownDescription: "The position of the list within the board. The position for the list is sed on the its position in the `lists` array.",
							Computed:            true,
							PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						},
						"label_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the label the list should be scoped to.",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupEpicBoardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type EpicBoardCreateResponse struct {
	Data struct {
		EpicBoardCreate struct {
			EpicBoard GraphQLEpicBoard `json:"epicBoard"`
		} `json:"epicBoardCreate"`
	} `json:"data"`
	Errors []string `json:"errors"`
}

type GraphQLEpicBoard struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Labels struct {
		Nodes []struct {
			Id string `json:"id"`
		} `json:"nodes"`
	} `json:"labels"`
}

type Label struct {
	ID string `json:"id"`
}

type EpicBoardUpdateResponse struct {
	Data struct {
		EpicBoardUpdate struct {
			EpicBoard GraphQLEpicBoard `json:"epicBoard"`
		} `json:"epicBoardUpdate"`
	} `json:"data"`
	Errors []string `json:"errors"`
}

type EpicBoardListCreateInput struct {
	BoardID string `json:"boardId"`
	LabelID string `json:"labelId"`
}

type EpicBoardListCreateMutation struct {
	ClientMutationID string   `json:"clientMutationId"`
	Errors           []string `json:"errors"`
}

func (r *gitlabGroupEpicBoardResource) groupEpicBoardToStateModel(groupID string, groupEpicBoard *gitlab.GroupEpicBoard, data *gitlabGroupEpicBoardResourceModel) {
	data.Group = types.StringValue(groupID)
	data.Name = types.StringValue(groupEpicBoard.Name)

	var filteredLists []*gitlab.BoardList
	// Filter the lists based on the condition label_id > 0
	for _, item := range groupEpicBoard.Lists {
		if item.Label != nil {
			filteredLists = append(filteredLists, item)
		}
	}
	listsData := make([]gitlabGroupEpicBoardListModel, len(filteredLists))
	for i, v := range filteredLists {
		listData := gitlabGroupEpicBoardListModel{}
		listData.Id = types.Int64Value(int64(v.ID))
		listData.Position = types.Int64Value(int64(v.Position))
		if v.Label != nil {
			listData.LabelId = types.Int64Value(int64(v.Label.ID))
		}

		listsData[i] = listData
	}
	data.Lists = listsData
}

func (r *gitlabGroupEpicBoardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupEpicBoardResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

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

	// Read board
	groupEpicBoard, _, err := r.client.GroupEpicBoards.GetGroupEpicBoard(groupID, boardId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "group epic board does not exist, removing from state", map[string]any{
				"group": groupID, "environment": boardID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read gitlab group epic board details: %s", err.Error()))
		return
	}

	// persist API response in state model
	data.Id = types.StringValue(utils.BuildTwoPartID(&groupID, &boardID))
	r.groupEpicBoardToStateModel(groupID, groupEpicBoard, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupEpicBoardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupEpicBoardResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

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
			"Internal provider error",
			fmt.Sprintf("Unable to convert board id to int: %s", err.Error()),
		)
		return
	}

	boardName := data.Name.ValueString()

	labels := make([]string, len(data.Lists))
	for i, v := range data.Lists {
		labels[i] = fmt.Sprintf("gid://gitlab/GroupLabel/%s", v.LabelId.String())
	}
	labels_str := fmt.Sprintf(`["%s"]`, strings.Join(labels, `","`))

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				epicBoardUpdate(
					input: {
						id: "gid://gitlab/Boards::EpicBoard/%d",
						name: "%s",
						labels: %s
					}
				) {
					epicBoard {
						id,
						name,
						labels {
							nodes {
								id
							}
						}
					}
				}
			}`, boardId, boardName, labels_str),
	}
	tflog.Debug(ctx, "executing GraphQL Query to create epic board", map[string]any{
		"query": query.Query,
	})

	var response EpicBoardUpdateResponse
	if _, err = r.client.GraphQL.Do(ctx, query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", fmt.Sprintf("Unable to update epic board: %s from query %s", err.Error(), query.Query))
		return
	}
	if len(response.Errors) > 0 {
		allerr := fmt.Sprintf("From update response %v\n", response)
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Code: %s\n", i, err)
		}
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}

	parts := strings.Split(response.Data.EpicBoardUpdate.EpicBoard.Id, "/")
	epicBoardId := parts[len(parts)-1]
	EpicBoardId, err := strconv.Atoi(epicBoardId)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update epic: %s - call response %v Query %s",
			"EpicBoard ID not found in GraphQL response", response, query.Query))
		return
	}

	var groupEpicBoard *gitlab.GroupEpicBoard
	groupEpicBoard, _, err = r.client.GroupEpicBoards.GetGroupEpicBoard(groupID, EpicBoardId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "group epic board does not exist, removing from state", map[string]any{
				"group": groupID, "environment": EpicBoardId,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read gitlab group epic board details: %s", err.Error()))
		return
	}

	data.Id = types.StringValue(utils.BuildTwoPartID(&groupID, &boardID))
	r.groupEpicBoardToStateModel(groupID, groupEpicBoard, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "updated a group epic board", map[string]any{
		"group": groupID, "board": groupEpicBoard.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabGroupEpicBoardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupEpicBoardResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, boardID, err := utils.ParseTwoPartID(data.Id.ValueString())
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
			"Internal provider error",
			fmt.Sprintf("Unable to convert board id to int: %s", err.Error()),
		)
		return
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				destroyEpicBoard(
					input: {
						id: "gid://gitlab/Boards::EpicBoard/%d",
					}
				) {
					epicBoard {
						id,
						name,
						labels {
							nodes {
								id
							}
						}
					}
					errors
				}
			}`, boardId),
	}

	tflog.Debug(ctx, "executing GraphQL Query to delete epic board", map[string]any{
		"query": query.Query,
	})

	var destroyResp map[string]any
	if _, err = r.client.GraphQL.Do(ctx, query, &destroyResp); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete epic board: %s from query %s", err.Error(), query.Query))
		return
	}
	if destroyResp["errors"] != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete epic board: %v from query %s", destroyResp["errors"], query.Query))
	}
}

func (r *gitlabGroupEpicBoardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabGroupEpicBoardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupEpicBoardResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	// read all information for refresh from resource id
	groupID := data.Group.ValueString()
	boardName := data.Name.ValueString()

	// we always resolve the group to gather the full path for subsequent GraphQL API calls which ALWAYS
	// require the full path ...
	group, _, err := r.client.Groups.GetGroup(groupID, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "group does not exist, removing resource from state", map[string]any{
				"group": groupID,
			})
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read group details: %s", err.Error()))
		return
	}

	labels := make([]Label, len(data.Lists))
	for i, v := range data.Lists {
		labels[i].ID = fmt.Sprintf("gid://gitlab/GroupLabel/%s", v.LabelId.String())
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				epicBoardCreate(
					input: {
						groupPath: "%s",
						name: "%s"
					}
				) {
					epicBoard {
						id,
						name,
						labels {
							nodes {
								id
							}
						}
					}
				}
			}`, group.FullPath, boardName),
	}
	tflog.Debug(ctx, "executing GraphQL Query to create epic board", map[string]any{
		"query": query.Query,
	})

	var response EpicBoardCreateResponse
	if _, err = r.client.GraphQL.Do(ctx, query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", fmt.Sprintf("Unable to create epic board: %s from query %s", err.Error(), query.Query))
		return
	}
	if len(response.Errors) > 0 {
		allerr := fmt.Sprintf("From create response %v\n", response)
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Code: %s\n", i, err)
		}
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}

	parts := strings.Split(response.Data.EpicBoardCreate.EpicBoard.Id, "/")
	epicBoardId := parts[len(parts)-1]
	EpicBoardId, err := strconv.Atoi(epicBoardId)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred",
			fmt.Sprintf("Unable to create epic board: %s - ID %s, Name %s from resp %v with query %s",
				"EpicBoard ID not found in GraphQL response",
				response.Data.EpicBoardCreate.EpicBoard.Id,
				response.Data.EpicBoardCreate.EpicBoard.Name,
				response,
				query.Query))
		return
	}

	// Create the lists on the epic board
	boardID := fmt.Sprintf("%d", EpicBoardId)

	for _, v := range labels {
		query = gitlab.GraphQLQuery{
			Query: fmt.Sprintf(`mutation {
				epicBoardListCreate(
				  input: {
					boardId: "gid://gitlab/Boards::EpicBoard/%s",
					labelId: "%s"
				  }
				) {
				  list {
					id
  					position
					label {
						id
					}
				  }
				  errors
				}
			}`, boardID, v.ID),
		}
		var listResp map[string]any
		if _, err = r.client.GraphQL.Do(ctx, query, &listResp); err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create epic board list: %s from query %s", err.Error(), query.Query))
		}
		if listResp["errors"] != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create epic board list: %v from query %s", listResp["errors"], query.Query))
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	groupEpicBoard, _, err := r.client.GroupEpicBoards.GetGroupEpicBoard(group.ID, EpicBoardId, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "group epic board does not exist, removing from state", map[string]any{
				"group": groupID, "environment": EpicBoardId,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read gitlab group epic board details: %s", err.Error()))
		return
	}

	data.Id = types.StringValue(utils.BuildTwoPartID(&groupID, &boardID))
	r.groupEpicBoardToStateModel(groupID, groupEpicBoard, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "created a group epic board", map[string]any{
		"group": groupID, "board": groupEpicBoard.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
