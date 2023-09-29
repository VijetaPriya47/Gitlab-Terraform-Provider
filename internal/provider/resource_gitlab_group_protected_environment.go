package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                   = &gitlabGroupProtectedEnvironmentResource{}
	_ resource.ResourceWithConfigure      = &gitlabGroupProtectedEnvironmentResource{}
	_ resource.ResourceWithImportState    = &gitlabGroupProtectedEnvironmentResource{}
	_ resource.ResourceWithValidateConfig = &gitlabGroupProtectedEnvironmentResource{}
)

func init() {
	registerResource(NewgitlabGroupProtectedEnvironmentResource)
}

func NewgitlabGroupProtectedEnvironmentResource() resource.Resource {
	return &gitlabGroupProtectedEnvironmentResource{}
}

type gitlabGroupProtectedEnvironmentResource struct {
	client *gitlab.Client
}

type gitlabGroupProtectedEnvironmentResourceModel struct {
	Id                    types.String `tfsdk:"id"`
	Group                 types.String `tfsdk:"group"`
	Environment           types.String `tfsdk:"environment"`
	RequiredApprovalCount types.Int64  `tfsdk:"required_approval_count"`

	// Set objects
	DeployAccessLevels types.Set `tfsdk:"deploy_access_levels"`
	ApprovalRules      types.Set `tfsdk:"approval_rules"`
}

type gitlabGroupProtectedEnvironmentDeployAccessLevelModel struct {
	ID                     types.Int64  `tfsdk:"id"`
	AccessLevel            types.String `tfsdk:"access_level"`
	AccessLevelDescription types.String `tfsdk:"access_level_description"`
	UserId                 types.Int64  `tfsdk:"user_id"`
	GroupId                types.Int64  `tfsdk:"group_id"`
}

type gitlabGroupProtectedEnvironmentApprovalRuleModel struct {
	ID                     types.Int64  `tfsdk:"id"`
	AccessLevel            types.String `tfsdk:"access_level"`
	AccessLevelDescription types.String `tfsdk:"access_level_description"`
	UserId                 types.Int64  `tfsdk:"user_id"`
	GroupId                types.Int64  `tfsdk:"group_id"`
	RequiredApprovals      types.Int64  `tfsdk:"required_approvals"`
}

func (r *gitlabGroupProtectedEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_protected_environment"
}

func (r *gitlabGroupProtectedEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Valid values for the schema
	var validEnvironments = []string{"production", "staging", "testing", "development", "other"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_protected_environment`" + ` resource allows to manage the lifecycle of a protected environment in a group.

~> In order to use a user_id in the ` + "`deploy_access_levels`" + ` configuration,
   you need to make sure that users have access to the group with Maintainer role or higher.
   In order to use a group_id in the ` + "`deploy_access_levels`" + ` configuration,
   the group_id must be a sub-group under the given group.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/group_protected_environments.html)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>:<environment-name>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the group which the protected environment is created against.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The deployment tier of the environment.  Valid values are %s.", utils.RenderValueListForDocs(validEnvironments)),
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf(validEnvironments...),
				},
			},
			"required_approval_count": schema.Int64Attribute{
				MarkdownDescription: "The number of approvals required to deploy to this environment.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"deploy_access_levels": groupDeployAccessLevelSchema(),
			"approval_rules":       groupApprovalRuleSchema(),
		},
	}
}

func groupDeployAccessLevelSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		MarkdownDescription: "Array of access levels allowed to deploy, with each described by a hash.",
		Required:            true,
		Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.Int64Attribute{
					MarkdownDescription: "The unique ID of the Deploy Access Level object.",
					Computed:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				},
				"access_level": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Levels of access required to deploy to this protected environment. Valid values are %s.", utils.RenderValueListForDocs(api.ValidProtectedEnvironmentDeploymentLevelNames)),
					Optional:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					Validators: []validator.String{
						stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("user_id"), path.MatchRelative().AtParent().AtName("group_id")),
						stringvalidator.OneOf(api.ValidProtectedEnvironmentDeploymentLevelNames...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of level of access.",
					Computed:            true,
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of the user allowed to deploy to this protected environment. The user must be a member of the group with Maintainer role or higher.",
					Optional:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of the group allowed to deploy to this protected environment. The group must be a sub-group under the given group.",
					Optional:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
			},
		},
	}
}

func groupApprovalRuleSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		MarkdownDescription: "Array of approval rules to deploy, with each described by a hash.",
		Optional:            true,
		Computed:            true,
		NestedObject: schema.NestedAttributeObject{
			PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
			Attributes: map[string]schema.Attribute{
				"id": schema.Int64Attribute{
					MarkdownDescription: "The unique ID of the Approval Rules object.",
					Computed:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				},
				"access_level": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Levels of access allowed to approve a deployment to this protected environment. Valid values are %s.", utils.RenderValueListForDocs(api.ValidProtectedEnvironmentDeploymentLevelNames)),
					Optional:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					Validators: []validator.String{
						stringvalidator.OneOf(api.ValidProtectedEnvironmentDeploymentLevelNames...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of level of access.",
					Computed:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of the user allowed to approve a deployment to this protected environment. The user must be a member of the group with Maintainer role or higher. This is mutually exclusive with group_id and required_approvals.",
					Optional:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of the group allowed to approve a deployment to this protected environment. TThe group must be a sub-group under the given group. This is mutually exclusive with user_id.",
					Optional:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
				"required_approvals": schema.Int64Attribute{
					MarkdownDescription: "The number of approval required to allow deployment to this protected environment. This is mutually exclusive with user_id.",
					Optional:            true,
					Computed:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabGroupProtectedEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

func (r *gitlabGroupProtectedEnvironmentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabGroupProtectedEnvironmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.DeployAccessLevels.IsNull() && !data.DeployAccessLevels.IsUnknown() {
		deployAccessLevels := make([]*gitlabGroupProtectedEnvironmentDeployAccessLevelModel, 0, len(data.DeployAccessLevels.Elements()))
		resp.Diagnostics.Append(data.DeployAccessLevels.ElementsAs(ctx, &deployAccessLevels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, dal := range deployAccessLevels {
			if !dal.UserId.IsNull() && !dal.GroupId.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("deploy_access_levels").AtListIndex(i), "Invalid Attribute Combination", "Cannot have user_id and group_id in the same deploy_access_levels block")
			}
		}

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !data.ApprovalRules.IsNull() && !data.DeployAccessLevels.IsUnknown() {
		approvalRules := make([]*gitlabGroupProtectedEnvironmentApprovalRuleModel, 0, len(data.ApprovalRules.Elements()))
		resp.Diagnostics.Append(data.ApprovalRules.ElementsAs(ctx, &approvalRules, true)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, ar := range approvalRules {
			if !ar.UserId.IsNull() && !ar.GroupId.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("approval_rules").AtListIndex(i), "Invalid Attribute Combination", "Cannot have user_id and group_id in the same approval_rules block")
			}
		}

		for i, ar := range approvalRules {
			if !ar.UserId.IsNull() && !ar.RequiredApprovals.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("approval_rules").AtListIndex(i), "Invalid Attribute Combination", "Cannot have user_id and required_approvals in the same approval_rules block")
			}
		}

		if resp.Diagnostics.HasError() {
			return
		}
	}
}

// Create creates a new upstream resources and adds it into the Terraform state.
func (r *gitlabGroupProtectedEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupProtectedEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	groupID := data.Group.ValueString()
	environmentName := data.Environment.ValueString()

	// configure GitLab API call
	options := &gitlab.ProtectGroupEnvironmentOptions{
		Name: gitlab.String(environmentName),
	}

	if !data.RequiredApprovalCount.IsNull() {
		options.RequiredApprovalCount = gitlab.Int(int(data.RequiredApprovalCount.ValueInt64()))
	}

	// deploy access levels
	deployAccessLevels := make([]*gitlabGroupProtectedEnvironmentDeployAccessLevelModel, 0, len(data.DeployAccessLevels.Elements()))
	resp.Diagnostics.Append(data.DeployAccessLevels.ElementsAs(ctx, &deployAccessLevels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deployAccessLevelsOption := make([]*gitlab.GroupEnvironmentAccessOptions, len(deployAccessLevels))
	for i, v := range deployAccessLevels {
		deployAccessLevelOptions := &gitlab.GroupEnvironmentAccessOptions{}

		if !v.AccessLevel.IsNull() && v.AccessLevel.ValueString() != "" {
			deployAccessLevelOptions.AccessLevel = gitlab.AccessLevel(api.AccessLevelNameToValue[v.AccessLevel.ValueString()])
		}
		if !v.UserId.IsNull() && v.UserId.ValueInt64() != 0 {
			deployAccessLevelOptions.UserID = gitlab.Int(int(v.UserId.ValueInt64()))
		}
		if !v.GroupId.IsNull() && v.GroupId.ValueInt64() != 0 {
			deployAccessLevelOptions.GroupID = gitlab.Int(int(v.GroupId.ValueInt64()))
		}
		deployAccessLevelsOption[i] = deployAccessLevelOptions
	}
	options.DeployAccessLevels = &deployAccessLevelsOption

	// approval rules
	approvalRules := make([]*gitlabGroupProtectedEnvironmentApprovalRuleModel, 0, len(data.ApprovalRules.Elements()))
	resp.Diagnostics.Append(data.ApprovalRules.ElementsAs(ctx, &approvalRules, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	approvalRulesOption := make([]*gitlab.GroupEnvironmentApprovalRuleOptions, len(approvalRules))
	for i, v := range approvalRules {
		approvalRuleOptions := &gitlab.GroupEnvironmentApprovalRuleOptions{}

		if !v.AccessLevel.IsNull() && v.AccessLevel.ValueString() != "" {
			approvalRuleOptions.AccessLevel = gitlab.AccessLevel(api.AccessLevelNameToValue[v.AccessLevel.ValueString()])
		}
		if !v.UserId.IsNull() && v.UserId.ValueInt64() != 0 {
			approvalRuleOptions.UserID = gitlab.Int(int(v.UserId.ValueInt64()))
		}
		if !v.GroupId.IsNull() && v.GroupId.ValueInt64() != 0 {
			approvalRuleOptions.GroupID = gitlab.Int(int(v.GroupId.ValueInt64()))
		}
		if !v.RequiredApprovals.IsNull() && v.RequiredApprovals.ValueInt64() != 0 {
			approvalRuleOptions.RequiredApprovalCount = gitlab.Int(int(v.RequiredApprovals.ValueInt64()))
		}

		approvalRulesOption[i] = approvalRuleOptions
	}
	options.ApprovalRules = &approvalRulesOption

	tflog.Debug(ctx, "Creating group protected environment with options", map[string]interface{}{
		"data":    data,
		"groupId": groupID,
		"name":    environmentName,
		"options": options,
	})

	// Protect environment
	protectedEnvironment, _, err := r.client.GroupProtectedEnvironments.ProtectGroupEnvironment(groupID, options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The protected environment feature is not available on this group. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to protect environment: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, "Group Protected Environment before state is persisted", map[string]interface{}{
		"data":        data,
		"groupId":     groupID,
		"name":        environmentName,
		"environment": protectedEnvironment,
	})

	// Create resource ID and persist in state model
	data.Id = types.StringValue(utils.BuildTwoPartID(&groupID, &protectedEnvironment.Name))

	// persist API response in state model
	r.protectedEnvironmentToStateModel(ctx, resp.Diagnostics, groupID, protectedEnvironment, data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabGroupProtectedEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupProtectedEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	groupID, environmentName, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<group>:<environment-name>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	// Read environment protection
	protectedEnvironment, _, err := r.client.GroupProtectedEnvironments.GetGroupProtectedEnvironment(groupID, environmentName, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "protected environment does not exist, removing from state", map[string]interface{}{
				"group": groupID, "environment": environmentName,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read group protected environment details: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.protectedEnvironmentToStateModel(ctx, resp.Diagnostics, groupID, protectedEnvironment, data)

	tflog.Debug(ctx, "Protected Environment when state is being read", map[string]interface{}{
		"group": groupID,
		"name":  environmentName,
		"data":  data,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabGroupProtectedEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupProtectedEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	groupID := data.Group.ValueString()
	environmentName := data.Environment.ValueString()

	// Retrieve the protected environment (to know which deploy/approval rules to remove)
	protectedEnvironment, _, err := r.client.GroupProtectedEnvironments.GetGroupProtectedEnvironment(groupID, environmentName, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The protected environment feature is not available on this group. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update protected environment details: %s", err.Error()))
		return
	}

	// configure GitLab API call
	options := &gitlab.UpdateGroupProtectedEnvironmentOptions{
		Name: gitlab.String(environmentName),
	}

	if !data.RequiredApprovalCount.IsNull() {
		options.RequiredApprovalCount = gitlab.Int(int(data.RequiredApprovalCount.ValueInt64()))
	}

	// deploy access levels
	deployAccessLevels := make([]*gitlabGroupProtectedEnvironmentDeployAccessLevelModel, 0, len(data.DeployAccessLevels.Elements()))
	resp.Diagnostics.Append(data.DeployAccessLevels.ElementsAs(ctx, &deployAccessLevels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deployAccessLevelsOption := make([]*gitlab.UpdateGroupEnvironmentAccessOptions, 0)
	for _, v := range deployAccessLevels {
		deployAccessLevelOptions := &gitlab.UpdateGroupEnvironmentAccessOptions{}

		// the ID will be null when adding a new deploy rule via update
		if !v.ID.IsNull() && v.ID.ValueInt64() != 0 {
			deployAccessLevelOptions.ID = gitlab.Int(int(v.ID.ValueInt64()))
		}

		if !v.AccessLevel.IsNull() && v.AccessLevel.ValueString() != "" {
			deployAccessLevelOptions.AccessLevel = gitlab.AccessLevel(api.AccessLevelNameToValue[v.AccessLevel.ValueString()])
		}
		if !v.UserId.IsNull() && v.UserId.ValueInt64() != 0 {
			deployAccessLevelOptions.UserID = gitlab.Int(int(v.UserId.ValueInt64()))
		}
		if !v.GroupId.IsNull() && v.GroupId.ValueInt64() != 0 {
			deployAccessLevelOptions.GroupID = gitlab.Int(int(v.GroupId.ValueInt64()))
		}

		deployAccessLevelsOption = append(deployAccessLevelsOption, deployAccessLevelOptions)
	}

	// Remove deploy access levels that aren't present in the config
	for _, v := range protectedEnvironment.DeployAccessLevels {
		isPresent := false
		for _, j := range deployAccessLevels {
			if v.ID == int(j.ID.ValueInt64()) {
				isPresent = true
				break
			}
		}

		// If the existing deploy isn't present, add it to the values to remove it
		if !isPresent {
			deployAccessLevelOptions := &gitlab.UpdateGroupEnvironmentAccessOptions{
				ID:      gitlab.Int(v.ID),
				Destroy: gitlab.Bool(true),
			}

			// Seems weird, but the API does validate that these values are present even
			// when destroying
			if v.AccessLevel != 0 {
				deployAccessLevelOptions.AccessLevel = &v.AccessLevel
			}
			if v.UserID != 0 {
				deployAccessLevelOptions.UserID = &v.UserID
			}
			if v.GroupID != 0 {
				deployAccessLevelOptions.GroupID = &v.GroupID
			}

			deployAccessLevelsOption = append(deployAccessLevelsOption, deployAccessLevelOptions)
		}
	}

	options.DeployAccessLevels = &deployAccessLevelsOption

	// approval rules
	approvalRules := make([]*gitlabGroupProtectedEnvironmentApprovalRuleModel, 0, len(data.ApprovalRules.Elements()))
	resp.Diagnostics.Append(data.ApprovalRules.ElementsAs(ctx, &approvalRules, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	approvalRulesOptionSlice := make([]*gitlab.UpdateGroupEnvironmentApprovalRuleOptions, 0)
	for _, v := range approvalRules {
		approvalRuleOptions := &gitlab.UpdateGroupEnvironmentApprovalRuleOptions{}

		// the ID will be null when adding a new approval rule via update
		if !v.ID.IsNull() && v.ID.ValueInt64() != 0 {
			approvalRuleOptions.ID = gitlab.Int(int(v.ID.ValueInt64()))
		}

		if !v.AccessLevel.IsNull() && v.AccessLevel.ValueString() != "" {
			approvalRuleOptions.AccessLevel = gitlab.AccessLevel(api.AccessLevelNameToValue[v.AccessLevel.ValueString()])
		}
		if !v.UserId.IsNull() && v.UserId.ValueInt64() != 0 {
			approvalRuleOptions.UserID = gitlab.Int(int(v.UserId.ValueInt64()))
		}
		if !v.GroupId.IsNull() && v.GroupId.ValueInt64() != 0 {
			approvalRuleOptions.GroupID = gitlab.Int(int(v.GroupId.ValueInt64()))
		}
		if !v.RequiredApprovals.IsNull() && v.RequiredApprovals.ValueInt64() != 0 {
			approvalRuleOptions.RequiredApprovalCount = gitlab.Int(int(v.RequiredApprovals.ValueInt64()))
		}

		approvalRulesOptionSlice = append(approvalRulesOptionSlice, approvalRuleOptions)
	}

	// Remove approval levels that aren't present in the config
	for _, v := range protectedEnvironment.ApprovalRules {
		isPresent := false
		for _, j := range approvalRules {
			if v.ID == int(j.ID.ValueInt64()) {
				isPresent = true
				break
			}
		}

		// If the existing deploy isn't present, add it to the values to remove it
		if !isPresent {
			approvalRuleOptions := &gitlab.UpdateGroupEnvironmentApprovalRuleOptions{
				ID:      gitlab.Int(v.ID),
				Destroy: gitlab.Bool(true),
			}

			// Seems weird, but the API does validate that these values are present even
			// when destroying
			if v.AccessLevel != 0 {
				approvalRuleOptions.AccessLevel = &v.AccessLevel
			}
			if v.UserID != 0 {
				approvalRuleOptions.UserID = &v.UserID
			}
			if v.GroupID != 0 {
				approvalRuleOptions.GroupID = &v.GroupID
			}
			approvalRulesOptionSlice = append(approvalRulesOptionSlice, approvalRuleOptions)
		}
	}
	options.ApprovalRules = &approvalRulesOptionSlice

	tflog.Debug(ctx, "Updating group protected environment with options", map[string]interface{}{
		"group":            groupID,
		"options":          options,
		"environment_name": environmentName,
	})

	protectedEnvironment, _, err = r.client.GroupProtectedEnvironments.UpdateGroupProtectedEnvironment(groupID, environmentName, options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The protected environment feature is not available on this group. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update protected environment details: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, "Updating group protected environment completed with options", map[string]interface{}{
		"group":            groupID,
		"environment_name": environmentName,
		"result":           protectedEnvironment,
	})

	// Add data to state
	r.protectedEnvironmentToStateModel(ctx, resp.Diagnostics, groupID, protectedEnvironment, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource.
func (r *gitlabGroupProtectedEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupProtectedEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	groupID, environmentName, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<group>:<environment-name>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	if _, err = r.client.GroupProtectedEnvironments.UnprotectGroupEnvironment(groupID, environmentName, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete protected environment: %s", err.Error()),
		)
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabGroupProtectedEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabGroupProtectedEnvironmentResource) protectedEnvironmentToStateModel(ctx context.Context, existingDiag diag.Diagnostics, groupID string, protectedEnvironment *gitlab.GroupProtectedEnvironment, data *gitlabGroupProtectedEnvironmentResourceModel) {
	data.Group = types.StringValue(groupID)
	data.Environment = types.StringValue(protectedEnvironment.Name)
	data.RequiredApprovalCount = types.Int64Value(int64(protectedEnvironment.RequiredApprovalCount))

	deployAccessLevelsData := make([]gitlabGroupProtectedEnvironmentDeployAccessLevelModel, 0)
	for _, v := range protectedEnvironment.DeployAccessLevels {
		obj := *v
		deployAccessLevelData := gitlabGroupProtectedEnvironmentDeployAccessLevelModel{
			ID:                     types.Int64Value(int64(obj.ID)),
			AccessLevelDescription: types.StringValue(obj.AccessLevelDescription),
		}
		if obj.AccessLevel != 0 {
			deployAccessLevelData.AccessLevel = types.StringValue(api.AccessLevelValueToName[obj.AccessLevel])
		}
		if obj.UserID != 0 {
			deployAccessLevelData.UserId = types.Int64Value(int64(obj.UserID))
		}
		if obj.GroupID != 0 {
			deployAccessLevelData.GroupId = types.Int64Value(int64(obj.GroupID))
		}

		deployAccessLevelsData = append(deployAccessLevelsData, deployAccessLevelData)
	}
	accessLevelSetType, newDiag := types.SetValueFrom(ctx, groupDeployAccessLevelSchema().NestedObject.Type(), deployAccessLevelsData)
	existingDiag.Append(newDiag...)
	data.DeployAccessLevels = accessLevelSetType

	approvalRulesData := make([]gitlabGroupProtectedEnvironmentApprovalRuleModel, 0)
	for _, v := range protectedEnvironment.ApprovalRules {
		obj := *v
		approvalRuleData := gitlabGroupProtectedEnvironmentApprovalRuleModel{
			ID:                     types.Int64Value(int64(obj.ID)),
			AccessLevelDescription: types.StringValue(obj.AccessLevelDescription),
		}
		if obj.AccessLevel != 0 {
			approvalRuleData.AccessLevel = types.StringValue((api.AccessLevelValueToName[obj.AccessLevel]))
		}
		if obj.UserID != 0 {
			approvalRuleData.UserId = types.Int64Value(int64(obj.UserID))
		}
		if obj.GroupID != 0 {
			approvalRuleData.GroupId = types.Int64Value(int64(obj.GroupID))
		}
		if obj.RequiredApprovalCount != 0 {
			approvalRuleData.RequiredApprovals = types.Int64Value(int64(obj.RequiredApprovalCount))
		}
		approvalRulesData = append(approvalRulesData, approvalRuleData)
	}
	approvalRulesSetType, newDiag := types.SetValueFrom(ctx, groupApprovalRuleSchema().NestedObject.Type(), approvalRulesData)
	existingDiag.Append(newDiag...)
	data.ApprovalRules = approvalRulesSetType
}
