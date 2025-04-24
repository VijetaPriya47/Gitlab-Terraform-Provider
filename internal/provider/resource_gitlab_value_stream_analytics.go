package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabValueStreamAnalyticsResource{}
	_ resource.ResourceWithConfigure   = &gitlabValueStreamAnalyticsResource{}
	_ resource.ResourceWithImportState = &gitlabValueStreamAnalyticsResource{}
	_ resource.ResourceWithModifyPlan  = &gitlabValueStreamAnalyticsResource{}
)

func init() {
	registerResource(NewGitLabValueStreamAnalyticsResource)
}

func NewGitLabValueStreamAnalyticsResource() resource.Resource {
	return &gitlabValueStreamAnalyticsResource{}
}

type gitlabValueStreamAnalyticsResource struct {
	client *gitlab.Client
}

type gitlabValueStreamAnalyticsResourceModel struct {
	Id              types.String                           `tfsdk:"id"`
	Name            types.String                           `tfsdk:"name"`
	GroupFullPath   types.String                           `tfsdk:"group_full_path"`
	ProjectFullPath types.String                           `tfsdk:"project_full_path"`
	Stages          []gitlabValueStreamAnalyticsStageModel `tfsdk:"stages"`
}

type gitlabValueStreamAnalyticsStageModel struct {
	Id                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Custom               types.Bool   `tfsdk:"custom"`
	Hidden               types.Bool   `tfsdk:"hidden"`
	EndEventIdentifier   types.String `tfsdk:"end_event_identifier"`
	EndEventLabelId      types.String `tfsdk:"end_event_label_id"`
	StartEventIdentifier types.String `tfsdk:"start_event_identifier"`
	StartEventLabelId    types.String `tfsdk:"start_event_label_id"`
}

// Metadata returns the resource name
func (d *gitlabValueStreamAnalyticsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_value_stream_analytics"
}

func (r *gitlabValueStreamAnalyticsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	allowedEventLabels := []string{
		"CODE_STAGE_START", "ISSUE_CLOSED", "ISSUE_CREATED", "ISSUE_DEPLOYED_TO_PRODUCTION",
		"ISSUE_FIRST_ADDED_TO_BOARD", "ISSUE_FIRST_ADDED_TO_ITERATION", "ISSUE_FIRST_ASSIGNED_AT", "ISSUE_FIRST_ASSOCIATED_WITH_MILESTONE",
		"ISSUE_FIRST_MENTIONED_IN_COMMIT", "ISSUE_LABEL_ADDED", "ISSUE_LABEL_REMOVED", "ISSUE_LAST_EDITED", "ISSUE_STAGE_END", "MERGE_REQUEST_CLOSED",
		"MERGE_REQUEST_CREATED", "MERGE_REQUEST_FIRST_ASSIGNED_AT", "MERGE_REQUEST_FIRST_COMMIT_AT", "MERGE_REQUEST_FIRST_DEPLOYED_TO_PRODUCTION",
		"MERGE_REQUEST_LABEL_ADDED", "MERGE_REQUEST_LABEL_REMOVED", "MERGE_REQUEST_LAST_BUILD_FINISHED", "MERGE_REQUEST_LAST_BUILD_STARTED",
		"MERGE_REQUEST_LAST_EDITED", "MERGE_REQUEST_MERGED", "MERGE_REQUEST_REVIEWER_FIRST_ASSIGNED", "MERGE_REQUEST_PLAN_STAGE_START",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_value_stream_analytics`" + ` resource allows to manage the lifecycle of value stream analytics.

-> This resource requires a GitLab Enterprise instance with a Premium license to create custom value stream analytics.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#mutationvaluestreamcreate)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Terraform ID of the value stream in the format of `group:<group_full_path>:<id>` or `project:<project_full_path>:<id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the value stream",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group_full_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the group the value stream is created in. **One of `group_full_path` OR `project_full_path` is required.**",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("project_full_path")),
				},
			},
			"project_full_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the project the value stream is created in. **One of `group_full_path` OR `project_full_path` is required.**",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("group_full_path")),
				},
			},
			"stages": schema.ListNestedAttribute{
				MarkdownDescription: "Stages of the value stream",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the value stream stage.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the value stream stage.",
							Required:            true,
							Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
							PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
						},
						"custom": schema.BoolAttribute{
							MarkdownDescription: "Boolean whether the stage is customized. If false, it assigns a built-in default stage by name.",
							Optional:            true,
							Validators:          []validator.Bool{boolvalidator.All()},
							PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
						},
						"hidden": schema.BoolAttribute{
							MarkdownDescription: "Boolean whether the stage is hidden, GitLab provided default stages are hidden by default.",
							Optional:            true,
							Validators:          []validator.Bool{boolvalidator.All()},
							PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
						},
						"end_event_identifier": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf(`End event identifier. Valid values are: %s`, utils.RenderValueListForDocs(allowedEventLabels)),

							Optional:      true,
							Computed:      true,
							Validators:    []validator.String{stringvalidator.OneOf(allowedEventLabels...)},
							PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						},
						"end_event_label_id": schema.StringAttribute{
							MarkdownDescription: "Label ID associated with the end event identifier. In the format of `gid://gitlab/GroupLabel/<id>` or `gid://gitlab/ProjectLabel/<id>`",
							Optional:            true,
							Computed:            true,
							Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
							PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
						},
						"start_event_identifier": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf(`Start event identifier. Valid values are: %s`, utils.RenderValueListForDocs(allowedEventLabels)),
							Optional:            true,
							Computed:            true,
							Validators:          []validator.String{stringvalidator.OneOf(allowedEventLabels...)},
							PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
						},
						"start_event_label_id": schema.StringAttribute{
							MarkdownDescription: "Label ID associated with the start event identifier. In the format of `gid://gitlab/GroupLabel/<id>` or `gid://gitlab/ProjectLabel/<id>`",
							Optional:            true,
							Computed:            true,
							Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
							PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabValueStreamAnalyticsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Use the `ModifyPlan` to validate conditions not configurable within the schema.
func (r *gitlabValueStreamAnalyticsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Retrieve the plan data to start with
	var planData *gitlabValueStreamAnalyticsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if planData == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no further validation is needed.")
		return
	}

	// Validate that group_full_path or project_full_path is defined.
	if (planData.GroupFullPath.IsNull() || planData.GroupFullPath.ValueString() == "") && (planData.ProjectFullPath.IsNull() || planData.ProjectFullPath.ValueString() == "") {
		resp.Diagnostics.AddAttributeError(path.Root("group_full_path"), "Missing Attribute", "Either `group_full_path` or `project_full_path` is required")
	}

	// Based on each stage's start and end event identifiers, validate whether a label id is provided or not.
	for i, v := range planData.Stages {

		// If StartEventIdentifier uses labels, but no label ID is provided throw an error
		if strings.Contains(v.StartEventIdentifier.ValueString(), "LABEL") && (v.StartEventLabelId.IsNull() || v.StartEventLabelId.ValueString() == "") {
			resp.Diagnostics.AddAttributeError(path.Root("stages").AtListIndex(i), "Missing Attribute", fmt.Sprintf("`start_event_label_id` is required when `start_event_identifier` is %s", v.StartEventIdentifier.ValueString()))
		}

		// If StartEventIdentifier doesn't use labels, but a label ID is provided throw an error
		if !strings.Contains(v.StartEventIdentifier.ValueString(), "LABEL") && !v.StartEventLabelId.IsNull() && v.StartEventLabelId.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(path.Root("stages").AtListIndex(i), "Unexpected Attribute", fmt.Sprintf("`start_event_label_id` is not allowed when `start_event_identifier` is %s", v.StartEventIdentifier.ValueString()))
		}

		// If EndEventIdentifier uses labels, but no label ID is provided throw an error
		if strings.Contains(v.EndEventIdentifier.ValueString(), "LABEL") && (v.EndEventLabelId.IsNull() || v.EndEventLabelId.ValueString() == "") {
			resp.Diagnostics.AddAttributeError(path.Root("stages").AtListIndex(i), "Missing Attribute", fmt.Sprintf("`end_event_label_id` is required when `end_event_identifier` is %s", v.EndEventIdentifier.ValueString()))
		}

		// If EndEventIdentifier doesn't use labels, but a label ID is provided throw an error
		if !strings.Contains(v.EndEventIdentifier.ValueString(), "LABEL") && !v.EndEventLabelId.IsNull() && v.EndEventLabelId.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(path.Root("stages").AtListIndex(i), "Unexpected Attribute", fmt.Sprintf("`end_event_label_id` is not allowed when `end_event_identifier` is %s", v.EndEventIdentifier.ValueString()))
		}
	}
}

