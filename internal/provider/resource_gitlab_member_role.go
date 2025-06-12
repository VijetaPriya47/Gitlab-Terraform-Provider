package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &gitlabMemberRoleResource{}
	_ resource.ResourceWithConfigure   = &gitlabMemberRoleResource{}
	_ resource.ResourceWithImportState = &gitlabMemberRoleResource{}
	_ resource.ResourceWithModifyPlan  = &gitlabMemberRoleResource{}
)

func init() {
	registerResource(NewGitLabMemberRoleResource)
}

func NewGitLabMemberRoleResource() resource.Resource {
	return &gitlabMemberRoleResource{}
}

type gitlabMemberRoleResource struct {
	client *gitlab.Client
}

type gitlabMemberRoleResourceModel struct {
	Id                 types.String   `tfsdk:"id"`
	Iid                types.Int64    `tfsdk:"iid"`
	GroupPath          types.String   `tfsdk:"group_path"`
	Name               types.String   `tfsdk:"name"`
	Description        types.String   `tfsdk:"description"`
	EditPath           types.String   `tfsdk:"edit_path"`
	CreatedAt          types.String   `tfsdk:"created_at"`
	BaseAccessLevel    types.String   `tfsdk:"base_access_level"`
	EnabledPermissions []types.String `tfsdk:"enabled_permissions"`
}

// Metadata returns the resource name
func (d *gitlabMemberRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member_role"
}

func (r *gitlabMemberRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// The API requires these to be in all uppercase, which is why they're done this way even though it's inconsistent
	// with other similar access levels in the provider.
	allowedBaseAccessLevels := []string{"DEVELOPER", "GUEST", "MAINTAINER", "MINIMAL_ACCESS", "OWNER", "REPORTER"}

	// similarly, these are also required to all be in uppercase.
	allowedEnabledPermissions := []string{
		"ADMIN_CICD_VARIABLES", "ADMIN_COMPLIANCE_FRAMEWORK", "ADMIN_GROUP_MEMBER",
		"ADMIN_INTEGRATIONS", "ADMIN_MERGE_REQUEST", "ADMIN_PROTECTED_BRANCH", "ADMIN_PUSH_RULES", "ADMIN_RUNNERS", "ADMIN_TERRAFORM_STATE",
		"ADMIN_VULNERABILITY", "ADMIN_WEB_HOOK", "ARCHIVE_PROJECT", "MANAGE_DEPLOY_TOKENS", "MANAGE_GROUP_ACCESS_TOKENS",
		"MANAGE_MERGE_REQUEST_SETTINGS", "MANAGE_PROJECT_ACCESS_TOKENS", "MANAGE_SECURITY_POLICY_LINK", "READ_ADMIN_CICD", "READ_ADMIN_DASHBOARD",
		"READ_CODE", "READ_COMPLIANCE_DASHBOARD", "READ_CRM_CONTACT", "READ_DEPENDENCY", "READ_RUNNERS", "READ_VULNERABILITY", "REMOVE_GROUP",
		"REMOVE_PROJECT",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_member_role`" + ` resource allows to manage the lifecycle of a custom member role.

Custom roles allow an organization to create user roles with the precise privileges and permissions required for that organization’s needs.

-> This resource requires an Ultimate license.

-> Most custom roles are considered billable users that use a seat. [Custom roles billing and seat usage](https://docs.gitlab.com/user/custom_roles/#billing-and-seat-usage)

-> There can be only 10 custom roles on your instance or namespace. See [issue 450929](https://gitlab.com/gitlab-org/gitlab/-/issues/450929) for more details.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#mutationmemberrolecreate)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Globally unique ID of the member role. In the format of `gid://gitlab/MemberRole/1`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"iid": schema.Int64Attribute{
				MarkdownDescription: "The id integer value extracted from the `id` attribute",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the namespace to create the member role in. **Required for SAAS** **Not allowed for self-managed**",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name for the member role.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description for the member role.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"edit_path": schema.StringAttribute{
				MarkdownDescription: "The Web UI path to edit the member role",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the member role was created.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"base_access_level": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The base access level for the custom role. Valid values are: %s", utils.RenderValueListForDocs(allowedBaseAccessLevels)),
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf(allowedBaseAccessLevels...)},
			},
			"enabled_permissions": schema.SetAttribute{
				MarkdownDescription: fmt.Sprintf("All permissions enabled for the custom role. Valid values are: %s", utils.RenderValueListForDocs(allowedEnabledPermissions)),
				Required:            true,
				ElementType:         types.StringType,
				Validators:          []validator.Set{setvalidator.ValueStringsAre(stringvalidator.OneOf(allowedEnabledPermissions...))},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabMemberRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// Use the `ModifyPlan` to determine if Gitlab instance is self-hosted vs SaaS.
// If instance is SaaS, group_path is required. If instance is self-hosted, group_path is not permitted.
func (r *gitlabMemberRoleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Retrieve the plan data to start with
	var planData *gitlabMemberRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if planData == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no check for instance type needed")
		return
	}

	if len(r.client.BaseURL().Host) == 0 || r.client.BaseURL().Host == "gitlab.com" {
		if planData.GroupPath.IsNull() || planData.GroupPath.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(path.Root("group_path"), "Missing Attribute", "`group_path` is required when using GitLab SaaS")
		}
	} else {
		if !planData.GroupPath.IsNull() && planData.GroupPath.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(path.Root("group_path"), "Attribute Not Permitted", "`group_path` is not allowed when using GitLab self-managed")
		}
	}
}

