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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                 = &gitlabBranchProtectionResource{}
	_ resource.ResourceWithConfigure    = &gitlabBranchProtectionResource{}
	_ resource.ResourceWithImportState  = &gitlabBranchProtectionResource{}
	_ resource.ResourceWithUpgradeState = &gitlabBranchProtectionResource{}
	_ resource.ResourceWithModifyPlan   = &gitlabBranchProtectionResource{}
)

func init() {
	registerResource(NewGitlabBranchProtectionResource)
}

// NewGitlabBranchProtectionResource is a helper function to simplify the provider implementation.
func NewGitlabBranchProtectionResource() resource.Resource {
	return &gitlabBranchProtectionResource{}
}

func (r *gitlabBranchProtectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_protection"
}

// gitlabBranchProtectionResource defines the resource implementation
type gitlabBranchProtectionResource struct {
	client *gitlab.Client
}

// gitlabBranchProtectionResourceModel describes the resource data model.
type gitlabBranchProtectionResourceModel struct {
	Id                        types.String `tfsdk:"id"`
	BranchProtectionId        types.Int64  `tfsdk:"branch_protection_id"`
	Project                   types.String `tfsdk:"project"`
	Branch                    types.String `tfsdk:"branch"`
	MergeAccessLevel          types.String `tfsdk:"merge_access_level"`
	PushAccessLevel           types.String `tfsdk:"push_access_level"`
	AllowForcePush            types.Bool   `tfsdk:"allow_force_push"`
	CodeOwnerApprovalRequired types.Bool   `tfsdk:"code_owner_approval_required"`
	AllowedToPush             types.Set    `tfsdk:"allowed_to_push"`
	AllowedToMerge            types.Set    `tfsdk:"allowed_to_merge"`
	AllowedToUnprotect        types.Set    `tfsdk:"allowed_to_unprotect"`
}

// gitlabBranchProtectionAllowedToObjectModel describes the generic allowed to block data model.
type gitlabBranchProtectionAllowedToObjectModel struct {
	AccessLevel            types.String `tfsdk:"access_level"`
	AccessLevelDescription types.String `tfsdk:"access_level_description"`
	UserId                 types.Int64  `tfsdk:"user_id"`
	GroupId                types.Int64  `tfsdk:"group_id"`
}

// gitlabBranchProtectionAllowedToPushObjectModel describes the allowed to push block data model.
type gitlabBranchProtectionAllowedToPushObjectModel struct {
	AccessLevel            types.String `tfsdk:"access_level"`
	AccessLevelDescription types.String `tfsdk:"access_level_description"`
	UserId                 types.Int64  `tfsdk:"user_id"`
	GroupId                types.Int64  `tfsdk:"group_id"`
	DeployKeyId            types.Int64  `tfsdk:"deploy_key_id"`
}

func (r *gitlabBranchProtectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.getV2Schema()
}

