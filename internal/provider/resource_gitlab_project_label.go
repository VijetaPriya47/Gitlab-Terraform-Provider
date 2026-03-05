package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                 = &gitlabProjectLabelResource{}
	_ resource.ResourceWithConfigure    = &gitlabProjectLabelResource{}
	_ resource.ResourceWithImportState  = &gitlabProjectLabelResource{}
	_ resource.ResourceWithUpgradeState = &gitlabProjectLabelResource{}
	_ resource.ResourceWithMoveState    = &gitlabProjectLabelResource{}
)

func init() {
	registerResource(NewGitLabProjectLabelResource)

	// Remove in 19.0
	registerResource(NewGitLabLabelResource)
}

func NewGitLabProjectLabelResource() resource.Resource {
	return &gitlabProjectLabelResource{
		ResourceName: "_project_label",
		ResourceDescription: `The ` + "`" + `gitlab_project_label` + "`" + ` resource manages the lifecycle of a project label.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/labels/#get-a-single-project-label)`,
	}
}

// Remove in 19.0
func NewGitLabLabelResource() resource.Resource {
	return &gitlabProjectLabelResource{
		ResourceName: "_label",
		ResourceDescription: `The ` + "`" + `gitlab_label` + "`" + ` resource manages the lifecycle of a project label.

~> This resource is deprecated and will be removed in 19.0. Use ` + "`" + `gitlab_project_label` + "`" + `instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/labels/#get-a-single-project-label)`,
		DeprecationMessage: "This resource is deprecated and will be removed in 19.0. Use `gitlab_project_label` instead.",
	}
}

type gitlabProjectLabelResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LabelID     types.Int64  `tfsdk:"label_id"`
	Project     types.String `tfsdk:"project"`
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	ColorHex    types.String `tfsdk:"color_hex"`
	Description types.String `tfsdk:"description"`
}

type gitlabProjectLabelResource struct {
	client *gitlab.Client

	// Represents the name and description of the resource, since this resource uses both `gitlab_project_label` and `gitlab_label` for
	// backwards compatibility reasons. Should be removed in %19.0
	ResourceName        string
	ResourceDescription string
	DeprecationMessage  string
}

func (r *gitlabProjectLabelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceName
}

func (r *gitlabProjectLabelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectLabelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getV1Schema()
}

func (r *gitlabProjectLabelResource) getV1Schema() schema.Schema {
	toReturn := schema.Schema{
		MarkdownDescription: r.ResourceDescription,
		DeprecationMessage:  r.DeprecationMessage,
		Version:             2,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id>:<label-id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"label_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the project label.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The name or id of the project to add the label to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the label.",
				Required:            true,
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "The color of the label given in 6-digit hex notation with leading '#' sign (e.g. #FFAABB) or one of the [CSS color names](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Values/color_value#Color_keywords).",
				Required:            true,
			},
			"color_hex": schema.StringAttribute{
				MarkdownDescription: "Read-only, used by the provider to store the API response color. This is always in the 6-digit hex notation with leading '#' sign (e.g. #FFAABB). If `color` contains a color name, this attribute contains the hex notation equivalent. Otherwise, the value of this attribute is the same as `color`.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the label.",
				Optional:            true,
				Computed:            true,
			},
		},
	}

	return toReturn
}

