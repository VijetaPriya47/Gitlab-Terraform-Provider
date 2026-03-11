package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
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
	_ resource.Resource                = &gitlabTagProtectionResource{}
	_ resource.ResourceWithConfigure   = &gitlabTagProtectionResource{}
	_ resource.ResourceWithImportState = &gitlabTagProtectionResource{}
)

func init() {
	registerResource(NewGitlabTagProtectionResource)
}

// NewGitlabTagProtectionResource is a helper function to simplify the provider implementation.
func NewGitlabTagProtectionResource() resource.Resource {
	return &gitlabTagProtectionResource{}
}

func (r *gitlabTagProtectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_protection"
}

// gitlabTagProtectionResource defines the resource implementation
type gitlabTagProtectionResource struct {
	client *gitlab.Client
}

// gitlabTagProtectionResourceModel describes the resource data model.
type gitlabTagProtectionResourceModel struct {
	Id                types.String                               `tfsdk:"id"`
	Project           types.String                               `tfsdk:"project"`
	Tag               types.String                               `tfsdk:"tag"`
	CreateAccessLevel types.String                               `tfsdk:"create_access_level"`
	AllowedToCreate   []*gitlabTagProtectionAllowedToObjectModel `tfsdk:"allowed_to_create"`
}

// gitlabTagProtectionAllowedToObjectModel describes the allowed to block data model.
type gitlabTagProtectionAllowedToObjectModel struct {
	AccessLevel            types.String `tfsdk:"access_level"`
	AccessLevelDescription types.String `tfsdk:"access_level_description"`
	UserId                 types.Int64  `tfsdk:"user_id"`
	GroupId                types.Int64  `tfsdk:"group_id"`
	DeployKeyId            types.Int64  `tfsdk:"deploy_key_id"`
}