func (d *gitlabBranchProtectionResource) getV2Schema() schema.Schema {
	return schema.Schema{
		Version: 2,
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_branch_protection`" + ` resource manages the lifecycle of a protected branch of a repository.

~> **Branch Protection Behavior for the default branch**
   Depending on the GitLab instance, group or project setting the default branch of a project is created automatically by GitLab behind the scenes.
   Due to [some](https://gitlab.com/gitlab-org/terraform-provider-gitlab/issues/792) [limitations](https://discuss.hashicorp.com/t/ignore-the-order-of-a-complex-typed-list/42242) in the Terraform Provider SDK and the GitLab API,
   when creating a new project and trying to manage the branch protection setting for its default branch the ` + "`gitlab_branch_protection`" + ` resource will
   automatically take ownership of the default branch without an explicit import by unprotecting and properly protecting it again.
   Having multiple ` + "`gitlab_branch_protection`" + ` resources for the same project and default branch will result in them overriding each other - make sure to only have a single one.

~> The ` + "`allowed_to_push`" + `, ` + "`allowed_to_merge`" + `, ` + "`allowed_to_unprotect`" + ` and ` + "`code_owner_approval_required`" + ` attributes require a GitLab Enterprise instance.

~> The ` + "`merge_access_level`" + ` and ` + "`push_access_level`" + ` attributes are not available for GitLab Enterprise.  Use ` + "`allowed_to_merge`" + ` and ` + "`allowed_to_push`" + ` instead.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/protected_branches/)`),
		// remove the above two disclaimers about attributes in 19.3
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id:branch>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"branch_protection_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the branch protection (not the branch name).",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The id of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "Name of the branch.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"merge_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to merge. Valid values are: %s. Only available for CE instances.", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...)},
			},
			"push_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to push. Valid values are: %s. Only available for CE instances.", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...)},
			},
			"allow_force_push": schema.BoolAttribute{
				MarkdownDescription: "Can be set to true to allow users with push access to force push.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"code_owner_approval_required": schema.BoolAttribute{
				MarkdownDescription: "Can be set to true to require code owner approval before merging. Only available for Premium and Ultimate instances.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"allowed_to_push":      allowedToPushSchema(),
			"allowed_to_merge":     allowedToMergeSchema(),
			"allowed_to_unprotect": allowedToUnprotectSchema(),
		},
	}
}

func allowedToSchema(action string, validValues []string) schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		MarkdownDescription: fmt.Sprintf("Array of %s access levels/users/groups allowed for the protected branch. Only available for Premium and Ultimate instances.", action),
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"access_level": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Access level allowed to perform the relevant action. Mutually exclusive with `group_id` and `user_id`. Valid values are: %s.",
						utils.RenderValueListForDocs(validValues)),
					Computed: true,
					Optional: true,
					Validators: []validator.String{
						stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("user_id"), path.MatchRelative().AtParent().AtName("group_id")),
						stringvalidator.OneOf(validValues...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of access level.",
					Computed:            true,
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `group_id` and `access_level`.",
					Optional:            true,
					Validators: []validator.Int64{
						int64validator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("group_id"), path.MatchRelative().AtParent().AtName("access_level")),
					},
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `user_id` and `access_level`.",
					Optional:            true,
					Validators: []validator.Int64{
						int64validator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("user_id"), path.MatchRelative().AtParent().AtName("access_level")),
					},
				},
			},
		},
	}
}

func allowedToPushSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		MarkdownDescription: "Array of push access levels/users/groups/deploy keys allowed for the protected branch. Only available for Premium and Ultimate instances.",
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"access_level": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Access level allowed to perform the relevant action. Mutually exclusive with `deploy_key_id`, `group_id`, and `user_id`. Valid values are: %s.",
						utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
					Computed: true,
					Optional: true,
					Validators: []validator.String{
						stringvalidator.ExactlyOneOf(
							path.MatchRelative().AtParent().AtName("user_id"),
							path.MatchRelative().AtParent().AtName("group_id"),
							path.MatchRelative().AtParent().AtName("deploy_key_id"),
						),
						stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of access level.",
					Computed:            true,
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `deploy_key_id`, `group_id`, and `access_level`.",
					Optional:            true,
					Validators: []validator.Int64{
						int64validator.ExactlyOneOf(
							path.MatchRelative().AtParent().AtName("group_id"),
							path.MatchRelative().AtParent().AtName("access_level"),
							path.MatchRelative().AtParent().AtName("deploy_key_id"),
						),
					},
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `deploy_key_id`, `user_id`, and `access_level`.",
					Optional:            true,
					Validators: []validator.Int64{
						int64validator.ExactlyOneOf(
							path.MatchRelative().AtParent().AtName("user_id"),
							path.MatchRelative().AtParent().AtName("access_level"),
							path.MatchRelative().AtParent().AtName("deploy_key_id"),
						),
					},
				},
				"deploy_key_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab deploy key allowed to perform the relevant action. Mutually exclusive with `user_id`, `group_id`, and `access_level`. This field is read-only until Gitlab 17.5.",
					Optional:            true,
					Validators: []validator.Int64{
						int64validator.ExactlyOneOf(
							path.MatchRelative().AtParent().AtName("group_id"),
							path.MatchRelative().AtParent().AtName("access_level"),
							path.MatchRelative().AtParent().AtName("user_id"),
						),
					},
				},
			},
		},
	}
}

func allowedToMergeSchema() schema.SetNestedAttribute {
	return allowedToSchema("merge", api.ValidProtectedBranchTagAccessLevelNames)
}

func allowedToUnprotectSchema() schema.SetNestedAttribute {
	return allowedToSchema("unprotect", api.ValidProtectedBranchUnprotectAccessLevelNames)
}

// Configure adds the provider configured client to the resource.
func (r *gitlabBranchProtectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabBranchProtectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Retrieve the plan data to start with
	var planData, configData *gitlabBranchProtectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	// Now retrieve the `config` values, because we need to get the *_access_levels from the config
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)

	if planData == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no check for license is needed")
		return
	}

	// check if running against EE
	isEE, err := utils.IsRunningInEEContext(r.client)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get gitlab server metadata details: %s", err.Error()))
		return
	}

	if !isEE {
		// check for disallowed attributes when running for CE
		if !configData.AllowedToPush.IsNull() && !configData.AllowedToPush.IsUnknown() {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_push`", "Enterprise license required")
		}
		if !configData.AllowedToMerge.IsNull() && !configData.AllowedToMerge.IsUnknown() {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_merge`", "Enterprise license required")
		}
		if !configData.AllowedToUnprotect.IsNull() && !configData.AllowedToUnprotect.IsUnknown() {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_unprotect`", "Enterprise license required")
		}
		if configData.CodeOwnerApprovalRequired.ValueBool() {
			resp.Diagnostics.AddError("feature unavailable `code_owner_approval_required`", "Enterprise license required")
		}
	} else {
		// check for disallowed attributes when running for EE
		if !configData.PushAccessLevel.IsNull() && !configData.PushAccessLevel.IsUnknown() {
			resp.Diagnostics.AddError("feature unavailable `push_access_level`", "Use `allowed_to_push` instead")
		}
		if !configData.MergeAccessLevel.IsNull() && !configData.MergeAccessLevel.IsUnknown() {
			resp.Diagnostics.AddError("feature unavailable `merge_access_level`", "Use `allowed_to_merge` instead")
		}
	}

}

