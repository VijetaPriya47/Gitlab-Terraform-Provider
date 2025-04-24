package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &gitlabProjectProtectedEnvironmentResource{}
var _ resource.ResourceWithConfigure = &gitlabProjectProtectedEnvironmentResource{}
var _ resource.ResourceWithImportState = &gitlabProjectProtectedEnvironmentResource{}
var _ resource.ResourceWithValidateConfig = &gitlabProjectProtectedEnvironmentResource{}

func init() {
	registerResource(NewGitLabProjectProtectedEnvironmentResource)
}

// NewGitLabProjectProtectedEnvironmentResource is a helper function to simplify the provider implementation.
func NewGitLabProjectProtectedEnvironmentResource() resource.Resource {
	return &gitlabProjectProtectedEnvironmentResource{}
}

// gitlabProjectProtectedEnvironmentResource defines the resource implementation.
type gitlabProjectProtectedEnvironmentResource struct {
	client *gitlab.Client
}

// gitlabProjectProtectedEnvironmentResourceModel describes the resource data model.
type gitlabProjectProtectedEnvironmentResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Project     types.String `tfsdk:"project"`
	Environment types.String `tfsdk:"environment"`

	// Set objects
	DeployAccessLevels types.Set  `tfsdk:"deploy_access_levels"`
	ApprovalRules      types.List `tfsdk:"approval_rules"`
}

type gitlabProjectProtectedEnvironmentDeployAccessLevelModel struct {
	ID                     types.Int64  `tfsdk:"id"`
	AccessLevel            types.String `tfsdk:"access_level"`
	AccessLevelDescription types.String `tfsdk:"access_level_description"`
	UserId                 types.Int64  `tfsdk:"user_id"`
	GroupId                types.Int64  `tfsdk:"group_id"`
	GroupInheritanceType   types.Int64  `tfsdk:"group_inheritance_type"`
}

type gitlabProjectProtectedEnvironmentApprovalRuleModel struct {
	ID                     types.Int64  `tfsdk:"id"`
	AccessLevel            types.String `tfsdk:"access_level"`
	AccessLevelDescription types.String `tfsdk:"access_level_description"`
	UserId                 types.Int64  `tfsdk:"user_id"`
	GroupId                types.Int64  `tfsdk:"group_id"`
	RequiredApprovals      types.Int64  `tfsdk:"required_approvals"`
	GroupInheritanceType   types.Int64  `tfsdk:"group_inheritance_type"`
}

func (r *gitlabProjectProtectedEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_protected_environment"
}

func (r *gitlabProjectProtectedEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_protected_environment`" + ` resource allows to manage the lifecycle of a protected environment in a project.

~> In order to use a user or group in the ` + "`deploy_access_levels`" + ` configuration,
   you need to make sure that users have access to the project and groups must have this project shared.
   You may use the ` + "`gitlab_project_membership`" + ` and ` + "`gitlab_project_shared_group`" + ` resources to achieve this.
   Unfortunately, the GitLab API does not complain about users and groups without access to the project and just ignores those.
   In case this happens you will get perpetual state diffs.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/protected_environments/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<environment-name>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project which the protected environment is created against.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "The name of the environment.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"approval_rules": approvalRuleSchema(),
		},
		Blocks: map[string]schema.Block{
			"deploy_access_levels": deployAccessLevelSchema(),
		},
	}
}

func deployAccessLevelSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		MarkdownDescription: "Array of access levels allowed to deploy, with each described by a hash.  Elements in the `deploy_access_levels` should be one of `user_id`, `group_id` or `access_level`.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.Int64Attribute{
					MarkdownDescription: "The unique ID of the Deploy Access Level object.",
					Computed:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				},
				"access_level": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Levels of access required to deploy to this protected environment. Mutually exclusive with `user_id` and `group_id`. Valid values are %s.", utils.RenderValueListForDocs(api.ValidProtectedEnvironmentDeploymentLevelNames)),
					Optional:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					Validators: []validator.String{
						stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("user_id"), path.MatchRelative().AtParent().AtName("group_id")),
						stringvalidator.OneOfCaseInsensitive(api.ValidProtectedEnvironmentDeploymentLevelNames...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of level of access.",
					Computed:            true,
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of the user allowed to deploy to this protected environment. The user must be a member of the project. Mutually exclusive with `access_level` and `group_id`.",
					Optional:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of the group allowed to deploy to this protected environment. The project must be shared with the group. Mutually exclusive with `access_level` and `user_id`.",
					Optional:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
				"group_inheritance_type": schema.Int64Attribute{
					MarkdownDescription: "Group inheritance allows deploy access levels to take inherited group membership into account. Valid values are `0`, `1`. `0` => Direct group membership only, `1` => All inherited groups. Default: `0`",
					Optional:            true,
					Computed:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators: []validator.Int64{
						int64validator.OneOf([]int64{0, 1}...),
					},
				},
			},
		},
	}
}

func approvalRuleSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "Array of approval rules to deploy, with each described by a hash. Elements in the `approval_rules` should be one of `user_id`, `group_id` or `access_level`.",
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
					MarkdownDescription: fmt.Sprintf("Levels of access allowed to approve a deployment to this protected environment. Mutually exclusive with `user_id` and `group_id`. Valid values are %s.", utils.RenderValueListForDocs(api.ValidProtectedEnvironmentDeploymentLevelNames)),
					Optional:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					Validators: []validator.String{
						stringvalidator.OneOfCaseInsensitive(api.ValidProtectedEnvironmentDeploymentLevelNames...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of level of access.",
					Computed:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of the user allowed to approve a deployment to this protected environment. The user must be a member of the project. Mutually exclusive with `access_level` and `group_id`.",
					Optional:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators:          []validator.Int64{int64validator.AtLeast(1)},
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of the group allowed to approve a deployment to this protected environment. The project must be shared with the group. Mutually exclusive with `access_level` and `user_id`.",
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
				"group_inheritance_type": schema.Int64Attribute{
					MarkdownDescription: "Group inheritance allows deploy access levels to take inherited group membership into account. Valid values are `0`, `1`. `0` => Direct group membership only, `1` => All inherited groups. Default: `0`",
					Optional:            true,
					Computed:            true,
					PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					Validators: []validator.Int64{
						int64validator.OneOf([]int64{0, 1}...),
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabProjectProtectedEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectProtectedEnvironmentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data gitlabProjectProtectedEnvironmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.DeployAccessLevels.IsNull() && !data.DeployAccessLevels.IsUnknown() {
		deployAccessLevels := make([]*gitlabProjectProtectedEnvironmentDeployAccessLevelModel, 0, len(data.DeployAccessLevels.Elements()))
		resp.Diagnostics.Append(data.DeployAccessLevels.ElementsAs(ctx, &deployAccessLevels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, dal := range deployAccessLevels {
			if !dal.UserId.IsNull() && !dal.GroupId.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("deploy_access_levels").AtListIndex(i), "Invalid Attribute Combination", "Cannot have user_id and group_id in the same deploy_access_levels block")
			}
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.ApprovalRules.IsNull() && !data.DeployAccessLevels.IsUnknown() {
		rules := make([]*gitlabProjectProtectedEnvironmentApprovalRuleModel, 0, len(data.ApprovalRules.Elements()))
		data.ApprovalRules.ElementsAs(ctx, &rules, true)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, ar := range rules {
			if !ar.UserId.IsNull() && !ar.GroupId.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("approval_rules").AtListIndex(i), "Invalid Attribute Combination", "Cannot have user_id and group_id in the same approval_rules block")
			}
		}

		for i, ar := range rules {
			if !ar.UserId.IsNull() && !ar.RequiredApprovals.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("approval_rules").AtListIndex(i), "Invalid Attribute Combination", "Cannot have user_id and required_approvals in the same approval_rules block")
			}
		}
	}
}

// Create creates a new upstream resources and adds it into the Terraform state.
func (r *gitlabProjectProtectedEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectProtectedEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	rules := make([]*gitlabProjectProtectedEnvironmentApprovalRuleModel, len(data.ApprovalRules.Elements()))
	data.ApprovalRules.ElementsAs(ctx, &rules, true)

	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	projectID := data.Project.ValueString()
	environmentName := data.Environment.ValueString()

	// configure GitLab API call
	options := &gitlab.ProtectRepositoryEnvironmentsOptions{
		Name: gitlab.Ptr(environmentName),
	}

	// deploy access levels
	deployAccessLevels := make([]*gitlabProjectProtectedEnvironmentDeployAccessLevelModel, 0, len(data.DeployAccessLevels.Elements()))
	resp.Diagnostics.Append(data.DeployAccessLevels.ElementsAs(ctx, &deployAccessLevels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deployAccessLevelsOption := make([]*gitlab.EnvironmentAccessOptions, len(deployAccessLevels))
	for i, v := range deployAccessLevels {
		deployAccessLevelOptions := &gitlab.EnvironmentAccessOptions{}

		if !v.AccessLevel.IsNull() && v.AccessLevel.ValueString() != "" {
			deployAccessLevelOptions.AccessLevel = gitlab.Ptr(api.AccessLevelNameToValue[v.AccessLevel.ValueString()])
		}
		if !v.UserId.IsNull() && v.UserId.ValueInt64() != 0 {
			deployAccessLevelOptions.UserID = gitlab.Ptr(int(v.UserId.ValueInt64()))
		}
		if !v.GroupId.IsNull() && v.GroupId.ValueInt64() != 0 {
			deployAccessLevelOptions.GroupID = gitlab.Ptr(int(v.GroupId.ValueInt64()))
		}
		if !v.GroupInheritanceType.IsNull() && v.GroupInheritanceType.ValueInt64() != 0 {
			deployAccessLevelOptions.GroupInheritanceType = gitlab.Ptr(int(v.GroupInheritanceType.ValueInt64()))
		}

		deployAccessLevelsOption[i] = deployAccessLevelOptions
	}
	options.DeployAccessLevels = &deployAccessLevelsOption

	// approval rules
	approvalRulesOption := make([]*gitlab.EnvironmentApprovalRuleOptions, len(rules))
	for i, v := range rules {
		approvalRuleOptions := &gitlab.EnvironmentApprovalRuleOptions{}

		if !v.AccessLevel.IsNull() && v.AccessLevel.ValueString() != "" {
			approvalRuleOptions.AccessLevel = gitlab.Ptr(api.AccessLevelNameToValue[v.AccessLevel.ValueString()])
		}
		if !v.UserId.IsNull() && v.UserId.ValueInt64() != 0 {
			approvalRuleOptions.UserID = gitlab.Ptr(int(v.UserId.ValueInt64()))
		}
		if !v.GroupId.IsNull() && v.GroupId.ValueInt64() != 0 {
			approvalRuleOptions.GroupID = gitlab.Ptr(int(v.GroupId.ValueInt64()))
		}
		if !v.GroupInheritanceType.IsNull() && v.GroupInheritanceType.ValueInt64() != 0 {
			approvalRuleOptions.GroupInheritanceType = gitlab.Ptr(int(v.GroupInheritanceType.ValueInt64()))
		}
		if !v.RequiredApprovals.IsNull() && v.RequiredApprovals.ValueInt64() != 0 {
			approvalRuleOptions.RequiredApprovalCount = gitlab.Ptr(int(v.RequiredApprovals.ValueInt64()))
		}

		approvalRulesOption[i] = approvalRuleOptions
	}
	options.ApprovalRules = &approvalRulesOption

	tflog.Debug(ctx, "Creating protected environment with options", map[string]any{
		"data":      data,
		"projectId": projectID,
		"name":      environmentName,
		"options":   options,
	})

	// Protect environment
	protectedEnvironment, _, err := r.client.ProtectedEnvironments.ProtectRepositoryEnvironments(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The protected environment feature is not available on this project. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to protect environment: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, "Protected Environment before state is persisted", map[string]any{
		"data":        data,
		"projectId":   projectID,
		"name":        environmentName,
		"environment": protectedEnvironment,
	})

	// Create resource ID and persist in state model
	data.Id = types.StringValue(utils.BuildTwoPartID(&projectID, &protectedEnvironment.Name))

	// persist API response in state model
	r.protectedEnvironmentToStateModel(ctx, resp.Diagnostics, projectID, protectedEnvironment, data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabProjectProtectedEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectProtectedEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	projectID, environmentName, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<environment-name>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	// Read environment protection
	protectedEnvironment, _, err := r.client.ProtectedEnvironments.GetProtectedEnvironment(projectID, environmentName, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "protected environment does not exist, removing from state", map[string]any{
				"project": projectID, "environment": environmentName,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read protected environment details: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.protectedEnvironmentToStateModel(ctx, resp.Diagnostics, projectID, protectedEnvironment, data)

	tflog.Debug(ctx, "Protected Environment when state is being read", map[string]any{
		"project": projectID,
		"name":    environmentName,
		"data":    data,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Updates updates the resource in-place.
func (r *gitlabProjectProtectedEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabProjectProtectedEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	rules := make([]*gitlabProjectProtectedEnvironmentApprovalRuleModel, len(data.ApprovalRules.Elements()))
	data.ApprovalRules.ElementsAs(ctx, &rules, true)

	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	projectID := data.Project.ValueString()
	environmentName := data.Environment.ValueString()

	// Retrieve the protected environment (to know which deploy/approval rules to remove)
	protectedEnvironment, _, err := r.client.ProtectedEnvironments.GetProtectedEnvironment(projectID, environmentName, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The protected environment feature is not available on this project. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update protected environment details: %s", err.Error()))
		return
	}

	// configure GitLab API call
	options := &gitlab.UpdateProtectedEnvironmentsOptions{
		Name: gitlab.Ptr(environmentName),
	}

	// deploy access levels
	deployAccessLevels := make([]*gitlabProjectProtectedEnvironmentDeployAccessLevelModel, 0, len(data.DeployAccessLevels.Elements()))
	resp.Diagnostics.Append(data.DeployAccessLevels.ElementsAs(ctx, &deployAccessLevels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deployAccessLevelsOption := make([]*gitlab.UpdateEnvironmentAccessOptions, 0)
	for _, v := range deployAccessLevels {
		deployAccessLevelOptions := &gitlab.UpdateEnvironmentAccessOptions{}

		// the ID will be null when adding a new deploy rule via update
		if !v.ID.IsNull() && v.ID.ValueInt64() != 0 {
			deployAccessLevelOptions.ID = gitlab.Ptr(int(v.ID.ValueInt64()))
		}

		if !v.AccessLevel.IsNull() && v.AccessLevel.ValueString() != "" {
			deployAccessLevelOptions.AccessLevel = gitlab.Ptr(api.AccessLevelNameToValue[v.AccessLevel.ValueString()])
		}
		if !v.UserId.IsNull() && v.UserId.ValueInt64() != 0 {
			deployAccessLevelOptions.UserID = gitlab.Ptr(int(v.UserId.ValueInt64()))
		}
		if !v.GroupId.IsNull() && v.GroupId.ValueInt64() != 0 {
			deployAccessLevelOptions.GroupID = gitlab.Ptr(int(v.GroupId.ValueInt64()))
		}
		if !v.GroupInheritanceType.IsNull() && v.GroupInheritanceType.ValueInt64() != 0 {
			deployAccessLevelOptions.GroupInheritanceType = gitlab.Ptr(int(v.GroupInheritanceType.ValueInt64()))
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
			deployAccessLevelOptions := &gitlab.UpdateEnvironmentAccessOptions{
				ID:      gitlab.Ptr(v.ID),
				Destroy: gitlab.Ptr(true),
			}

			// Seems weird, but the API does validate that these values are present even
			// when destroyin
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
	approvalRulesOptionSlice := make([]*gitlab.UpdateEnvironmentApprovalRuleOptions, 0)
	for _, v := range rules {
		approvalRuleOptions := &gitlab.UpdateEnvironmentApprovalRuleOptions{}

		// the ID will be null when adding a new approval rule via update
		if !v.ID.IsNull() && v.ID.ValueInt64() != 0 {
			approvalRuleOptions.ID = gitlab.Ptr(int(v.ID.ValueInt64()))
		}

		if !v.AccessLevel.IsNull() && v.AccessLevel.ValueString() != "" {
			approvalRuleOptions.AccessLevel = gitlab.Ptr(api.AccessLevelNameToValue[v.AccessLevel.ValueString()])
		}
		if !v.UserId.IsNull() && v.UserId.ValueInt64() != 0 {
			approvalRuleOptions.UserID = gitlab.Ptr(int(v.UserId.ValueInt64()))
		}
		if !v.GroupId.IsNull() && v.GroupId.ValueInt64() != 0 {
			approvalRuleOptions.GroupID = gitlab.Ptr(int(v.GroupId.ValueInt64()))
		}
		if !v.GroupInheritanceType.IsNull() && v.GroupInheritanceType.ValueInt64() != 0 {
			approvalRuleOptions.GroupInheritanceType = gitlab.Ptr(int(v.GroupInheritanceType.ValueInt64()))
		}
		if !v.RequiredApprovals.IsNull() && v.RequiredApprovals.ValueInt64() != 0 {
			approvalRuleOptions.RequiredApprovalCount = gitlab.Ptr(int(v.RequiredApprovals.ValueInt64()))
		}

		approvalRulesOptionSlice = append(approvalRulesOptionSlice, approvalRuleOptions)
	}

	// Remove approval levels that aren't present in the config
	for _, v := range protectedEnvironment.ApprovalRules {
		isPresent := false
		for _, j := range rules {
			if v.ID == int(j.ID.ValueInt64()) {
				isPresent = true
				break
			}
		}

		// If the existing deploy isn't present, add it to the values to remove it
		if !isPresent {
			approvalRuleOptions := &gitlab.UpdateEnvironmentApprovalRuleOptions{
				ID:      gitlab.Ptr(v.ID),
				Destroy: gitlab.Ptr(true),
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

	tflog.Debug(ctx, "Updating protected environment with options", map[string]any{
		"project":          projectID,
		"options":          options,
		"environment_name": environmentName,
	})

	protectedEnvironment, _, err = r.client.ProtectedEnvironments.UpdateProtectedEnvironments(projectID, environmentName, options, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.Diagnostics.AddError(
				"GitLab Feature not available",
				fmt.Sprintf("The protected environment feature is not available on this project. Make sure it's part of an enterprise plan. Error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to update protected environment details: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, "Updating protected environment completed with options", map[string]any{
		"project":          projectID,
		"environment_name": environmentName,
		"result":           protectedEnvironment,
	})

	// Add data to state
	r.protectedEnvironmentToStateModel(ctx, resp.Diagnostics, projectID, protectedEnvironment, data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Deletes removes the resource.
func (r *gitlabProjectProtectedEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectProtectedEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	projectID, environmentName, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<environment-name>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	if _, err = r.client.ProtectedEnvironments.UnprotectEnvironment(projectID, environmentName, gitlab.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete protected environment: %s", err.Error()),
		)
	}
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabProjectProtectedEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabProjectProtectedEnvironmentResource) protectedEnvironmentToStateModel(ctx context.Context, existingDiag diag.Diagnostics, projectID string, protectedEnvironment *gitlab.ProtectedEnvironment, data *gitlabProjectProtectedEnvironmentResourceModel) {
	data.Project = types.StringValue(projectID)
	data.Environment = types.StringValue(protectedEnvironment.Name)

	deployAccessLevelsData := make([]gitlabProjectProtectedEnvironmentDeployAccessLevelModel, 0)
	for _, v := range protectedEnvironment.DeployAccessLevels {
		obj := *v
		deployAccessLevelData := gitlabProjectProtectedEnvironmentDeployAccessLevelModel{
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
		if obj.GroupInheritanceType != 0 {
			deployAccessLevelData.GroupInheritanceType = types.Int64Value(int64(obj.GroupInheritanceType))
		}

		deployAccessLevelsData = append(deployAccessLevelsData, deployAccessLevelData)
	}
	dalSetType, newDiag := types.SetValueFrom(ctx, deployAccessLevelSchema().NestedObject.Type(), deployAccessLevelsData)
	existingDiag.Append(newDiag...)
	data.DeployAccessLevels = dalSetType

	approvalRulesData := make([]gitlabProjectProtectedEnvironmentApprovalRuleModel, 0)
	for _, v := range protectedEnvironment.ApprovalRules {
		obj := *v
		approvalRuleData := gitlabProjectProtectedEnvironmentApprovalRuleModel{
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
		if obj.GroupInheritanceType != 0 {
			approvalRuleData.GroupInheritanceType = types.Int64Value(int64(obj.GroupInheritanceType))
		}
		if obj.RequiredApprovalCount != 0 {
			approvalRuleData.RequiredApprovals = types.Int64Value(int64(obj.RequiredApprovalCount))
		}

		approvalRulesData = append(approvalRulesData, approvalRuleData)
	}
	arSetType, newDiag := types.ListValueFrom(ctx, approvalRuleSchema().NestedObject.Type(), approvalRulesData)
	existingDiag.Append(newDiag...)
	data.ApprovalRules = arSetType
}
