package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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
	_ resource.Resource                 = &gitlabBranchProtectionResource{}
	_ resource.ResourceWithConfigure    = &gitlabBranchProtectionResource{}
	_ resource.ResourceWithImportState  = &gitlabBranchProtectionResource{}
	_ resource.ResourceWithUpgradeState = &gitlabBranchProtectionResource{}
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
	resp.Schema = r.getV1Schema()
}

func (d *gitlabBranchProtectionResource) getV1Schema() schema.Schema {
	return schema.Schema{
		Version: 1,
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_branch_protection`" + ` resource allows to manage the lifecycle of a protected branch of a repository.

~> **Branch Protection Behavior for the default branch**
   Depending on the GitLab instance, group or project setting the default branch of a project is created automatically by GitLab behind the scenes.
   Due to [some](https://gitlab.com/gitlab-org/terraform-provider-gitlab/issues/792) [limitations](https://discuss.hashicorp.com/t/ignore-the-order-of-a-complex-typed-list/42242) in the Terraform Provider SDK and the GitLab API,
   when creating a new project and trying to manage the branch protection setting for its default branch the ` + "`gitlab_branch_protection`" + ` resource will
   automatically take ownership of the default branch without an explicit import by unprotecting and properly protecting it again.
   Having multiple ` + "`gitlab_branch_protection`" + ` resources for the same project and default branch will result in them overriding each other - make sure to only have a single one.
   This behavior might change in the future.

~> The ` + "`allowed_to_push`" + `, ` + "`allowed_to_merge`" + `, ` + "`allowed_to_unprotect`" + `, ` + "`unprotect_access_level`" + ` and ` + "`code_owner_approval_required`" + ` attributes require a GitLab Enterprise instance.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/protected_branches.html)`),

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
					Description: "Readable description of access level.",
					Computed:    true,
				},
				"user_id": schema.Int64Attribute{
					Description: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `group_id`.",
					Optional:    true,
				},
				"group_id": schema.Int64Attribute{
					Description: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `user_id`.",
					Optional:    true,
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
					Description: "Readable description of access level.",
					Computed:    true,
				},
				"user_id": schema.Int64Attribute{
					Description: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `deploy_key_id` and `group_id`.",
					Optional:    true,
				},
				"group_id": schema.Int64Attribute{
					Description: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `deploy_key_id` and `user_id`.",
					Optional:    true,
				},
				"deploy_key_id": schema.Int64Attribute{
					Description: "The ID of a GitLab deploy key allowed to perform the relevant action. Mutually exclusive with `group_id` and `user_id`. This field is read-only until Gitlab 17.5.",
					Optional:    true,
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabBranchProtectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*gitlab.Client)
}

// Create creates a new upstream resources and adds it into the Terraform state.
func (r *gitlabBranchProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabBranchProtectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
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

	pushAccessLevel := api.AccessLevelNameToValue[data.PushAccessLevel.ValueString()]
	mergeAccessLevel := api.AccessLevelNameToValue[data.MergeAccessLevel.ValueString()]
	unprotectAccessLevel := api.AccessLevelNameToValue[data.UnprotectAccessLevel.ValueString()]

	allowedToPush := generateAllowedToPushStateToAccessLevels([]*gitlab.BranchAccessDescription{}, data.AllowedToPush)
	allowedToMerge := generateAllowedToStateToAccessLevels([]*gitlab.BranchAccessDescription{}, data.AllowedToMerge)
	allowedToUnprotect := generateAllowedToStateToAccessLevels([]*gitlab.BranchAccessDescription{}, data.AllowedToUnprotect)

	// configure GitLab protected branch creation API call
	options := gitlab.ProtectRepositoryBranchesOptions{
		Name:                      gitlab.Ptr(data.Branch.ValueString()),
		PushAccessLevel:           &pushAccessLevel,
		MergeAccessLevel:          &mergeAccessLevel,
		UnprotectAccessLevel:      &unprotectAccessLevel,
		AllowForcePush:            gitlab.Ptr(data.AllowForcePush.ValueBool()),
		CodeOwnerApprovalRequired: data.CodeOwnerApprovalRequired.ValueBoolPointer(),
		AllowedToPush:             &allowedToPush,
		AllowedToMerge:            &allowedToMerge,
		AllowedToUnprotect:        &allowedToUnprotect,
	}

	// validate attribute compliance with license model
	if !areCreateAttributesCompliantWithLicenseModel(r.client, options, resp) {
		return
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
	r.protectedBranchToStateModel(projectID, protectedBranch, data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// Log the creation of the resource
	tflog.Debug(ctx, "Created a protected branch", map[string]interface{}{
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
			tflog.Debug(ctx, "protected branch does not exist, removing from state", map[string]interface{}{
				"project_id": projectID, "branch": branch,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read protected branch details: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.protectedBranchToStateModel(projectID, protectedBranch, data)

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

	// local copies of plan arguments
	projectID := data.Project.ValueString()
	branch := data.Branch.ValueString()

	// call read, protected repository branch API to retrieve current unprotected access levels
	protectedBranch, _, err := r.client.ProtectedBranches.GetProtectedBranch(projectID, branch, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "protected branch does not exist, removing from state", map[string]interface{}{
				"project_id": projectID, "branch": branch,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read protected branch details: %s", err.Error()))
		return
	}

	allowedToPush := generateAllowedToPushStateToAccessLevels(protectedBranch.PushAccessLevels, data.AllowedToPush)
	allowedToMerge := generateAllowedToStateToAccessLevels(protectedBranch.MergeAccessLevels, data.AllowedToMerge)
	allowedToUnprotect := generateAllowedToStateToAccessLevels(protectedBranch.UnprotectAccessLevels, data.AllowedToUnprotect)

	// configure protect repository branch update GitLab API call
	options := gitlab.UpdateProtectedBranchOptions{
		Name:                      gitlab.Ptr(branch),
		AllowForcePush:            gitlab.Ptr(data.AllowForcePush.ValueBool()),
		CodeOwnerApprovalRequired: data.CodeOwnerApprovalRequired.ValueBoolPointer(),
		AllowedToPush:             &allowedToPush,
		AllowedToMerge:            &allowedToMerge,
		AllowedToUnprotect:        &allowedToUnprotect,
	}

	// validate attribute compliance with license model
	if !areUpdateAttributesCompliantWithLicenseModel(r.client, options, resp) {
		return
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
	r.protectedBranchToStateModel(projectID, updatedProtectedBranch, data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// Log the update of the resource
	tflog.Debug(ctx, "Updated a protected branch", map[string]interface{}{
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

	tflog.Debug(ctx, "Delete gitlab protected branch", map[string]interface{}{
		"project_id": data.Project.ValueString(), "branch": data.Branch.ValueString(),
	})
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabBranchProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabBranchProtectionResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	// The v0 schema. Needed so the pointer can be used.
	schema := d.getV0Schema()

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data *gitlabBranchProtectionResourceModelv0

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
					UnprotectAccessLevel:      data.UnprotectAccessLevel,
					AllowForcePush:            data.AllowForcePush,
					AllowedToPush:             migrateAllowedToBlock(data.AllowedToPush),
					AllowedToMerge:            data.AllowedToMerge,
					AllowedToUnprotect:        data.AllowedToUnprotect,
					CodeOwnerApprovalRequired: data.CodeOwnerApprovalRequired,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)

				tflog.Debug(ctx, "Upgraded gitlab_branch_protection resource from v0 to v1 verion", map[string]interface{}{
					"project_id": data.Project.ValueString(), "branch": data.Branch.ValueString(),
				})
			},
		},
	}
}

func areCreateAttributesCompliantWithLicenseModel(client *gitlab.Client, pbOptions gitlab.ProtectRepositoryBranchesOptions, resp *resource.CreateResponse) bool {
	// part of functionalities are available only for enterpise licensing model. To check if determine license model
	ee_context, err := utils.IsRunningInEEContext(client)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get gitlab server metadata details: %s", err.Error()))
		return false
	}

	// if license model is not enterpise make sure that no enterprise features are set
	if !ee_context {
		if len(*pbOptions.AllowedToPush) > 0 {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_push`", "Enterprise license required")
			return false
		}
		if len(*pbOptions.AllowedToMerge) > 0 {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_merge`", "Enterprise license required")
			return false
		}
		if len(*pbOptions.AllowedToUnprotect) > 0 {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_unprotect`", "Enterprise license required")
			return false
		}
		if *pbOptions.CodeOwnerApprovalRequired {
			resp.Diagnostics.AddError("feature unavailable `code_owner_approval_required`", "Enterprise license required")
			return false
		}
	}
	return true
}