// Create creates a new upstream resources and adds it into the Terraform state.
func (r *gitlabBranchProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabBranchProtectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// check if running against EE
	isEE, err := utils.IsRunningInEEContext(r.client)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get gitlab server metadata details: %s", err.Error()))
		return
	}

	// local copies of plan arguments
	projectID := data.Project.ValueString()
	branch := data.Branch.ValueString()

	// call protected repository branch, read Gitlab API to detect if given branch is project default branch and requires default
	// branch protection rule removal
	existingProtectedBranch, _, err := r.client.ProtectedBranches.GetProtectedBranch(projectID, branch, gitlab.WithContext(ctx))
	if err == nil {
		projectDetails, _, err := r.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read project details: %s", err.Error()))
			return
		}

		// Gitlab automatically creates branch protection rule for repository default branch. This results in protection rule existence
		// which is not managed by terraform. Then each branch protection rule creation attempt will fail. To fix that it is required
		// to firstly remove already existing rule to be able to add provider managed one.
		if projectDetails.DefaultBranch == branch {
			tflog.Debug(ctx, fmt.Sprintf("This branch protection is for the default branch %q in project %q! It is always "+
				"created by gitlab so firstly we heave to unprotect it, because it's not editable ...!", projectID, branch))

			_, err := r.client.ProtectedBranches.UnprotectRepositoryBranches(projectID, branch, gitlab.WithContext(ctx))
			if err != nil {
				resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Failed to unprotect default branch %q in "+
					"project %q while trying to 'import' it: %v", branch, projectID, err.Error()))
				return
			}
		} else {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("protected branch %q on project %q already exists: %+v",
				branch, projectID, *existingProtectedBranch))
			return
		}
	}

	// configure GitLab protected branch creation API call
	options := gitlab.ProtectRepositoryBranchesOptions{
		Name:           gitlab.Ptr(data.Branch.ValueString()),
		AllowForcePush: gitlab.Ptr(data.AllowForcePush.ValueBool()),
	}

	if !isEE {
		// set CE only attributes
		if !data.PushAccessLevel.IsNull() && !data.PushAccessLevel.IsUnknown() {
			pushAccessLevel := api.AccessLevelNameToValue[data.PushAccessLevel.ValueString()]
			options.PushAccessLevel = &pushAccessLevel
		}

		if !data.MergeAccessLevel.IsNull() && !data.MergeAccessLevel.IsUnknown() {
			mergeAccessLevel := api.AccessLevelNameToValue[data.MergeAccessLevel.ValueString()]
			options.MergeAccessLevel = &mergeAccessLevel
		}
	} else {
		// define EE only attributes if EE
		allowedToPush := make([]*gitlabBranchProtectionAllowedToPushObjectModel, 0, len(data.AllowedToPush.Elements()))
		resp.Diagnostics.Append(data.AllowedToPush.ElementsAs(ctx, &allowedToPush, true)...)
		if resp.Diagnostics.HasError() {
			return
		}

		allowedToPushOption := generateAllowedToPushStateToAccessLevels([]*gitlab.BranchAccessDescription{}, allowedToPush)

		allowedToMerge := make([]*gitlabBranchProtectionAllowedToObjectModel, 0, len(data.AllowedToMerge.Elements()))
		resp.Diagnostics.Append(data.AllowedToMerge.ElementsAs(ctx, &allowedToMerge, true)...)
		if resp.Diagnostics.HasError() {
			return
		}

		allowedToMergeOption := generateAllowedToStateToAccessLevels([]*gitlab.BranchAccessDescription{}, allowedToMerge)

		allowedToUnprotect := make([]*gitlabBranchProtectionAllowedToObjectModel, 0, len(data.AllowedToUnprotect.Elements()))
		resp.Diagnostics.Append(data.AllowedToUnprotect.ElementsAs(ctx, &allowedToUnprotect, true)...)
		if resp.Diagnostics.HasError() {
			return
		}

		allowedToUnprotectOption := generateAllowedToStateToAccessLevels([]*gitlab.BranchAccessDescription{}, allowedToUnprotect)

		options.AllowedToPush = &allowedToPushOption
		options.AllowedToMerge = &allowedToMergeOption
		options.AllowedToUnprotect = &allowedToUnprotectOption

		options.CodeOwnerApprovalRequired = data.CodeOwnerApprovalRequired.ValueBoolPointer()
	}

	// call Gitlab protected repository branch creation API
	protectedBranch, _, err := r.client.ProtectedBranches.ProtectRepositoryBranches(projectID, &options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to protect repository branch: %s", err.Error()))
		return
	}

	// Create resource ID and persist in state model
	data.Id = types.StringValue(utils.BuildTwoPartID(&projectID, &protectedBranch.Name))

	// persist API response in state model
	r.protectedBranchToStateModel(ctx, resp.Diagnostics, projectID, protectedBranch, data, isEE)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// Log the creation of the resource
	tflog.Debug(ctx, "Created a protected branch", map[string]any{
		"project_id": data.Project.ValueString(), "branch": data.Branch.ValueString(),
	})
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabBranchProtectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabBranchProtectionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	projectID, branch, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project-id>:<branch>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	// call protected repository branch, read Gitlab API
	protectedBranch, _, err := r.client.ProtectedBranches.GetProtectedBranch(projectID, branch, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "protected branch does not exist, removing from state", map[string]any{
				"project_id": projectID, "branch": branch,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read protected branch details: %s", err.Error()))
		return
	}

	// check if running against EE as we only want to set EE only attributes when running against EE
	isEE, err := utils.IsRunningInEEContext(r.client)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get gitlab server metadata details: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.protectedBranchToStateModel(ctx, resp.Diagnostics, projectID, protectedBranch, data, isEE)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Updates updates the resource in-place.
func (r *gitlabBranchProtectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabBranchProtectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// check if running against EE
	isEE, err := utils.IsRunningInEEContext(r.client)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get gitlab server metadata details: %s", err.Error()))
		return
	}

	// local copies of plan arguments
	projectID := data.Project.ValueString()
	branch := data.Branch.ValueString()

	// call read, protected repository branch API to retrieve current unprotected access levels
	protectedBranch, _, err := r.client.ProtectedBranches.GetProtectedBranch(projectID, branch, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "protected branch does not exist, removing from state", map[string]any{
				"project_id": projectID, "branch": branch,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read protected branch details: %s", err.Error()))
		return
	}

	// configure protect repository branch update GitLab API call
	options := gitlab.UpdateProtectedBranchOptions{
		Name:           gitlab.Ptr(branch),
		AllowForcePush: gitlab.Ptr(data.AllowForcePush.ValueBool()),
	}

	// define EE only attributes if EE
	if isEE {

		// determine what's new and what needs to be removed
		allowedToPush := make([]*gitlabBranchProtectionAllowedToPushObjectModel, 0, len(data.AllowedToPush.Elements()))
		resp.Diagnostics.Append(data.AllowedToPush.ElementsAs(ctx, &allowedToPush, true)...)
		if resp.Diagnostics.HasError() {
			return
		}

		allowedToPushOption := generateAllowedToPushStateToAccessLevels(protectedBranch.PushAccessLevels, allowedToPush)

		allowedToMerge := make([]*gitlabBranchProtectionAllowedToObjectModel, 0, len(data.AllowedToMerge.Elements()))
		resp.Diagnostics.Append(data.AllowedToMerge.ElementsAs(ctx, &allowedToMerge, true)...)
		if resp.Diagnostics.HasError() {
			return
		}

		allowedToMergeOption := generateAllowedToStateToAccessLevels(protectedBranch.MergeAccessLevels, allowedToMerge)

		allowedToUnprotect := make([]*gitlabBranchProtectionAllowedToObjectModel, 0, len(data.AllowedToUnprotect.Elements()))
		resp.Diagnostics.Append(data.AllowedToUnprotect.ElementsAs(ctx, &allowedToUnprotect, true)...)
		if resp.Diagnostics.HasError() {
			return
		}

		allowedToUnprotectOption := generateAllowedToStateToAccessLevels(protectedBranch.UnprotectAccessLevels, allowedToUnprotect)

		options.AllowedToPush = &allowedToPushOption
		options.AllowedToMerge = &allowedToMergeOption
		options.AllowedToUnprotect = &allowedToUnprotectOption

		options.CodeOwnerApprovalRequired = data.CodeOwnerApprovalRequired.ValueBoolPointer()
	}

	// call Gitlab protected repository branch update API
	updatedProtectedBranch, _, err := r.client.ProtectedBranches.UpdateProtectedBranch(projectID, branch, &options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update protected repository branch: %s", err.Error()))
		return
	}

	// Create resource ID and persist in state model
	data.Id = types.StringValue(utils.BuildTwoPartID(&projectID, &branch))

	// persist API response in state model
	r.protectedBranchToStateModel(ctx, resp.Diagnostics, projectID, updatedProtectedBranch, data, isEE)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// Log the update of the resource
	tflog.Debug(ctx, "Updated a protected branch", map[string]any{
		"project_id": projectID, "branch": branch,
	})
}

// Deletes removes the resource.
func (r *gitlabBranchProtectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabBranchProtectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// call Gitlab unprotect repository branch API
	_, err := r.client.ProtectedBranches.UnprotectRepositoryBranches(data.Project.ValueString(), data.Branch.ValueString(), gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete gitlab branch protection", err.Error())
		return
	}

	tflog.Debug(ctx, "Delete gitlab protected branch", map[string]any{
		"project_id": data.Project.ValueString(), "branch": data.Branch.ValueString(),
	})
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabBranchProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabBranchProtectionResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	// The v0 schema. Needed so the pointer can be used.
	schemaV0 := d.getV0Schema()
	// The v1 schema. Needed so the pointer can be used.
	schemaV1 := d.getV1Schema()

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schemaV0,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data *gitlabBranchProtectionResourceModelV0

				// Read Terraform plan data into the model
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Move all data to the new struct values. Note: Only values that were present at v0 time
				// are included here, because they couldn't exist as known in the old struct.
				project_id := data.Project.ValueString()
				branch := data.Branch.ValueString()
				resource_id := utils.BuildTwoPartID(&project_id, &branch)

				newData := &gitlabBranchProtectionResourceModel{
					Id:                        types.StringValue(resource_id),
					BranchProtectionId:        data.BranchProtectionId,
					Project:                   data.Project,
					Branch:                    data.Branch,
					MergeAccessLevel:          data.MergeAccessLevel,
					PushAccessLevel:           data.PushAccessLevel,
					AllowForcePush:            data.AllowForcePush,
					CodeOwnerApprovalRequired: data.CodeOwnerApprovalRequired,
				}

				migratedAllowedToPush := migrateAllowedToBlock(data.AllowedToPush)
				allowedToPushSetType, diag := types.SetValueFrom(ctx, allowedToPushSchema().NestedObject.Type(), migratedAllowedToPush)
				resp.Diagnostics.Append(diag...)
				newData.AllowedToPush = allowedToPushSetType

				allowedToMergeSetType, diag := types.SetValueFrom(ctx, allowedToMergeSchema().NestedObject.Type(), data.AllowedToMerge)
				resp.Diagnostics.Append(diag...)
				newData.AllowedToMerge = allowedToMergeSetType

				allowedToUnprotectSetType, diag := types.SetValueFrom(ctx, allowedToUnprotectSchema().NestedObject.Type(), data.AllowedToUnprotect)
				resp.Diagnostics.Append(diag...)
				newData.AllowedToUnprotect = allowedToUnprotectSetType

				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)

				tflog.Debug(ctx, "Upgraded gitlab_branch_protection resource from v0 to v2 version", map[string]any{
					"project_id": data.Project.ValueString(), "branch": data.Branch.ValueString(),
				})
			},
		},
		1: {
			PriorSchema: &schemaV1,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data *gitlabBranchProtectionResourceModelV1

				// Read Terraform plan data into the model
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Move all data to the new struct values. Note: Only values that were present at v1 time
				// are included here, because they couldn't exist as known in the old struct.
				project_id := data.Project.ValueString()
				branch := data.Branch.ValueString()
				resource_id := utils.BuildTwoPartID(&project_id, &branch)

				newData := &gitlabBranchProtectionResourceModel{
					Id:                        types.StringValue(resource_id),
					BranchProtectionId:        data.BranchProtectionId,
					Project:                   data.Project,
					Branch:                    data.Branch,
					MergeAccessLevel:          data.MergeAccessLevel,
					PushAccessLevel:           data.PushAccessLevel,
					AllowForcePush:            data.AllowForcePush,
					CodeOwnerApprovalRequired: data.CodeOwnerApprovalRequired,
				}

				allowedToPushSetType, diag := types.SetValueFrom(ctx, allowedToPushSchema().NestedObject.Type(), data.AllowedToPush)
				resp.Diagnostics.Append(diag...)
				newData.AllowedToPush = allowedToPushSetType

				allowedToMergeSetType, diag := types.SetValueFrom(ctx, allowedToMergeSchema().NestedObject.Type(), data.AllowedToMerge)
				resp.Diagnostics.Append(diag...)
				newData.AllowedToMerge = allowedToMergeSetType

				allowedToUnprotectSetType, diag := types.SetValueFrom(ctx, allowedToUnprotectSchema().NestedObject.Type(), data.AllowedToUnprotect)
				resp.Diagnostics.Append(diag...)
				newData.AllowedToUnprotect = allowedToUnprotectSetType

				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)

				tflog.Debug(ctx, "Upgraded gitlab_branch_protection resource from v1 to v2 version", map[string]any{
					"project_id": data.Project.ValueString(), "branch": data.Branch.ValueString(),
				})
			},
		},
	}
}