func (r *gitlabMemberRoleResource) memberRoleToStateModel(response *MemberRole, groupPath string, data *gitlabMemberRoleResourceModel) {
	data.Id = types.StringValue(response.ID)
	data.Name = types.StringValue(response.Name)
	data.Description = types.StringValue(response.Description)
	data.EditPath = types.StringValue(response.EditPath)
	data.CreatedAt = types.StringValue(response.CreatedAt)
	data.BaseAccessLevel = types.StringValue(response.BaseAccessLevel.StringValue)
	data.GroupPath = types.StringValue(groupPath)

	enabledPermissions := []basetypes.StringValue{}
	for _, v := range response.EnabledPermissions.Nodes {
		enabledPermissions = append(enabledPermissions, types.StringValue(v.Value))
	}
	data.EnabledPermissions = enabledPermissions

	iid, _ := api.ExtractIIDFromGlobalID(response.ID)
	data.Iid = types.Int64Value(int64(iid))
}

// Read refreshes the Terraform state with the latest data.
func (r *gitlabMemberRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabMemberRoleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// read all information for refresh from resource id
	id := data.Id.ValueString()
	groupPath := data.GroupPath.ValueString()

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			query {
				memberRole(id: "%s") {
					baseAccessLevel {
						stringValue
					},
					createdAt,
					description,
					editPath,
					enabledPermissions {
						nodes {
							value
						}
					},
					id,
					name,
				}
			}`, id),
	}
	tflog.Debug(ctx, "executing GraphQL Query to retrieve current custom member role", map[string]any{
		"query": query.Query,
	})

	var response MemberRoleResponse
	if _, err := r.client.GraphQL.Do(query, &response); err != nil {
		if response.Data.MemberRole.ID == "" {
			tflog.Debug(ctx, "member role does not exist, removing from state", map[string]any{
				"id": id,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read member role details: %s", err.Error()))
		return
	}

	// persist API response in state model
	r.memberRoleToStateModel(&response.Data.MemberRole, groupPath, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create creates a new upstream resource and adds it into the Terraform state.
func (r *gitlabMemberRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabMemberRoleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	description := data.Description.ValueString()
	baseAccessLevel := data.BaseAccessLevel.ValueString()
	groupPath := data.GroupPath.ValueString()

	var permissions []string
	for _, v := range data.EnabledPermissions {
		permissions = append(permissions, v.ValueString())
	}

	// If SaaS instance, include groupPath
	groupPathQuery := ""
	if len(r.client.BaseURL().Host) == 0 || r.client.BaseURL().Host == "gitlab.com" {
		groupPathQuery = fmt.Sprintf(`groupPath: "%s",`, groupPath)
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				memberRoleCreate(
					input: {
						%s
						name: "%s",
						description: "%s",
						baseAccessLevel: %s,
						permissions: %s
					}
				) {
					memberRole {
						baseAccessLevel {
							stringValue
						},
						createdAt,
						description,
						editPath,
						enabledPermissions {
							nodes {
								value
							}
						},
						id,
						name
					}
					errors
				}
			}`, groupPathQuery, name, description, baseAccessLevel, permissions),
	}

	tflog.Debug(ctx, "executing GraphQL Query to create custom member role", map[string]any{
		"query": query.Query,
	})

	var response createMemberRoleResponse
	if _, err := r.client.GraphQL.Do(query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create custom member role: %s", err.Error()))
		return
	}

	// check response for errors
	var allerr string
	if len(response.Errors) > 0 {
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
	}
	if len(response.Data.MemberRoleCreate.Errors) > 0 {
		for i, err := range response.Data.MemberRoleCreate.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
	}
	if len(allerr) > 0 {
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}

	// persist API response in state model
	r.memberRoleToStateModel(&response.Data.MemberRoleCreate.MemberRole, groupPath, data)

	// Log the creation of the resource
	tflog.Debug(ctx, "created a custom member role", map[string]any{
		"id": data.Id.ValueString(), "name": data.Name.ValueString(), "description": data.Description.ValueString(), "group_path": data.GroupPath.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource.
func (r *gitlabMemberRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabMemberRoleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id := data.Id.ValueString()

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				memberRoleDelete(
					input: {
						id: "%s"
					}
				) {
					errors
				}
			}`, id),
	}
	tflog.Debug(ctx, "executing GraphQL Query to delete custom member role", map[string]any{
		"query": query.Query,
	})

	var response deleteMemberRoleResponse
	if _, err := r.client.GraphQL.Do(query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete custom member role: %s", err.Error()))
		return
	}

	// check response for errors
	var allerr string
	if len(response.Errors) > 0 {
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
	}
	if len(response.Data.MemberRoleDelete.Errors) > 0 {
		for i, err := range response.Data.MemberRoleDelete.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
	}
	if len(allerr) > 0 {
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}
}

// Update updates the resource in-place.
func (r *gitlabMemberRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabMemberRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id := data.Id.ValueString()
	groupPath := data.GroupPath.ValueString()
	name := data.Name.ValueString()
	description := data.Description.ValueString()

	var permissions []string
	for _, v := range data.EnabledPermissions {
		permissions = append(permissions, v.ValueString())
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				memberRoleUpdate(
					input: {
						id: "%s",
						name: "%s",
						description: "%s",
						permissions: %v
					}
				) {
					memberRole {
						baseAccessLevel {
							stringValue
						},
						createdAt,
						description,
						editPath,
						enabledPermissions {
							nodes {
								value
							}
						},
						id,
						name
					}
					errors
				}
			}`, id, name, description, permissions),
	}
	tflog.Debug(ctx, "executing GraphQL Query to update custom member role", map[string]any{
		"query": query.Query,
	})

	var response updateMemberRoleResponse
	if _, err := r.client.GraphQL.Do(query, &response); err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to update custom member role: %s", err.Error()))
		return
	}

	// check response for errors
	var allerr string
	if len(response.Errors) > 0 {
		for i, err := range response.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err.Message)
		}
	}
	if len(response.Data.MemberRoleUpdate.Errors) > 0 {
		for i, err := range response.Data.MemberRoleUpdate.Errors {
			allerr += fmt.Sprintf("Error %d Message: %s\n", i, err)
		}
	}
	if len(allerr) > 0 {
		resp.Diagnostics.AddError("GitLab GraphQL error occurred", allerr)
		return
	}

	// persist API response in state model
	r.memberRoleToStateModel(&response.Data.MemberRoleUpdate.MemberRole, groupPath, data)

	// Log the update of the resource
	tflog.Debug(ctx, "updated a custom member role", map[string]any{
		"id": data.Id.ValueString(), "name": data.Name.ValueString(), "description": data.Description.ValueString(), "group_path": data.GroupPath.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabMemberRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type MemberRole struct {
	BaseAccessLevel struct {
		StringValue string `json:"stringValue"`
	} `json:"baseAccessLevel"`
	CreatedAt          string `json:"createdAt"`
	Description        string `json:"description"`
	EditPath           string `json:"editPath"`
	EnabledPermissions struct {
		Nodes []struct {
			Value string `json:"value"`
		} `json:"nodes"`
	} `json:"enabledPermissions"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MemberRoleResponse struct {
	Data struct {
		MemberRole MemberRole `json:"memberRole"`
	} `json:"data"`
}

type createMemberRoleResponse struct {
	Data struct {
		MemberRoleCreate struct {
			MemberRole MemberRole `json:"memberRole"`
			Errors     []string   `json:"errors"`
		} `json:"memberRoleCreate"`
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

type updateMemberRoleResponse struct {
	Data struct {
		MemberRoleUpdate struct {
			MemberRole MemberRole `json:"memberRole"`
			Errors     []string   `json:"errors"`
		} `json:"memberRoleUpdate"`
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

type deleteMemberRoleResponse struct {
	Data struct {
		MemberRoleDelete struct {
			MemberRole MemberRole `json:"memberRole"`
			Errors     []string   `json:"errors"`
		} `json:"memberRoleDelete"`
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