func areUpdateAttributesCompliantWithLicenseModel(client *gitlab.Client, pbOptions gitlab.UpdateProtectedBranchOptions, resp *resource.UpdateResponse) bool {
	// part of functionalities are available only for enterpise licensing model. To check if determine license model
	ee_context, err := utils.IsRunningInEEContext(client)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get gitlab server metadata details: %s", err.Error()))
		return false
	}

	// if license model is not enterpise make sure that no enterprise features are set
	if !ee_context {
		if len(*pbOptions.AllowedToPush) > 0 {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_push`", "Enterprise license required")
			return false
		}
		if len(*pbOptions.AllowedToMerge) > 0 {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_merge`", "Enterprise license required")
			return false
		}
		if len(*pbOptions.AllowedToUnprotect) > 0 {
			resp.Diagnostics.AddError("feature unavailable `allowed_to_unprotect`", "Enterprise license required")
			return false
		}
		if *pbOptions.CodeOwnerApprovalRequired {
			resp.Diagnostics.AddError("feature unavailable `code_owner_approval_required`", "Enterprise license required")
			return false
		}
	}
	return true
}

// When accessing protectedBranch.*AccessLevels argument by gitlab API 'ProtectedBranches.GetProtectedBranch' list contains
// elements with AccessLevel field defined and elements connected with user_id&group_id which does not have it. Then it can
// result in setting AccessLevel for null value which will result in error. To prohibite such cases only elements with
// AccessLevel field set are returned
func firstValidAccessLevel(descriptions []*gitlab.BranchAccessDescription) (*gitlab.AccessLevelValue, error) {
	for _, description := range descriptions {
		if description.UserID != 0 || description.GroupID != 0 {
			continue
		}
		return &description.AccessLevel, nil
	}

	return nil, fmt.Errorf("no valid access level found")
}