// When accessing protectedBranch.*AccessLevels argument by gitlab API 'ProtectedBranches.GetProtectedBranch' list contains
// elements with AccessLevel field defined and elements connected with user_id&group_id which does not have it. Then it can
// result in setting AccessLevel for null value which will result in error. To prohibit such cases only elements with
// AccessLevel field set are returned
func firstValidAccessLevel(descriptions []*gitlab.BranchAccessDescription) (*gitlab.AccessLevelValue, error) {
	for _, description := range descriptions {
		if description.UserID != 0 || description.GroupID != 0 || description.DeployKeyID != 0 {
			continue
		}
		return &description.AccessLevel, nil
	}

	return nil, fmt.Errorf("no valid access level found")
}

// Generates slice of allowed_to_merge/unprotect settings, marking ones not provided in the config as needing to be removed
func generateAllowedToStateToAccessLevels(currentAllowedTos []*gitlab.BranchAccessDescription, plannedAllowedTos []*gitlabBranchProtectionAllowedToObjectModel) []*gitlab.BranchPermissionOptions {
	finalAllowedTo := []*gitlab.BranchPermissionOptions{}
	// detect entities to be created
	for _, plannedAllowedTo := range plannedAllowedTos {
		var allowedToBranchPermissionOptionData *gitlab.BranchPermissionOptions = populateBranchPermissionOptionsData(currentAllowedTos, plannedAllowedTo)
		if allowedToBranchPermissionOptionData != nil {
			finalAllowedTo = append(finalAllowedTo, allowedToBranchPermissionOptionData)
		}
	}

	// detect entities to be removed
	for _, validCurrentAllowedTo := range currentAllowedTos {
		requireRemoval := true
		for _, plannedAllowedTo := range plannedAllowedTos {
			// if element exists in planned values skip removal
			if (plannedAllowedTo.UserId.ValueInt64() == int64(validCurrentAllowedTo.UserID)) &&
				(plannedAllowedTo.GroupId.ValueInt64() == int64(validCurrentAllowedTo.GroupID)) &&
				(plannedAllowedTo.AccessLevel.IsUnknown() || plannedAllowedTo.AccessLevel.IsNull() ||
					(plannedAllowedTo.AccessLevel.ValueString() == api.AccessLevelValueToName[validCurrentAllowedTo.AccessLevel])) {
				requireRemoval = false
				continue
			}
		}

		if requireRemoval {
			requireRemovalBranchPermissionOptionData := &gitlab.BranchPermissionOptions{
				ID:      &validCurrentAllowedTo.ID,
				Destroy: gitlab.Ptr(true),
			}
			finalAllowedTo = append(finalAllowedTo, requireRemovalBranchPermissionOptionData)
		}
	}

	return finalAllowedTo
}