func (r *gitlabValueStreamAnalyticsResource) valueStreamToStateModel(response *valueStream, data *gitlabValueStreamAnalyticsResourceModel, fullPathIsGroupPath bool) {
	data.Name = types.StringValue(response.Name)

	if fullPathIsGroupPath {
		groupString := "group"
		data.GroupFullPath = types.StringValue(response.Namespace.FullPath)
		data.Id = types.StringValue(utils.BuildThreePartID(&groupString, &response.Namespace.FullPath, &response.Id))
	} else {
		projectString := "project"
		data.ProjectFullPath = types.StringValue(response.Namespace.FullPath)
		data.Id = types.StringValue(utils.BuildThreePartID(&projectString, &response.Namespace.FullPath, &response.Id))
	}

	stages := make([]gitlabValueStreamAnalyticsStageModel, 0, len(response.Stages))
	for _, v := range response.Stages {
		var stage gitlabValueStreamAnalyticsStageModel
		stage.Id = types.StringValue(v.Id)
		stage.Name = types.StringValue(v.Name)
		stage.Custom = types.BoolValue(v.Custom)
		stage.Hidden = types.BoolValue(v.Hidden)
		stage.EndEventIdentifier = types.StringValue(v.EndEventIdentifier)
		stage.EndEventLabelId = types.StringValue(v.EndEventLabel.Id)
		stage.StartEventIdentifier = types.StringValue(v.StartEventIdentifier)
		stage.StartEventLabelId = types.StringValue(v.StartEventLabel.Id)

		stages = append(stages, stage)
	}
	data.Stages = stages
}

func (r *gitlabValueStreamAnalyticsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabValueStreamAnalyticsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	fullPathType, fullPath, id, err := utils.ParseThreePartID(data.Id.ValueString())
	fullPathType = strings.ToLower(fullPathType)
	if err != nil {
		resp.Diagnostics.AddError("Value Stream Analytics", "Errors found while parsing three part id")
		return
	}

	if fullPathType != "group" && fullPathType != "project" {
		resp.Diagnostics.AddError("Value Stream Analytics", "Error validating fullPathType, must be either 'group' or 'project'")
		return
	}

	// read all information for refresh from resource id
	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			query {
				%s(fullPath: "%s") {
					valueStreams(id: "%s") {
						nodes {
							id
							name
							namespace {
								fullPath
							}
							stages {
								id
								name
								custom
								hidden
								startEventIdentifier
								startEventLabel {
									id
								}
								endEventIdentifier
								endEventLabel {
									id
								}
							}
						}
					}
				}
			}`, fullPathType, fullPath, id),
	}
	tflog.Debug(ctx, "executing GraphQL Query to retrieve current value stream analytics on project", map[string]any{
		"query": query.Query,
	})

	if fullPathType == "group" {
		var response groupValueStreamResponse
		if _, err := api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
			if api.Is404(err) {
				tflog.Debug(ctx, "value stream analytics does not exist, removing from state", map[string]any{
					"full_path": fullPath,
				})
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read value stream analytics details: %s", err.Error()))
			return
		}

		if len(response.Data.Group.ValueStreams.Nodes) != 1 {
			resp.Diagnostics.AddError("Value Stream Analytics Error", fmt.Sprintf("Expected only one value stream analytics, found %d", len(response.Data.Group.ValueStreams.Nodes)))
			return
		}
		r.valueStreamToStateModel(&response.Data.Group.ValueStreams.Nodes[0], data, true)

	} else {
		var response projectValueStreamResponse
		if _, err := api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
			if api.Is404(err) {
				tflog.Debug(ctx, "value stream analytics does not exist, removing from state", map[string]any{
					"full_path": fullPath,
				})
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read value stream analytics details: %s", err.Error()))
			return
		}

		if len(response.Data.Project.ValueStreams.Nodes) != 1 {
			resp.Diagnostics.AddError("Value Stream Analytics Error", fmt.Sprintf("Expected only one value stream analytics, found %d", len(response.Data.Project.ValueStreams.Nodes)))
			return
		}
		r.valueStreamToStateModel(&response.Data.Project.ValueStreams.Nodes[0], data, false)

	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabValueStreamAnalyticsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabValueStreamAnalyticsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	stages := stagesStringBuilder(data.Stages)

	var fullPath string
	var fullPathIsGroupPath bool
	if !data.GroupFullPath.IsNull() && data.GroupFullPath.ValueString() != "" {
		fullPath = data.GroupFullPath.ValueString()
		fullPathIsGroupPath = true
	} else {
		fullPath = data.ProjectFullPath.ValueString()
		fullPathIsGroupPath = false
	}

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				valueStreamCreate(
					input:{
						stages:[%s]
						name: "%s"
						namespacePath: "%s"
					}
				)
			{
				valueStream {
					id
					name
					namespace {
						fullPath
					}
					stages{
						id
						name
						custom
						hidden
						startEventIdentifier
						startEventLabel {
							id
						}
						endEventIdentifier
						endEventLabel {
							id
						}
					}
				}
				errors
			}
			}`, stages, name, fullPath),
	}

	var response createValueStreamResponse
	if _, err := api.SendGraphQLRequest(ctx, r.client, query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create value stream analytics: %s", err.Error()))
		return
	}

	// check response for errors
	var allerr string
	if len(response.Errors) > 0 {
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
	}
	if len(response.Data.ValueStreamCreate.Errors) > 0 {
		for i, err := range response.Data.ValueStreamCreate.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
	}
	if len(allerr) > 0 {
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}

	// persist API response in state model
	r.valueStreamToStateModel(&response.Data.ValueStreamCreate.ValueStream, data, fullPathIsGroupPath)

	// Log the creation of the resource
	tflog.Debug(ctx, "created a value stream analytics", map[string]any{
		"id": data.Id.ValueString(), "fullPath": fullPath,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabValueStreamAnalyticsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

func (r *gitlabValueStreamAnalyticsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabValueStreamAnalyticsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, _, id, err := utils.ParseThreePartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error occured while parsing ID", fmt.Sprintf("Unable to parse ID: %s", err.Error()))
		return
	}

	query := api.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				valueStreamDestroy(
					input: {
						id: "%s"
					}
				)
				{
					errors
				}
			}`, id),
	}

	tflog.Debug(ctx, "executing GraphQL Query to update value stream analytics", map[string]any{
		"query": query.Query,
	})

	if _, err := api.SendGraphQLRequest(ctx, r.client, query, nil); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete value stream analytics: %s", err.Error()))
		return
	}
}

func (r *gitlabValueStreamAnalyticsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func stagesStringBuilder(stages []gitlabValueStreamAnalyticsStageModel) string {
	var stageStrings []string

	for _, stage := range stages {
		// Build each attribute in the stage as a string formatted for GraphQL query
		var formattedAttributes []string
		formattedAttributes = append(formattedAttributes, fmt.Sprintf("name: %q", stage.Name.ValueString()))
		formattedAttributes = append(formattedAttributes, fmt.Sprintf("custom: %t", stage.Custom.ValueBool()))
		formattedAttributes = append(formattedAttributes, fmt.Sprintf("hidden: %t", stage.Hidden.ValueBool()))
		if !stage.StartEventIdentifier.IsNull() && stage.StartEventIdentifier.ValueString() != "" {
			formattedAttributes = append(formattedAttributes, fmt.Sprintf("startEventIdentifier: %s", stage.StartEventIdentifier.ValueString()))
		}
		if !stage.StartEventLabelId.IsNull() && stage.StartEventLabelId.ValueString() != "" {
			formattedAttributes = append(formattedAttributes, fmt.Sprintf("startEventLabelId: %q", stage.StartEventLabelId.ValueString()))
		}
		if !stage.EndEventIdentifier.IsNull() && stage.EndEventIdentifier.ValueString() != "" {
			formattedAttributes = append(formattedAttributes, fmt.Sprintf("endEventIdentifier: %s", stage.EndEventIdentifier.ValueString()))
		}
		if !stage.EndEventLabelId.IsNull() && stage.EndEventLabelId.ValueString() != "" {
			formattedAttributes = append(formattedAttributes, fmt.Sprintf("endEventLabelId: %q", stage.EndEventLabelId.ValueString()))
		}

		// Build GraphQL query string from individual attributes for a single stage
		var formattedStageString string
		for _, attr := range formattedAttributes {
			formattedStageString += fmt.Sprintf("%s,", attr)
		}
		// trim last comma
		formattedStageString = strings.TrimSuffix(formattedStageString, ",")

		stageStrings = append(stageStrings, fmt.Sprintf("{%s}", formattedStageString))
	}

	return (strings.Join(stageStrings, ","))
}

type valueStream struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Namespace struct {
		FullPath string `json:"fullPath"`
	} `json:"namespace"`
	Stages []struct {
		Id                 string `json:"id"`
		Name               string `json:"name"`
		Custom             bool   `json:"custom"`
		Hidden             bool   `json:"hidden"`
		EndEventIdentifier string `json:"endEventIdentifier"`
		EndEventLabel      struct {
			Id string `json:"id"`
		} `json:"endEventLabel"`
		StartEventIdentifier string `json:"startEventIdentifier"`
		StartEventLabel      struct {
			Id string `json:"id"`
		} `json:"startEventLabel"`
	}
}

type projectValueStreamResponse struct {
	Data struct {
		Project struct {
			ValueStreams struct {
				Nodes []valueStream `json:"nodes"`
			} `json:"valueStreams"`
		} `json:"project"`
	} `json:"data"`
}

type groupValueStreamResponse struct {
	Data struct {
		Group struct {
			ValueStreams struct {
				Nodes []valueStream `json:"nodes"`
			} `json:"valueStreams"`
		} `json:"group"`
	} `json:"data"`
}

type createValueStreamResponse struct {
	Data struct {
		ValueStreamCreate struct {
			ValueStream valueStream `json:"valueStream"`
			Errors      []string    `json:"errors"`
		} `json:"valueStreamCreate"`
	} `json:"data"`
	Errors []struct {
		Message   string `json:"message"`
		Locations []struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"locations"`
		Path []string `json:"path"`
	} `json:"errors"`
}