func generateAllowedToStateToAccessLevels(currentAllowedTos []*gitlab.BranchAccessDescription, plannedAllowedTos []*gitlabBranchProtectionAllowedToObjectModel) []*gitlab.BranchPermissionOptions {
	validCurrentAllowedTos := []*gitlab.BranchAccessDescription{}
	// retrieve only elements assigned to user(s) or group(s)
	for _, currentAllowedTo := range currentAllowedTos {
		if currentAllowedTo.UserID != 0 || currentAllowedTo.GroupID != 0 {
			validCurrentAllowedTos = append(validCurrentAllowedTos, currentAllowedTo)
		}
	}

	finalAllowedTo := []*gitlab.BranchPermissionOptions{}
	//detect entities to be created
	for _, plannedAllowedTo := range plannedAllowedTos {
		var allowedToBranchPermissionOptionData *gitlab.BranchPermissionOptions = populateBranchPermissionOptionsData(validCurrentAllowedTos, plannedAllowedTo)
		if allowedToBranchPermissionOptionData != nil {
			finalAllowedTo = append(finalAllowedTo, allowedToBranchPermissionOptionData)
		}
	}

	//detect entities to be removed
	for _, validCurrentAllowedTo := range validCurrentAllowedTos {
		requireRemoval := true
		for _, plannedAllowedTo := range plannedAllowedTos {
			// if element exists in planned values skip removal
			if !plannedAllowedTo.UserId.IsNull() && plannedAllowedTo.UserId.ValueInt64() == int64(validCurrentAllowedTo.UserID) ||
				!plannedAllowedTo.GroupId.IsNull() && plannedAllowedTo.GroupId.ValueInt64() == int64(validCurrentAllowedTo.GroupID) {
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

func generateAllowedToPushStateToAccessLevels(currentAllowedTos []*gitlab.BranchAccessDescription, plannedAllowedTos []*gitlabBranchProtectionAllowedToPushObjectModel) []*gitlab.BranchPermissionOptions {
	validCurrentAllowedTos := []*gitlab.BranchAccessDescription{}
	// retrieve only elements assigned to user(s) or group(s)
	for _, currentAllowedTo := range currentAllowedTos {
		if currentAllowedTo.UserID != 0 || currentAllowedTo.GroupID != 0 || currentAllowedTo.DeployKeyID != 0 {
			validCurrentAllowedTos = append(validCurrentAllowedTos, currentAllowedTo)
		}
	}

	finalAllowedTo := []*gitlab.BranchPermissionOptions{}
	//detect entities to be created
	for _, plannedAllowedTo := range plannedAllowedTos {
		var allowedToBranchPermissionOptionData *gitlab.BranchPermissionOptions = populateBranchPermissionOptionsDataForPush(validCurrentAllowedTos, plannedAllowedTo)
		if allowedToBranchPermissionOptionData != nil {
			finalAllowedTo = append(finalAllowedTo, allowedToBranchPermissionOptionData)
		}
	}

	//detect entities to be removed
	for _, validCurrentAllowedTo := range validCurrentAllowedTos {
		requireRemoval := true
		for _, plannedAllowedTo := range plannedAllowedTos {
			// if element exists in planned values skip removal
			if !plannedAllowedTo.UserId.IsNull() && plannedAllowedTo.UserId.ValueInt64() == int64(validCurrentAllowedTo.UserID) ||
				!plannedAllowedTo.GroupId.IsNull() && plannedAllowedTo.GroupId.ValueInt64() == int64(validCurrentAllowedTo.GroupID) ||
				!plannedAllowedTo.DeployKeyId.IsNull() && plannedAllowedTo.DeployKeyId.ValueInt64() == int64(validCurrentAllowedTo.DeployKeyID) {
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
	//detect if element already exists
	for _, currentAllowedTo := range currentAllowedTos {
		//if element already exists skip creation
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
			allowedToBranchPermissionOptionData.UserID = gitlab.Ptr(int(allowedTo.UserId.ValueInt64()))
		}
		if !allowedTo.GroupId.IsNull() && allowedTo.GroupId.ValueInt64() != 0 {
			allowedToBranchPermissionOptionData.GroupID = gitlab.Ptr(int(allowedTo.GroupId.ValueInt64()))
		}
	}

	return allowedToBranchPermissionOptionData
}

func populateBranchPermissionOptionsDataForPush(currentAllowedTos []*gitlab.BranchAccessDescription, allowedTo *gitlabBranchProtectionAllowedToPushObjectModel) *gitlab.BranchPermissionOptions {
	var allowedToBranchPermissionOptionData *gitlab.BranchPermissionOptions
	requireCreation := true
	//detect if element already exists
	for _, currentAllowedTo := range currentAllowedTos {
		//if element already exists skip creation
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
			allowedToBranchPermissionOptionData.UserID = gitlab.Ptr(int(allowedTo.UserId.ValueInt64()))
		}
		if !allowedTo.GroupId.IsNull() && allowedTo.GroupId.ValueInt64() != 0 {
			allowedToBranchPermissionOptionData.GroupID = gitlab.Ptr(int(allowedTo.GroupId.ValueInt64()))
		}
		if !allowedTo.DeployKeyId.IsNull() && allowedTo.DeployKeyId.ValueInt64() != 0 {
			allowedToBranchPermissionOptionData.DeployKeyID = gitlab.Ptr(int(allowedTo.DeployKeyId.ValueInt64()))
		}
	}

	return allowedToBranchPermissionOptionData
}

func populateAllowedToToStateModel(accessLevels []*gitlab.BranchAccessDescription) []*gitlabBranchProtectionAllowedToObjectModel {
	valid_access_levels := []*gitlab.BranchAccessDescription{}
	for i := range accessLevels {
		if accessLevels[i].UserID != 0 || accessLevels[i].GroupID != 0 {
			valid_access_levels = append(valid_access_levels, accessLevels[i])
		}
	}

	if len(valid_access_levels) == 0 {
		return nil
	}
	return populateAllowedToObjectList(valid_access_levels)
}

func populateAllowedToPushToStateModel(accessLevels []*gitlab.BranchAccessDescription) []*gitlabBranchProtectionAllowedToPushObjectModel {
	valid_access_levels := []*gitlab.BranchAccessDescription{}
	for i := range accessLevels {
		if accessLevels[i].UserID != 0 || accessLevels[i].GroupID != 0 || accessLevels[i].DeployKeyID != 0 {
			valid_access_levels = append(valid_access_levels, accessLevels[i])
		}
	}

	if len(valid_access_levels) == 0 {
		return nil
	}
	return populateAllowedToPushObjectList(valid_access_levels)
}

func populateAllowedToObjectList(access_levels []*gitlab.BranchAccessDescription) []*gitlabBranchProtectionAllowedToObjectModel {
	allowedTosData := make([]*gitlabBranchProtectionAllowedToObjectModel, len(access_levels))
	for i, v := range access_levels {
		allowedToData := gitlabBranchProtectionAllowedToObjectModel{
			AccessLevelDescription: types.StringValue(v.AccessLevelDescription),
		}
		allowedToData.AccessLevel = types.StringValue(api.AccessLevelValueToName[v.AccessLevel])
		if v.UserID != 0 {
			allowedToData.UserId = types.Int64Value(int64(v.UserID))
		}
		if v.GroupID != 0 {
			allowedToData.GroupId = types.Int64Value(int64(v.GroupID))
		}
		allowedTosData[i] = &allowedToData
	}

	return allowedTosData
}

func populateAllowedToPushObjectList(access_levels []*gitlab.BranchAccessDescription) []*gitlabBranchProtectionAllowedToPushObjectModel {
	allowedTosData := make([]*gitlabBranchProtectionAllowedToPushObjectModel, len(access_levels))
	for i, v := range access_levels {
		allowedToData := gitlabBranchProtectionAllowedToPushObjectModel{
			AccessLevelDescription: types.StringValue(v.AccessLevelDescription),
		}
		allowedToData.AccessLevel = types.StringValue(api.AccessLevelValueToName[v.AccessLevel])
		if v.UserID != 0 {
			allowedToData.UserId = types.Int64Value(int64(v.UserID))
		}
		if v.GroupID != 0 {
			allowedToData.GroupId = types.Int64Value(int64(v.GroupID))
		}
		if v.DeployKeyID != 0 {
			allowedToData.DeployKeyId = types.Int64Value(int64(v.DeployKeyID))
		}
		allowedTosData[i] = &allowedToData
	}

	return allowedTosData

}

func (r *gitlabBranchProtectionResource) protectedBranchToStateModel(projectID string, protectedBranch *gitlab.ProtectedBranch, data *gitlabBranchProtectionResourceModel) {

	data.Project = types.StringValue(projectID)
	data.Branch = types.StringValue(protectedBranch.Name)

	if mergeAccessLevel, err := firstValidAccessLevel(protectedBranch.MergeAccessLevels); err == nil {
		data.MergeAccessLevel = types.StringValue(api.AccessLevelValueToName[*mergeAccessLevel])
	}
	if pushAccessLevel, err := firstValidAccessLevel(protectedBranch.PushAccessLevels); err == nil {
		data.PushAccessLevel = types.StringValue(api.AccessLevelValueToName[*pushAccessLevel])
	}
	if unprotectAccessLevel, err := firstValidAccessLevel(protectedBranch.UnprotectAccessLevels); err == nil {
		data.UnprotectAccessLevel = types.StringValue(api.AccessLevelValueToName[*unprotectAccessLevel])
	}

	data.AllowForcePush = types.BoolValue(protectedBranch.AllowForcePush)

	data.AllowedToPush = populateAllowedToPushToStateModel(protectedBranch.PushAccessLevels)
	data.AllowedToMerge = populateAllowedToToStateModel(protectedBranch.MergeAccessLevels)
	data.AllowedToUnprotect = populateAllowedToToStateModel(protectedBranch.UnprotectAccessLevels)

	data.CodeOwnerApprovalRequired = types.BoolValue(protectedBranch.CodeOwnerApprovalRequired)

	data.BranchProtectionId = types.Int64Value(int64(protectedBranch.ID))
}

//////////////////////////////////////////////////////////////////
// 				resource schema verion v0			         	//
//////////////////////////////////////////////////////////////////

// gitlabBranchProtectionResourceModelv0 describes the resource data model in verison v0.
type gitlabBranchProtectionResourceModelv0 struct {
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
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_branch_protection`" + ` resource allows to manage the lifecycle of a protected branch of a repository.

~> **Branch Protection Behavior for the default branch**
   Depending on the GitLab instance, group or project setting the default branch of a project is created automatically by GitLab behind the scenes.
   Due to [some](https://gitlab.com/gitlab-org/terraform-provider-gitlab/issues/792) [limitations](https://discuss.hashicorp.com/t/ignore-the-order-of-a-complex-typed-list/42242) in the Terraform Provider SDK and the GitLab API,
   when creating a new project and trying to manage the branch protection setting for its default branch the ` + "`gitlab_branch_protection`" + ` resource will
   automatically take ownership of the default branch without an explicit import by unprotecting and properly protecting it again.
   Having multiple ` + "`gitlab_branch_protection`" + ` resources for the same project and default branch will result in them overriding each other - make sure to only have a single one.
   This behavior might change in the future.

~> The ` + "`allowed_to_push`" + `, ` + "`allowed_to_merge`" + `, ` + "`allowed_to_unprotect`" + `, ` + "`unprotect_access_level`" + ` and ` + "`code_owner_approval_required`" + ` attributes require a GitLab Enterprise instance.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/protected_branches.html)`),

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
					Description: "Readable description of access level.",
					Computed:    true,
				},
				"user_id": schema.Int64Attribute{
					Description: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `group_id`.",
					Optional:    true,
				},
				"group_id": schema.Int64Attribute{
					Description: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `user_id`.",
					Optional:    true,
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