// Generates slice of allowed_to_push settings, marking ones not provided in the config as needing to be removed
func generateAllowedToPushStateToAccessLevels(currentAllowedTos []*gitlab.BranchAccessDescription, plannedAllowedTos []*gitlabBranchProtectionAllowedToPushObjectModel) []*gitlab.BranchPermissionOptions {
	finalAllowedTo := []*gitlab.BranchPermissionOptions{}
	// detect entities to be created
	for _, plannedAllowedTo := range plannedAllowedTos {
		var allowedToBranchPermissionOptionData *gitlab.BranchPermissionOptions = populateBranchPermissionOptionsDataForPush(currentAllowedTos, plannedAllowedTo)
		if allowedToBranchPermissionOptionData != nil {
			finalAllowedTo = append(finalAllowedTo, allowedToBranchPermissionOptionData)
		}
	}

	// detect entities to be removed
	for _, validCurrentAllowedTo := range currentAllowedTos {
		requireRemoval := true
		for _, plannedAllowedTo := range plannedAllowedTos {
			// if element exists in planned values skip removal
			if (plannedAllowedTo.UserId.ValueInt64() == int64(validCurrentAllowedTo.UserID)) &&
				(plannedAllowedTo.GroupId.ValueInt64() == int64(validCurrentAllowedTo.GroupID)) &&
				(plannedAllowedTo.DeployKeyId.ValueInt64() == int64(validCurrentAllowedTo.DeployKeyID)) &&
				(plannedAllowedTo.AccessLevel.IsUnknown() || plannedAllowedTo.AccessLevel.IsNull() ||
					(plannedAllowedTo.AccessLevel.ValueString() == api.AccessLevelValueToName[validCurrentAllowedTo.AccessLevel])) {
				requireRemoval = false
				continue
			}
		}

		if requireRemoval {
			requireRemovalBranchPermissionOptionData := &gitlab.BranchPermissionOptions{
				ID:      &validCurrentAllowedTo.ID,
				Destroy: gitlab.Ptr(true),
			}
			finalAllowedTo = append(finalAllowedTo, requireRemovalBranchPermissionOptionData)
		}
	}

	return finalAllowedTo
}

func populateBranchPermissionOptionsData(currentAllowedTos []*gitlab.BranchAccessDescription, allowedTo *gitlabBranchProtectionAllowedToObjectModel) *gitlab.BranchPermissionOptions {
	var allowedToBranchPermissionOptionData *gitlab.BranchPermissionOptions
	requireCreation := true
	// detect if element already exists
	for _, currentAllowedTo := range currentAllowedTos {
		// if element already exists skip creation
		if allowedTo.AccessLevel == types.StringValue(api.AccessLevelValueToName[currentAllowedTo.AccessLevel]) ||
			!allowedTo.UserId.IsNull() && allowedTo.UserId.ValueInt64() == int64(currentAllowedTo.UserID) ||
			!allowedTo.GroupId.IsNull() && allowedTo.GroupId.ValueInt64() == int64(currentAllowedTo.GroupID) {
			requireCreation = false
		}
	}

	if requireCreation {
		allowedToBranchPermissionOptionData = &gitlab.BranchPermissionOptions{}

		if !allowedTo.AccessLevel.IsNull() && allowedTo.AccessLevel.ValueString() != "" {
			allowedToBranchPermissionOptionData.AccessLevel = gitlab.Ptr(api.AccessLevelNameToValue[allowedTo.AccessLevel.ValueString()])
		}
		if !allowedTo.UserId.IsNull() && allowedTo.UserId.ValueInt64() != 0 {
			allowedToBranchPermissionOptionData.UserID = gitlab.Ptr(allowedTo.UserId.ValueInt64())
		}
		if !allowedTo.GroupId.IsNull() && allowedTo.GroupId.ValueInt64() != 0 {
			allowedToBranchPermissionOptionData.GroupID = gitlab.Ptr(allowedTo.GroupId.ValueInt64())
		}
	}

	return allowedToBranchPermissionOptionData
}