func (r *gitlabTagProtectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_tag_protection`" + ` resource manages the lifecycle of a tag protection.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/protected_tags/)`),

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project-id:tag>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The id of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"tag": schema.StringAttribute{
				MarkdownDescription: "Name of the tag or wildcard.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"create_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Access levels allowed to create. Default value of `maintainer`. The default value is always sent if not provided in the configuration. Valid values are: %s.", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(api.AccessLevelValueToName[gitlab.MaintainerPermissions]),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...)},
			},
		},
		Blocks: map[string]schema.Block{
			"allowed_to_create": schema.SetNestedBlock{
				MarkdownDescription: "Array of access levels/user(s)/group(s) allowed to create protected tags.",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"access_level": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("Access levels allowed to create protected tags. Valid values are: %s.",
								utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
							Computed:      true,
							Optional:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
							Validators: []validator.String{
								stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("user_id"), path.MatchRelative().AtParent().AtName("group_id"), path.MatchRelative().AtParent().AtName("deploy_key_id")),
								stringvalidator.OneOf(api.ValidProtectedBranchTagAccessLevelNames...),
							},
						},
						"access_level_description": schema.StringAttribute{
							MarkdownDescription: "Readable description of access level.",
							Computed:            true,
						},
						"user_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of a GitLab user allowed to perform the relevant action. Mutually exclusive with `deploy_key_id` and `group_id`.",
							Optional:            true,
							PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
						},
						"group_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of a GitLab group allowed to perform the relevant action. Mutually exclusive with `deploy_key_id` and `user_id`.",
							Optional:            true,
							PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
						},
						"deploy_key_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of a GitLab deploy key allowed to perform the relevant action. Mutually exclusive with `group_id` and `user_id`.",
							Optional:            true,
							PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabTagProtectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Create creates a new upstream resources and adds it into the Terraform state.
func (r *gitlabTagProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabTagProtectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// local copies of plan arguments
	projectID := data.Project.ValueString()

	createAccessLevel := api.AccessLevelNameToValue[data.CreateAccessLevel.ValueString()]

	allowedToCreate := []*gitlab.TagsPermissionOptions{}
	// detect entities to be created
	for _, plannedAllowedTo := range data.AllowedToCreate {
		allowedToTagsPermissionOptionData := &gitlab.TagsPermissionOptions{}

		if !plannedAllowedTo.AccessLevel.IsNull() && plannedAllowedTo.AccessLevel.ValueString() != "" {
			allowedToTagsPermissionOptionData.AccessLevel = gitlab.Ptr(api.AccessLevelNameToValue[plannedAllowedTo.AccessLevel.ValueString()])
		}
		if !plannedAllowedTo.UserId.IsNull() && plannedAllowedTo.UserId.ValueInt64() != 0 {
			allowedToTagsPermissionOptionData.UserID = gitlab.Ptr(plannedAllowedTo.UserId.ValueInt64())
		}
		if !plannedAllowedTo.GroupId.IsNull() && plannedAllowedTo.GroupId.ValueInt64() != 0 {
			allowedToTagsPermissionOptionData.GroupID = gitlab.Ptr(plannedAllowedTo.GroupId.ValueInt64())
		}
		if !plannedAllowedTo.DeployKeyId.IsNull() && plannedAllowedTo.DeployKeyId.ValueInt64() != 0 {
			allowedToTagsPermissionOptionData.DeployKeyID = gitlab.Ptr(plannedAllowedTo.DeployKeyId.ValueInt64())
		}
		allowedToCreate = append(allowedToCreate, allowedToTagsPermissionOptionData)
	}

	// configure GitLab protected tag creation API call
	options := gitlab.ProtectRepositoryTagsOptions{
		Name:              gitlab.Ptr(data.Tag.ValueString()),
		CreateAccessLevel: &createAccessLevel,
		AllowedToCreate:   &allowedToCreate,
	}

	// call Gitlab protected repository tag creation API
	protectedTag, _, err := r.client.ProtectedTags.ProtectRepositoryTags(projectID, &options, gitlab.WithContext(ctx))
	if err != nil {
		// Remove existing tag protection
		_, err = r.client.ProtectedTags.UnprotectRepositoryTags(projectID, data.Tag.ValueString(), gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to unprotect repository tag: %s", err.Error()))
			return
		}
		// Reprotect tag with updated values
		protectedTag, _, err = r.client.ProtectedTags.ProtectRepositoryTags(projectID, &options, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to protect repository tag: %s", err.Error()))
			return
		}
	}

	// If 'allowed_to_create' has been set but didn't come back, it means it's not supported under this license
	// Since the user provided 'create_access_level' gets bundled into the 'create_access_levels' list with the other 'allowed_to_create'
	// entries, we need to subtract 1 (the 'create_access_level') from the returned access levels in order to determine
	// if the user is trying to pass additional 'allowed_to_create' values that aren't allowed in CE
	if len(allowedToCreate) > (len(protectedTag.CreateAccessLevels) - 1) {
		resp.Diagnostics.AddError("GitLab API error occurred", "feature unavailable: `allowed_to_create`, Premium or Ultimate license required. Warning: Changes have partially been persisted!")
		return
	}

	// Create resource ID and persist in state model
	data.Id = types.StringValue(utils.BuildTwoPartID(&projectID, &protectedTag.Name))

	// persist API response in state model
	r.protectedTagToStateModel(projectID, protectedTag, data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// Log the creation of the resource
	tflog.Debug(ctx, "Created a protected tag", map[string]any{
		"project_id": data.Project.ValueString(), "tag": data.Tag.ValueString(),
	})
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabTagProtectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabTagProtectionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	projectID, tag, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project-id>:<tag>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	// call protected repository tag, read Gitlab API
	protectedTag, _, err := r.client.ProtectedTags.GetProtectedTag(projectID, tag, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "protected tag does not exist, removing from state", map[string]any{
				"project_id": projectID, "tag": tag,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read protected tag details: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.protectedTagToStateModel(projectID, protectedTag, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource in-place.
func (r *gitlabTagProtectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error, report upstream", "Somehow the resource was requested to perform an in-place upgrade which is not possible.")
}

// Delete removes the resource.
func (r *gitlabTagProtectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabTagProtectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// call Gitlab unprotect repository tag API
	_, err := r.client.ProtectedTags.UnprotectRepositoryTags(data.Project.ValueString(), data.Tag.ValueString(), gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete gitlab tag protection", err.Error())
		return
	}

	tflog.Debug(ctx, "Delete gitlab protected tag", map[string]any{
		"project_id": data.Project.ValueString(), "tag": data.Tag.ValueString(),
	})
}

// ImportState imports the resource into the Terraform state.
func (r *gitlabTagProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// When accessing protectedTag.*AccessLevels argument by gitlab API 'ProtectedTags.GetProtectedTag' list contains
// elements with AccessLevel field defined and elements connected with user_id&group_id which does not have it. Then it can
// result in setting AccessLevel for null value which will result in error. To prohibit such cases only elements with
// AccessLevel field set are returned
func firstValidAccessLevelForTag(descriptions []*gitlab.TagAccessDescription) (*gitlab.TagAccessDescription, error) {
	for _, description := range descriptions {
		if description.UserID != 0 || description.GroupID != 0 {
			continue
		}
		return description, nil
	}

	return nil, fmt.Errorf("no valid access level found")
}

func populateTagAllowedToObjectList(access_levels []*gitlab.TagAccessDescription) []*gitlabTagProtectionAllowedToObjectModel {
	allowedTosData := make([]*gitlabTagProtectionAllowedToObjectModel, len(access_levels))
	for i, v := range access_levels {
		allowedToData := gitlabTagProtectionAllowedToObjectModel{
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

func (r *gitlabTagProtectionResource) protectedTagToStateModel(projectID string, protectedTag *gitlab.ProtectedTag, data *gitlabTagProtectionResourceModel) {
	data.Project = types.StringValue(projectID)
	data.Tag = types.StringValue(protectedTag.Name)

	// the 'create_access_level' value seems to always be the first entry in the list of 'create_access_levels' on the response
	if createAccessLevel, err := firstValidAccessLevelForTag(protectedTag.CreateAccessLevels); err == nil {
		data.CreateAccessLevel = types.StringValue(api.AccessLevelValueToName[createAccessLevel.AccessLevel])
		// remove the corresponding create access level so it's not added to 'allowed to create' in the state
		protectedTag.CreateAccessLevels = slices.DeleteFunc(protectedTag.CreateAccessLevels, func(desc *gitlab.TagAccessDescription) bool {
			return desc == createAccessLevel
		})

		protectedTag.CreateAccessLevels = slices.Clip(protectedTag.CreateAccessLevels)
	}

	if len(protectedTag.CreateAccessLevels) == 0 {
		data.AllowedToCreate = nil
	} else {
		data.AllowedToCreate = populateTagAllowedToObjectList(protectedTag.CreateAccessLevels)
	}
}