func (r *gitlabProjectLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectLabelResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	color := data.Color.ValueString()
	options := &gitlab.CreateLabelOptions{
		Name:  gitlab.Ptr(data.Name.ValueString()),
		Color: gitlab.Ptr(color),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab label %s", *options.Name))

	label, _, err := r.client.Labels.CreateLabel(project, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to create project label: %s", err.Error()))
		return
	}

	labelID := strconv.FormatInt(int64(label.ID), 10)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &labelID))
	data.modelToStateModel(label, color, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, labelID, err := data.ResourceGitlabProjectLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	label, _, err := r.client.Labels.GetLabel(project, strconv.FormatInt(int64(labelID), 10), gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddWarning("GitLab API error occured", fmt.Sprintf("Project label doesn't exist anymore, removing from state: %s", err.Error()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to get project label: %s", err.Error()))
		return
	}

	data.modelToStateModel(label, data.Color.ValueString(), project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project, labelID, err := data.ResourceGitlabProjectLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	color := data.Color.ValueString()
	options := &gitlab.UpdateLabelOptions{
		NewName: gitlab.Ptr(data.Name.ValueString()),
		Color:   gitlab.Ptr(color),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		options.Description = gitlab.Ptr(data.Description.ValueString())
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab label %s", data.ID.ValueString()))
	label, _, err := r.client.Labels.UpdateLabel(project, strconv.FormatInt(int64(labelID), 10), options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update project label: %s", err.Error()))
		return
	}

	data.modelToStateModel(label, color, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, labelID, err := data.ResourceGitlabProjectLabelParseID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource ID", fmt.Sprintf("Unable to parse resource ID: %s, %s", data.ID.ValueString(), err.Error()))
		return
	}

	_, err = r.client.Labels.DeleteLabel(project, strconv.FormatInt(int64(labelID), 10), nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to delete project label: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *gitlabProjectLabelResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	schema := r.getV1Schema()

	// We can reuse the same schema for all state upgraders because the schema definition
	// itself didn't change between versions - only the internal ID format changed from
	// V0 (label-name) to V1 (project:label-name) to V2 (project:label-id). The ID field
	// is not part of the formal schema attributes, so the same schema is valid for all versions.
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data gitlabProjectLabelResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				newData := resourceGitlabProjectLabelStateUpgradeV0(ctx, &data)

				// As we are upgrading from V0 to V2 directly, we need to apply the V1 -> V2 migration as well.
				// This is because Terraform does not chain state upgraders.
				err := r.upgradeIdToV2Id(ctx, newData)
				if err != nil {
					tflog.Error(ctx, "Failed to upgrade resource ID", map[string]any{
						"oldId":   data.ID.ValueString(),
						"project": data.Project.ValueString(),
					})
					resp.Diagnostics.AddError("Failed to upgrade resource ID", fmt.Sprintf("Unable to upgrade resource ID: %s, %s", data.ID.ValueString(), err.Error()))
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)
			},
		},
		1: {
			PriorSchema: &schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data gitlabProjectLabelResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				err := r.upgradeIdToV2Id(ctx, &data)
				if err != nil {
					tflog.Error(ctx, "Failed to upgrade resource ID", map[string]any{
						"oldId":   data.ID.ValueString(),
						"project": data.Project.ValueString(),
					})
					resp.Diagnostics.AddError("Failed to upgrade resource ID", fmt.Sprintf("Unable to upgrade resource ID: %s, %s", data.ID.ValueString(), err.Error()))
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

// resourceGitlabProjectLabelStateUpgradeV0 performs the state migration from V0 to V1.
func resourceGitlabProjectLabelStateUpgradeV0(ctx context.Context, data *gitlabProjectLabelResourceModel) *gitlabProjectLabelResourceModel {
	oldID := data.ID.ValueString()
	project := data.Project.ValueString()
	tflog.Debug(ctx, "attempting state migration from V0 to V1 - changing the `id` attribute format", map[string]any{"project": project, "v0-id": oldID})
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &oldID))
	tflog.Debug(ctx, "migrated `id` attribute for V0 to V1", map[string]any{"v0-id": oldID, "v1-id": data.ID.ValueString()})
	return data
}

// upgradeIdToV2Id performs the state migration from V1 to V2 ID format (<project>:<label_id>).
// This handles migration from V1 (<project>:<label_name>) to V2 (<project>:<label_id>).
// It first tries to use the LabelID from state, and only falls back to API calls when necessary.
func (r *gitlabProjectLabelResource) upgradeIdToV2Id(ctx context.Context, input *gitlabProjectLabelResourceModel) error {
	// If LabelID is already available in state, use it directly (no API call needed)
	if !input.LabelID.IsNull() && !input.LabelID.IsUnknown() {
		tflog.Debug(ctx, "Using LabelID from state, no API call needed", map[string]any{
			"label_id": input.LabelID.ValueInt64(),
		})
	} else {
		// At this point the state has already passed through the V0→V1 upgrader, so any
		// remaining legacy IDs are guaranteed to be in the V1 format <project>:<label_name>.
		// LabelID is not available in state, need to fetch it from API
		tflog.Info(ctx, "LabelID not available in state, fetching from API", map[string]any{
			"project": input.Project.ValueString(),
			"old_id":  input.ID.ValueString(),
		})

		project := input.Project.ValueString()
		oldID := input.ID.ValueString()

		// Parse the V1 ID format: <project>:<label_name>
		_, labelName, err := utils.ParseTwoPartID(oldID)
		if err != nil {
			return fmt.Errorf("failed to parse V1 ID format '%s': %w", oldID, err)
		}

		tflog.Debug(ctx, "Fetching label by name to get label ID", map[string]any{
			"project":    project,
			"label_name": labelName,
		})

		label, _, err := r.client.Labels.GetLabel(project, labelName, gitlab.WithContext(ctx))
		if err != nil {
			// Log the error but don't fail the migration completely
			// The resource will be recreated if the label doesn't exist
			tflog.Warn(ctx, "Failed to fetch label for state migration", map[string]any{
				"project":    project,
				"label_name": labelName,
				"error":      err.Error(),
			})
			// Continue with migration without setting LabelID - resource will be recreated if needed
		} else {
			// Set the label ID into state for future use
			input.LabelID = types.Int64Value(int64(label.ID))

			tflog.Info(ctx, "Successfully fetched label ID from API", map[string]any{
				"label_id":   label.ID,
				"label_name": label.Name,
			})
		}
	}

	// Check if we have a valid LabelID to build the new ID format
	if input.LabelID.IsNull() || input.LabelID.IsUnknown() {
		tflog.Warn(ctx, "LabelID not available after migration attempt, keeping original ID", map[string]any{
			"original_id": input.ID.ValueString(),
		})
		// Keep the original ID - Terraform will handle resource recreation if needed
		return nil
	}

	// Build the new V2 ID format
	stringLabelID := strconv.FormatInt(input.LabelID.ValueInt64(), 10)
	project := input.Project.ValueString()
	newID := utils.BuildTwoPartID(&project, &stringLabelID)

	tflog.Debug(ctx, "Upgrading state to the V2 ID", map[string]any{
		"oldId":   input.ID.ValueString(),
		"project": project,
		"newId":   newID,
	})

	input.ID = types.StringValue(newID)
	return nil
}

// MoveState implements the ResourceWithMoveState interface to support moving state from the deprecated gitlab_label resource.
// This enables users to migrate from gitlab_label to gitlab_project_label using Terraform's moved block.
// Note: Cross-resource-type state moves require Terraform 1.8 or later.
func (r *gitlabProjectLabelResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		// This StateMover implements the migration from
		// `gitlab_label` -> `gitlab_project_label`.
		// The SourceSchema needs to match the deprecated `gitlab_label` as a result.
		{
			SourceSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"label_id": schema.Int64Attribute{
						Computed: true,
					},
					"project": schema.StringAttribute{
						Required: true,
					},
					"name": schema.StringAttribute{
						Required: true,
					},
					"color": schema.StringAttribute{
						Required: true,
					},
					"color_hex": schema.StringAttribute{
						Computed: true,
					},
					"description": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
				},
			},
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Only handle moves from gitlab_label resource
				if req.SourceTypeName != "gitlab_label" {
					resp.Diagnostics.AddError("Invalid source resource type", fmt.Sprintf("Expected source type 'gitlab_label', got '%s'", req.SourceTypeName))
					return
				}

				// Check provider address (without hostname for compatibility)
				// Accept anything that ends with gitlab, which seems the safest.
				//  hashicorp/gitlab is used in tests
				//  gitlab-org/gitlab is used in production
				//  gitlabhq/gitlab is referenced on the provider docs.
				if !strings.HasSuffix(req.SourceProviderAddress, "gitlab") {
					resp.Diagnostics.AddError("Invalid source provider address", fmt.Sprintf("Expected provider address ending with 'gitlab', got '%s'", req.SourceProviderAddress))
					return
				}

				// Define the source model matching the old gitlab_label schema
				type sourceModel struct {
					ID          types.String `tfsdk:"id"`
					LabelID     types.Int64  `tfsdk:"label_id"`
					Project     types.String `tfsdk:"project"`
					Name        types.String `tfsdk:"name"`
					Color       types.String `tfsdk:"color"`
					ColorHex    types.String `tfsdk:"color_hex"`
					Description types.String `tfsdk:"description"`
				}

				var sourceStateData sourceModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Create the target state data - schema is identical, so direct copy
				targetStateData := gitlabProjectLabelResourceModel{
					ID:          sourceStateData.ID,
					LabelID:     sourceStateData.LabelID,
					Project:     sourceStateData.Project,
					Name:        sourceStateData.Name,
					Color:       sourceStateData.Color,
					ColorHex:    sourceStateData.ColorHex,
					Description: sourceStateData.Description,
				}

				tflog.Debug(ctx, "Moving state from gitlab_label to gitlab_project_label", map[string]any{
					"id":      sourceStateData.ID.ValueString(),
					"project": sourceStateData.Project.ValueString(),
					"name":    sourceStateData.Name.ValueString(),
				})

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}

func (data *gitlabProjectLabelResourceModel) modelToStateModel(label *gitlab.Label, color string, project string) {
	data.LabelID = types.Int64Value(int64(label.ID))
	data.Project = types.StringValue(project)
	data.Description = types.StringValue(label.Description)
	if color == "" {
		data.Color = types.StringValue(label.Color)
	} else {
		data.Color = types.StringValue(color)
	}
	data.ColorHex = types.StringValue(label.Color)
	data.Name = types.StringValue(label.Name)
}

func (d *gitlabProjectLabelResourceModel) ResourceGitlabProjectLabelParseID(id string) (string, int, error) {
	project, rawLabelID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	labelID, err := strconv.Atoi(rawLabelID)
	if err != nil {
		return "", 0, err
	}

	return project, labelID, nil
}