func populateBranchPermissionOptionsDataForPush(currentAllowedTos []*gitlab.BranchAccessDescription, allowedTo *gitlabBranchProtectionAllowedToPushObjectModel) *gitlab.BranchPermissionOptions {
	var allowedToBranchPermissionOptionData *gitlab.BranchPermissionOptions
	requireCreation := true
	// detect if element already exists
	for _, currentAllowedTo := range currentAllowedTos {
		// if element already exists skip creation
		if allowedTo.AccessLevel == types.StringValue(api.AccessLevelValueToName[currentAllowedTo.AccessLevel]) ||
			!allowedTo.UserId.IsNull() && allowedTo.UserId.ValueInt64() == int64(currentAllowedTo.UserID) ||
			!allowedTo.GroupId.IsNull() && allowedTo.GroupId.ValueInt64() == int64(currentAllowedTo.GroupID) ||
			!allowedTo.DeployKeyId.IsNull() && allowedTo.DeployKeyId.ValueInt64() == int64(currentAllowedTo.DeployKeyID) {
			requireCreation = false
		}
	}

	if requireCreation {
		allowedToBranchPermissionOptionData = &gitlab.BranchPermissionOptions{}

		if !allowedTo.AccessLevel.IsNull() && allowedTo.AccessLevel.ValueString() != "" {
			allowedToBranchPermissionOptionData.AccessLevel = gitlab.Ptr(api.AccessLevelNameToValue[allowedTo.AccessLevel.ValueString()])
		}
		if !allowedTo.UserId.IsNull() && allowedTo.UserId.ValueInt64() != 0 {
			allowedToBranchPermissionOptionData.UserID = gitlab.Ptr(allowedTo.UserId.ValueInt64())
		}
		if !allowedTo.GroupId.IsNull() && allowedTo.GroupId.ValueInt64() != 0 {
			allowedToBranchPermissionOptionData.GroupID = gitlab.Ptr(allowedTo.GroupId.ValueInt64())
		}
		if !allowedTo.DeployKeyId.IsNull() && allowedTo.DeployKeyId.ValueInt64() != 0 {
			allowedToBranchPermissionOptionData.DeployKeyID = gitlab.Ptr(allowedTo.DeployKeyId.ValueInt64())
		}
	}

	return allowedToBranchPermissionOptionData
}

func populateAllowedToToStateModel(accessLevels []*gitlab.BranchAccessDescription) []*gitlabBranchProtectionAllowedToObjectModel {
	allowedToData := make([]*gitlabBranchProtectionAllowedToObjectModel, len(accessLevels))
	for i, v := range accessLevels {
		allowedToModel := gitlabBranchProtectionAllowedToObjectModel{
			AccessLevelDescription: types.StringValue(v.AccessLevelDescription),
		}

		if v.UserID != 0 {
			allowedToModel.UserId = types.Int64Value(int64(v.UserID))
		} else if v.GroupID != 0 {
			allowedToModel.GroupId = types.Int64Value(int64(v.GroupID))
		} else {
			// access level should come back as null if not set, but it comes back as int
			// so we check if the other values are set first before setting access level
			allowedToModel.AccessLevel = types.StringValue(api.AccessLevelValueToName[v.AccessLevel])
		}

		allowedToData[i] = &allowedToModel
	}

	return allowedToData
}

func populateAllowedToPushToStateModel(accessLevels []*gitlab.BranchAccessDescription) []*gitlabBranchProtectionAllowedToPushObjectModel {
	allowedToData := make([]*gitlabBranchProtectionAllowedToPushObjectModel, len(accessLevels))
	for i, v := range accessLevels {
		allowedToModel := gitlabBranchProtectionAllowedToPushObjectModel{
			AccessLevelDescription: types.StringValue(v.AccessLevelDescription),
		}

		if v.UserID != 0 {
			allowedToModel.UserId = types.Int64Value(int64(v.UserID))
		} else if v.GroupID != 0 {
			allowedToModel.GroupId = types.Int64Value(int64(v.GroupID))
		} else if v.DeployKeyID != 0 {
			allowedToModel.DeployKeyId = types.Int64Value(int64(v.DeployKeyID))
		} else {
			allowedToModel.AccessLevel = types.StringValue(api.AccessLevelValueToName[v.AccessLevel])
		}

		allowedToData[i] = &allowedToModel
	}

	return allowedToData
}

func (r *gitlabBranchProtectionResource) protectedBranchToStateModel(ctx context.Context, existingDiag diag.Diagnostics, projectID string, protectedBranch *gitlab.ProtectedBranch, data *gitlabBranchProtectionResourceModel, isEE bool) {
	data.Project = types.StringValue(projectID)
	data.Branch = types.StringValue(protectedBranch.Name)
	data.BranchProtectionId = types.Int64Value(int64(protectedBranch.ID))

	data.AllowForcePush = types.BoolValue(protectedBranch.AllowForcePush)

	// set PushAccessLevel in state
	// if license model is not enterpise make sure to set PushAccessLevel to first valid level returned by the API
	if !isEE {
		pushAccessLevel, err := firstValidAccessLevel(protectedBranch.PushAccessLevels)
		if err == nil {
			data.PushAccessLevel = types.StringValue(api.AccessLevelValueToName[*pushAccessLevel])
		}
		mergeAccessLevel, err := firstValidAccessLevel(protectedBranch.MergeAccessLevels)
		if err == nil {
			data.MergeAccessLevel = types.StringValue(api.AccessLevelValueToName[*mergeAccessLevel])
		}

		data.AllowedToMerge = types.SetNull(allowedToMergeSchema().NestedObject.Type())
		data.AllowedToPush = types.SetNull(allowedToPushSchema().NestedObject.Type())
		data.AllowedToUnprotect = types.SetNull(allowedToUnprotectSchema().NestedObject.Type())
		data.CodeOwnerApprovalRequired = types.BoolValue(false)
	} else {
		allowedToPushData := populateAllowedToPushToStateModel(protectedBranch.PushAccessLevels)
		allowedToPushSetType, diag := types.SetValueFrom(ctx, allowedToPushSchema().NestedObject.Type(), allowedToPushData)
		existingDiag.Append(diag...)
		data.AllowedToPush = allowedToPushSetType

		allowedToMergeData := populateAllowedToToStateModel(protectedBranch.MergeAccessLevels)
		allowedToMergeSetType, diag := types.SetValueFrom(ctx, allowedToMergeSchema().NestedObject.Type(), allowedToMergeData)
		existingDiag.Append(diag...)
		data.AllowedToMerge = allowedToMergeSetType

		allowedToUnprotectData := populateAllowedToToStateModel(protectedBranch.UnprotectAccessLevels)
		allowedToUnprotectSetType, diag := types.SetValueFrom(ctx, allowedToUnprotectSchema().NestedObject.Type(), allowedToUnprotectData)
		existingDiag.Append(diag...)
		data.AllowedToUnprotect = allowedToUnprotectSetType

		data.CodeOwnerApprovalRequired = types.BoolValue(protectedBranch.CodeOwnerApprovalRequired)

		data.PushAccessLevel = types.StringNull()
		data.MergeAccessLevel = types.StringNull()
	}

}

