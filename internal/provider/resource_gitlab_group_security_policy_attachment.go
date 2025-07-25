package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &gitlabGroupSecurityPolicyAttachmentResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupSecurityPolicyAttachmentResource{}
	_ resource.ResourceWithImportState = &gitlabGroupSecurityPolicyAttachmentResource{}
	_ resource.ResourceWithModifyPlan  = &gitlabGroupSecurityPolicyAttachmentResource{}
)

func init() {
	registerResource(NewGitlabGroupSecurityPolicyAttachmentResource)
}

func NewGitlabGroupSecurityPolicyAttachmentResource() resource.Resource {
	return &gitlabGroupSecurityPolicyAttachmentResource{}
}

type gitlabGroupSecurityPolicyAttachmentResource struct {
	client *gitlab.Client
}

type gitlabGroupSecurityPolicyAttachmentResourceModel struct {
	Id                     types.String `tfsdk:"id"`
	Group                  types.String `tfsdk:"group"`
	GroupGraphQLId         types.String `tfsdk:"group_graphql_id"`
	PolicyProject          types.String `tfsdk:"policy_project"`
	PolicyProjectGraphQLId types.String `tfsdk:"policy_project_graphql_id"`
}

func (r *gitlabGroupSecurityPolicyAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_security_policy_attachment"
}

func (r *gitlabGroupSecurityPolicyAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_group_security_policy_attachment`" + ` resource allows to attach a security policy project to a group.
This resource requires being an owner on the group that is having the security policy applied.

~> [Policies](https://docs.gitlab.com/user/application_security/policies/) are files stored in a policy project as raw YAML, to allow maximum flexibility with support of all kind of policy and all their options. See the examples for how to create a policy project, add a policy, and link it. Use the ` + "`gitlab_repository_file`" + ` resource to create policies instead of a specific policy resource. This ensures all policy options are immediately via Terraform once released.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/index/#mutationsecuritypolicyprojectassign)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<group>:<policy_project>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "The ID or Full Path of the group which will have the security policy project assigned to it.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"group_graphql_id": schema.StringAttribute{
				MarkdownDescription: "The GraphQL ID of the group to which the security policty project will be attached.",
				Computed:            true,
			},
			"policy_project": schema.StringAttribute{
				MarkdownDescription: "The ID or Full Path of the security policy project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"policy_project_graphql_id": schema.StringAttribute{
				MarkdownDescription: "The GraphQL ID of the security policy project.",
				Computed:            true,
			},
		},
	}
}

func (r *gitlabGroupSecurityPolicyAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (d *gitlabGroupSecurityPolicyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabGroupSecurityPolicyAttachmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var data *gitlabGroupSecurityPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data == nil {
		// Log a note that there is no plan data, usually because we're importing.
		tflog.Debug(ctx, "Plan data is nil, no check for token permissions is needed")
		return
	}

	// Check if the current user has Owner permissions on the group
	user, _, err := d.client.Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to check current user permissions: %s", err.Error()))
		return
	}

	// Admin users can always apply security policies
	if user.IsAdmin {
		return
	}

	// Check group membership for Owner permissions
	membership, _, err := d.client.GroupMembers.GetInheritedGroupMember(data.Group.ValueString(), user.ID, gitlab.WithContext(ctx))
	if err != nil && !api.Is404(err) {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to check group membership: %s", err.Error()))
		return
	}

	// Handle the scenario where the user is not a member at all
	if err != nil {
		// User is not a member of the group at all (404 error)
		if api.Is404(err) {
			resp.Diagnostics.AddError("Access Denied", fmt.Sprintf("Current user is not a member of the group '%s'. To apply a security policy, the token must be added as an Owner to the group.", data.Group.ValueString()))
			return
		}

		resp.Diagnostics.AddError("GitLab API error occurred when attempting to read group membership", err.Error())
		return
	}

	if membership.AccessLevel != gitlab.OwnerPermissions {
		// User is a member but doesn't have Owner permissions
		resp.Diagnostics.AddError("Insufficient Permissions", fmt.Sprintf("Current user has %s access to group '%s', but Owner permissions are required to apply security policies.", api.AccessLevelValueToName[membership.AccessLevel], data.Group.ValueString()))
		return
	}
}

func (d *gitlabGroupSecurityPolicyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupSecurityPolicyAttachmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupIds, err := d.parseGraphQLIds(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse GraphQL IDs", err.Error())
		return
	}

	err = d.updatePolicy(data, groupIds)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update GraphQL ID", err.Error())
		return
	}

	// Verify the association applied properly with a ticker
	err = d.verifyPolicyAssociation(ctx, data, groupIds)
	if err != nil {
		resp.Diagnostics.AddError("Failed to verify policy association", err.Error())
		return
	}

	// Set the ID
	data.Id = types.StringValue(utils.BuildTwoPartID(data.Group.ValueStringPointer(), data.PolicyProject.ValueStringPointer()))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGroupSecurityPolicyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupSecurityPolicyAttachmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, policyProject, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse IDs", err.Error())
		return
	}
	data.Group = types.StringValue(group)
	data.PolicyProject = types.StringValue(policyProject)

	// Get the GraphQL IDs of the project and group
	groupIds, err := d.parseGraphQLIds(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse GraphQL IDs", err.Error())
		return
	}

	// Read the policy project
	tflog.Info(ctx, "Reading the security policy project for the group.", map[string]any{
		"group":          group,
		"policy_project": policyProject,
	})

	response, err := d.readPolicy(groupIds)
	if err != nil {
		tflog.Error(ctx, "Received an error when reading the policy. Exiting", map[string]any{
			"grooup":         group,
			"policy_project": policyProject,
		})
		resp.Diagnostics.AddError("Failed to read policy", "Could not read policy: "+err.Error())
		return
	}

	if response.Data.Group == nil {
		tflog.Warn(ctx, "Group for the gitlab_group_security_policy_attachment returned nil from the GraphQL call, which usually means the group doesn't exist anymore.", map[string]any{
			"grooup":         group,
			"policy_project": policyProject,
		})
		resp.State.RemoveResource(ctx)
		return
	}
	// Get the policy project ID, which is the final digit in the GraphQL ID of the response
	if response.Data.Group.SecurityPolicyProject != nil && response.Data.Group.SecurityPolicyProject.ID != "" {
		parts := strings.Split(response.Data.Group.SecurityPolicyProject.ID, "/")
		parsedPolicyId := parts[len(parts)-1]

		data.PolicyProject = types.StringValue(parsedPolicyId)

		tflog.Debug(ctx, "Parsed a valid security policy project. Adding to state", map[string]any{
			"group":          group,
			"policy_project": parsedPolicyId,
		})

		// Parse GraphQL IDs again to validate the policy project GID
		_, err := d.parseGraphQLIds(ctx, data)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse GraphQL ID of the policy project", err.Error())
			return
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGroupSecurityPolicyAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *gitlabGroupSecurityPolicyAttachmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupIds, err := d.parseGraphQLIds(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse GraphQL IDs", err.Error())
		return
	}

	// Sometimes when we update the policy, if the GitLab instance is under heavy load, the
	// "removal" of the previous policy happens after the update, and no policy is left behind,
	// causing a situation where the `apply` is successful, then an immediate `plan` is generated.
	// The retry will read after update until we get the policy project we expect.
	err = retry.RetryContext(ctx, 1*time.Minute, func() *retry.RetryError {
		err = d.updatePolicy(data, groupIds)
		if err != nil {
			return retry.NonRetryableError(err)
		}

		response, err := d.readPolicy(groupIds)
		if err != nil {
			tflog.Error(ctx, "Received an error when reading the policy. Exiting", map[string]any{
				"group":          data.Group.ValueString(),
				"policy_project": data.PolicyProject.ValueString(),
			})
			return retry.NonRetryableError(err)
		}

		// If we read, and our read doesn't match our expected policy project, retry.
		if response.Data.Group.SecurityPolicyProject.ID != data.PolicyProject.ValueString() {
			tflog.Warn(ctx, "Received a mismatched policy post-update, retryin update", map[string]any{
				"group":          data.Group.ValueString(),
				"policy_project": data.PolicyProject.ValueString(),
			})
			return retry.RetryableError(fmt.Errorf("Received a mismatched policy post-update. Expected %s, got %s. Retrying update.", data.PolicyProject.ValueString(), response.Data.Group.SecurityPolicyProject.ID))
		}

		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to update policy", err.Error())
		return
	}

	// Verify the association applied properly with a ticker
	err = d.verifyPolicyAssociation(ctx, data, groupIds)
	if err != nil {
		resp.Diagnostics.AddError("Failed to verify policy association", err.Error())
		return
	}

	tflog.Debug(ctx, "Updated security policy project for group", map[string]any{
		"group":          data.Group.ValueString(),
		"policy_project": data.PolicyProject.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabGroupSecurityPolicyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabGroupSecurityPolicyAttachmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectIds, err := d.parseGraphQLIds(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse projectIds to determine which project to remove", err.Error())
		return
	}

	query := fmt.Sprintf(`
		mutation {
			securityPolicyProjectUnassign( input: {
				fullPath:"%s"
			}) {
				errors
			}
		}
	`, projectIds.GroupFullPath)
	var response SecurityProjectUnassignResponse
	_, err = d.client.GraphQL.Do(gitlab.GraphQLQuery{Query: query}, &response)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete the group security policy attachment - generic GraphQL error", err.Error())
		return
	}

	if len(response.Data.SecurityPolicyProjectUnassign.Errors) > 0 {
		// If the policy project has been deleted (I.e., cleaned up from a test or something else) it's removed
		// automatically and don't need to "delete" here. Otherwise it's a valid error
		if !strings.Contains(response.Data.SecurityPolicyProjectUnassign.Errors[0], "Policy project doesn't exist") {
			resp.Diagnostics.AddError("Failed to delete the group security policy attachment", response.Data.SecurityPolicyProjectUnassign.Errors[0])
			return
		}
		return
	}

	tflog.Debug(ctx, "Successfully deleted security policy project from group", map[string]any{
		"group":          data.Group.ValueString(),
		"policy_project": data.PolicyProject.ValueString(),
	})

	resp.State.RemoveResource(ctx)
}

// Create a function that reads the security policy associated to the group
func (d *gitlabGroupSecurityPolicyAttachmentResource) readPolicy(ids *api.GroupIdentifiers) (*GetGroupSecurityPolicyProjectResponse, error) {
	// Read the policy project
	var response GetGroupSecurityPolicyProjectResponse
	query := fmt.Sprintf(`
	query {
		group(fullPath:"%s") {
			id,
			securityPolicyProject {id}
		}
	}
	`, ids.GroupFullPath)
	_, err := d.client.GraphQL.Do(gitlab.GraphQLQuery{Query: query}, &response)
	if err != nil {
		return nil, fmt.Errorf("generic GraphQL error: %s", err.Error())
	}

	if len(response.Errors) > 0 {
		// Similarly, if we successfully get a response, but it has errors, don't retry
		return nil, fmt.Errorf("graphQL query returned an error: %s", response.Errors[0].Message)
	}

	return &response, nil
}

// Update the security policy associated to the group
func (d *gitlabGroupSecurityPolicyAttachmentResource) updatePolicy(data *gitlabGroupSecurityPolicyAttachmentResourceModel, ids *api.GroupIdentifiers) error {
	// Update the policy project - This uses the same mutation as assigning a project to a project, but passes in the group path instead.
	query := fmt.Sprintf(`
		mutation {
			securityPolicyProjectAssign( input: {
				fullPath:"%s",
				securityPolicyProjectId: "%s"
			}) {
				errors
			}
		}
	`, ids.GroupFullPath, data.PolicyProjectGraphQLId.ValueString())

	var response SecurityProjectAssignResponse
	_, err := d.client.GraphQL.Do(gitlab.GraphQLQuery{Query: query}, &response)
	if err != nil {
		return err
	}

	if len(response.Data.SecurityPolicyProjectAssign.Errors) > 0 {
		return errors.New(response.Data.SecurityPolicyProjectAssign.Errors[0].Message)
	}

	return nil
}

// Get the GraphQL IDs for the project
func (d *gitlabGroupSecurityPolicyAttachmentResource) parseGraphQLIds(ctx context.Context, data *gitlabGroupSecurityPolicyAttachmentResourceModel) (*api.GroupIdentifiers, error) {
	// Get the GraphQL of the project Id
	groupGid, err := api.GetGroupGIDFromID(ctx, d.client, data.Group.ValueString())
	if err != nil {
		return nil, err
	}
	data.GroupGraphQLId = types.StringValue(groupGid.GroupGQLID)

	// Get the GraphQL of the policy project ID
	if !data.PolicyProject.IsUnknown() && !data.PolicyProject.IsNull() {
		policyProjectGid, err := api.GetProjectGIDFromID(ctx, d.client, data.PolicyProject.ValueString())
		if err != nil {
			return nil, err
		}
		data.PolicyProjectGraphQLId = types.StringValue(policyProjectGid.ProjectGQLID)
	}

	return groupGid, nil
}

// verifyPolicyAssociation checks every 20 seconds for up to 1 minute to ensure the policy association applied properly
func (d *gitlabGroupSecurityPolicyAttachmentResource) verifyPolicyAssociation(ctx context.Context, data *gitlabGroupSecurityPolicyAttachmentResourceModel, groupIds *api.GroupIdentifiers) error {
	// Create a context with timeout for the verification process
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	// Check immediately first
	response, err := d.readPolicy(groupIds)
	if err != nil {
		return fmt.Errorf("failed to read policy during verification: %w", err)
	}

	if response.Data.Group != nil && response.Data.Group.SecurityPolicyProject != nil {
		// Extract the policy project ID from the GraphQL ID
		parts := strings.Split(response.Data.Group.SecurityPolicyProject.ID, "/")
		actualPolicyId := parts[len(parts)-1]

		if actualPolicyId == data.PolicyProject.ValueString() {
			tflog.Debug(ctx, "Policy association verified successfully", map[string]any{
				"group":          data.Group.ValueString(),
				"policy_project": data.PolicyProject.ValueString(),
			})
			return nil
		}
	}

	// If not immediately successful, start checking with ticker
	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("policy association verification timed out after 1 minute. Expected policy project %s to be associated with group %s", data.PolicyProject.ValueString(), data.Group.ValueString())
			}
			return timeoutCtx.Err()
		case <-ticker.C:
			response, err := d.readPolicy(groupIds)
			if err != nil {
				tflog.Warn(ctx, "Error reading policy during verification, will retry", map[string]any{
					"group": data.Group.ValueString(),
					"error": err.Error(),
				})
				continue
			}

			if response.Data.Group != nil && response.Data.Group.SecurityPolicyProject != nil {
				// Extract the policy project ID from the GraphQL ID
				parts := strings.Split(response.Data.Group.SecurityPolicyProject.ID, "/")
				actualPolicyId := parts[len(parts)-1]

				if actualPolicyId == data.PolicyProject.ValueString() {
					tflog.Debug(ctx, "Policy association verified successfully", map[string]any{
						"group":          data.Group.ValueString(),
						"policy_project": data.PolicyProject.ValueString(),
					})
					return nil
				}
			}

			tflog.Debug(ctx, "Policy association not yet applied, continuing to wait", map[string]any{
				"group":          data.Group.ValueString(),
				"policy_project": data.PolicyProject.ValueString(),
			})
		}
	}
}

type GetGroupSecurityPolicyProjectResponse struct {
	Data struct {
		Group *struct {
			SecurityPolicyProject *struct {
				ID string `json:"id"`
			} `json:"securityPolicyProject"`
			ID string `json:"id"`
		} `json:"group"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}