//////////////////////////////////////////////////////////////////
// 				resource schema version v0			         	//
//////////////////////////////////////////////////////////////////

// gitlabBranchProtectionResourceModelV0 describes the resource data model in version v0.
type gitlabBranchProtectionResourceModelV0 struct {
	Project                   types.String                                  `tfsdk:"project"`
	Branch                    types.String                                  `tfsdk:"branch"`
	MergeAccessLevel          types.String                                  `tfsdk:"merge_access_level"`
	PushAccessLevel           types.String                                  `tfsdk:"push_access_level"`
	UnprotectAccessLevel      types.String                                  `tfsdk:"unprotect_access_level"`
	AllowForcePush            types.Bool                                    `tfsdk:"allow_force_push"`
	CodeOwnerApprovalRequired types.Bool                                    `tfsdk:"code_owner_approval_required"`
	BranchProtectionId        types.Int64                                   `tfsdk:"branch_protection_id"`
	AllowedToPush             []*gitlabBranchProtectionAllowedToObjectModel `tfsdk:"allowed_to_push"`
	AllowedToMerge            []*gitlabBranchProtectionAllowedToObjectModel `tfsdk:"allowed_to_merge"`
	AllowedToUnprotect        []*gitlabBranchProtectionAllowedToObjectModel `tfsdk:"allowed_to_unprotect"`
}

// gitlabBranchProtectionResource describes the resource schema in version v0.
func (d *gitlabBranchProtectionResource) getV0Schema() schema.Schema {
	return schema.Schema{
		Version: 0,
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_branch_protection`" + ` resource manages the lifecycle of a protected branch of a repository.

~> **Branch Protection Behavior for the default branch**
   Depending on the GitLab instance, group or project setting the default branch of a project is created automatically by GitLab behind the scenes.
   Due to [some](https://gitlab.com/gitlab-org/terraform-provider-gitlab/issues/792) [limitations](https://discuss.hashicorp.com/t/ignore-the-order-of-a-complex-typed-list/42242) in the Terraform Provider SDK and the GitLab API,
   when creating a new project and trying to manage the branch protection setting for its default branch the ` + "`gitlab_branch_protection`" + ` resource will
   automatically take ownership of the default branch without an explicit import by unprotecting and properly protecting it again.
   Having multiple ` + "`gitlab_branch_protection`" + ` resources for the same project and default branch will result in them overriding each other - make sure to only have a single one.

~> The ` + "`allowed_to_push`" + `, ` + "`allowed_to_merge`" + `, ` + "`allowed_to_unprotect`" + `, ` + "`unprotect_access_level`" + ` and ` + "`code_owner_approval_required`" + ` attributes require a GitLab Enterprise instance.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/protected_branches/)`),

		Attributes: map[string]schema.Attribute{
			"branch_protection_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the branch protection (not the branch name).",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The id of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "Name of the branch.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"merge_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to merge. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...)},
			},
			"push_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to push. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...)},
			},
			"unprotect_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to unprotect. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidProtectedBranchUnprotectAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchUnprotectAccessLevelNames...)},
			},
			"allow_force_push": schema.BoolAttribute{
				MarkdownDescription: "Can be set to true to allow users with push access to force push.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"code_owner_approval_required": schema.BoolAttribute{
				MarkdownDescription: "Can be set to true to require code owner approval before merging. Only available for Premium and Ultimate instances.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"allowed_to_push":      schemaAllowedToObject("push", api.ValidProtectedBranchTagAccessLevelNames),
			"allowed_to_merge":     schemaAllowedToObject("merge", api.ValidProtectedBranchTagAccessLevelNames),
			"allowed_to_unprotect": schemaAllowedToObject("unprotect push", api.ValidProtectedBranchUnprotectAccessLevelNames),
		},
	}
}

func schemaAllowedToObject(action string, validValues []string) schema.Block {
	return schema.SetNestedBlock{
		MarkdownDescription: fmt.Sprintf("Array of access levels and user(s)/group(s) allowed to %s to protected branch.", action),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"access_level": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Access levels allowed to %s to protected branch. Valid values are: %s.", action,
						utils.RenderValueListForDocs(validValues)),
					Computed: true,
					Validators: []validator.String{
						stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("user_id"), path.MatchRelative().AtParent().AtName("group_id")),
						stringvalidator.OneOf(validValues...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of access level.",
					Computed:            true,
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `group_id`.",
					Optional:            true,
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `user_id`.",
					Optional:            true,
				},
			},
		},
	}
}

func migrateAllowedToBlock(previousData []*gitlabBranchProtectionAllowedToObjectModel) []*gitlabBranchProtectionAllowedToPushObjectModel {
	newData := make([]*gitlabBranchProtectionAllowedToPushObjectModel, len(previousData))
	for i, v := range previousData {
		newData[i] = &gitlabBranchProtectionAllowedToPushObjectModel{
			AccessLevel:            v.AccessLevel,
			AccessLevelDescription: v.AccessLevelDescription,
			UserId:                 v.UserId,
			GroupId:                v.GroupId,
		}
	}

	return newData
}

//////////////////////////////////////////////////////////////////
// 				resource schema version v1			         	//
//////////////////////////////////////////////////////////////////

// gitlabBranchProtectionResourceModelV1 describes the resource data model in version v1
type gitlabBranchProtectionResourceModelV1 struct {
	Id                        types.String                                      `tfsdk:"id"`
	BranchProtectionId        types.Int64                                       `tfsdk:"branch_protection_id"`
	Project                   types.String                                      `tfsdk:"project"`
	Branch                    types.String                                      `tfsdk:"branch"`
	MergeAccessLevel          types.String                                      `tfsdk:"merge_access_level"`
	PushAccessLevel           types.String                                      `tfsdk:"push_access_level"`
	UnprotectAccessLevel      types.String                                      `tfsdk:"unprotect_access_level"`
	AllowForcePush            types.Bool                                        `tfsdk:"allow_force_push"`
	CodeOwnerApprovalRequired types.Bool                                        `tfsdk:"code_owner_approval_required"`
	AllowedToPush             []*gitlabBranchProtectionAllowedToPushObjectModel `tfsdk:"allowed_to_push"`
	AllowedToMerge            []*gitlabBranchProtectionAllowedToObjectModel     `tfsdk:"allowed_to_merge"`
	AllowedToUnprotect        []*gitlabBranchProtectionAllowedToObjectModel     `tfsdk:"allowed_to_unprotect"`
}

func (d *gitlabBranchProtectionResource) getV1Schema() schema.Schema {
	return schema.Schema{
		Version: 1,
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_branch_protection`" + ` resource manages the lifecycle of a protected branch of a repository.

~> **Branch Protection Behavior for the default branch**
   Depending on the GitLab instance, group or project setting the default branch of a project is created automatically by GitLab behind the scenes.
   Due to [some](https://gitlab.com/gitlab-org/terraform-provider-gitlab/issues/792) [limitations](https://discuss.hashicorp.com/t/ignore-the-order-of-a-complex-typed-list/42242) in the Terraform Provider SDK and the GitLab API,
   when creating a new project and trying to manage the branch protection setting for its default branch the ` + "`gitlab_branch_protection`" + ` resource will
   automatically take ownership of the default branch without an explicit import by unprotecting and properly protecting it again.
   Having multiple ` + "`gitlab_branch_protection`" + ` resources for the same project and default branch will result in them overriding each other - make sure to only have a single one.
   This behavior might change in the future.

~> The ` + "`allowed_to_push`" + `, ` + "`allowed_to_merge`" + `, ` + "`allowed_to_unprotect`" + `, ` + "`unprotect_access_level`" + ` and ` + "`code_owner_approval_required`" + ` attributes require a GitLab Enterprise instance.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/protected_branches/)`),

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id:branch>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"branch_protection_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the branch protection (not the branch name).",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The id of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "Name of the branch.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"merge_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to merge. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...)},
			},
			"push_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to push. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...)},
			},
			"unprotect_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to unprotect. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidProtectedBranchUnprotectAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchUnprotectAccessLevelNames...)},
			},
			"allow_force_push": schema.BoolAttribute{
				MarkdownDescription: "Can be set to true to allow users with push access to force push.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"code_owner_approval_required": schema.BoolAttribute{
				MarkdownDescription: "Can be set to true to require code owner approval before merging. Only available for Premium and Ultimate instances.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"allowed_to_push":      schemaAllowedToPushBlock(api.ValidProtectedBranchTagAccessLevelNames),
			"allowed_to_unprotect": schemaAllowedToBlock("unprotect push", api.ValidProtectedBranchUnprotectAccessLevelNames),
			"allowed_to_merge":     schemaAllowedToBlock("merge", api.ValidProtectedBranchTagAccessLevelNames),
		},
	}
}

func schemaAllowedToBlock(action string, validValues []string) schema.Block {
	return schema.SetNestedBlock{
		MarkdownDescription: fmt.Sprintf("Array of access levels and user(s)/group(s) allowed to %s to protected branch.", action),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"access_level": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Access levels allowed to %s to protected branch. Valid values are: %s.", action,
						utils.RenderValueListForDocs(validValues)),
					Computed: true,
					Validators: []validator.String{
						stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("user_id"), path.MatchRelative().AtParent().AtName("group_id")),
						stringvalidator.OneOf(validValues...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of access level.",
					Computed:            true,
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `group_id`.",
					Optional:            true,
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `user_id`.",
					Optional:            true,
				},
			},
		},
	}
}

func schemaAllowedToPushBlock(validValues []string) schema.Block {
	return schema.SetNestedBlock{
		MarkdownDescription: "Array of access levels and user(s)/group(s) allowed to push to protected branch.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"access_level": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Access levels allowed to push to protected branch. Valid values are: %s.",
						utils.RenderValueListForDocs(validValues)),
					Computed: true,
					Validators: []validator.String{
						stringvalidator.ExactlyOneOf(
							path.MatchRelative().AtParent().AtName("user_id"),
							path.MatchRelative().AtParent().AtName("group_id"),
							path.MatchRelative().AtParent().AtName("deploy_key_id"),
						),
						stringvalidator.OneOf(validValues...),
					},
				},
				"access_level_description": schema.StringAttribute{
					MarkdownDescription: "Readable description of access level.",
					Computed:            true,
				},
				"user_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `deploy_key_id` and `group_id`.",
					Optional:            true,
				},
				"group_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `deploy_key_id` and `user_id`.",
					Optional:            true,
				},
				"deploy_key_id": schema.Int64Attribute{
					MarkdownDescription: "The ID of a GitLab deploy key allowed to perform the relevant action. Mutually exclusive with `group_id` and `user_id`. This field is read-only until Gitlab 17.5.",
					Optional:            true,
				},
			},
		},
	}
}
