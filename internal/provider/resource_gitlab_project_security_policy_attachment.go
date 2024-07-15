package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &gitlabProjectSecurityPolicyAttachmentResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectSecurityPolicyAttachmentResource{}
	_ resource.ResourceWithImportState = &gitlabProjectSecurityPolicyAttachmentResource{}
)

func init() {
	registerResource(NewGitlabProjectSecurityPolicyAttachmentResource)
}

func NewGitlabProjectSecurityPolicyAttachmentResource() resource.Resource {
	return &gitlabProjectSecurityPolicyAttachmentResource{}
}

type gitlabProjectSecurityPolicyAttachmentResource struct {
	client *gitlab.Client
}

type gitlabGroupSecurityPolicyAttachmentResourceModel struct {
	Id                     types.String `tfsdk:"id"`
	Project                types.String `tfsdk:"project"`
	PolicyProject          types.String `tfsdk:"policy_project"`
	PolicyProjectGraphQLId types.String `tfsdk:"policy_project_graphql_id"`
	ProjectGraphQLId       types.String `tfsdk:"project_graphql_id"`
}

func (r *gitlabProjectSecurityPolicyAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_security_policy_attachment"
}

func (r *gitlabProjectSecurityPolicyAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_security_policy_attachment`" + ` resource allows to attach a security policy project to a project.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/ee/api/graphql/reference/index.html#mutationsecuritypolicyprojectassign)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<policy_project>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or Full Path of the project which will have the security policy project assigned to it.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
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
			"project_graphql_id": schema.StringAttribute{
				MarkdownDescription: "The GraphQL ID of the project to which the security policty project will be attached.",
				Computed:            true,
			},
		},
	}
}

func (d *gitlabProjectSecurityPolicyAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*gitlab.Client)
}

func (d *gitlabProjectSecurityPolicyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectSecurityPolicyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabGroupSecurityPolicyAttachmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectIds, err := d.parseGraphQLIds(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse GraphQL IDs", err.Error())
		return
	}

	err = d.updatePolicy(ctx, data, projectIds)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update GraphQL ID", err.Error())
		return
	}

	// Set the ID
	data.Id = types.StringValue(utils.BuildTwoPartID(data.Project.ValueStringPointer(), data.PolicyProject.ValueStringPointer()))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabProjectSecurityPolicyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabGroupSecurityPolicyAttachmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, policyProject, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse IDs", err.Error())
	}
	data.Project = types.StringValue(project)

	// Get the GraphQL IDs of the project and group
	projectIds, err := d.parseGraphQLIds(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse GraphQL IDs", err.Error())
	}

	// Read the policy project
	query := fmt.Sprintf(`
		query {
			project(fullPath:"%s") {
				id,
				securityPolicyProject {id}
			}
		}
	`, projectIds.ProjectFullPath)
	var response GetSecurityPolicyProjectResponse
	_, err = api.SendGraphQLRequest(ctx, d.client, api.GraphQLQuery{Query: query}, &response)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read security policy project", err.Error())
		return
	}

	if response.Errors != nil && len(response.Errors) > 0 {
		resp.Diagnostics.AddError("Failed to read security policy project", response.Errors[0].Message)
	}

	if response.Data.Project == nil {
		tflog.Warn(ctx, "Project for the gitlab_project_security_policy_attachment returned nil from the GraphQL call, which usually means the project doesn't exist anymore.", map[string]interface{}{
			"project":        project,
			"policy_project": policyProject,
		})
		resp.State.RemoveResource(ctx)
	}

	// Get the policy project ID, which is the final digit in the GraphQL ID of the response
	if response.Data.Project.SecurityPolicyProject != nil && response.Data.Project.SecurityPolicyProject.ID != "" {
		parts := strings.Split(response.Data.Project.SecurityPolicyProject.ID, "/")
		parsedPolicyId := parts[len(parts)-1]

		data.PolicyProject = types.StringValue(parsedPolicyId)

		tflog.Debug(ctx, "Parsed a valid security policy project. Adding to state", map[string]interface{}{
			"project":        project,
			"policy_project": parsedPolicyId,
		})

		// Parse GraphQL IDs again to validate the policy project GID
		_, err := d.parseGraphQLIds(ctx, data)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse GraphQL ID of the policy project", err.Error())
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabProjectSecurityPolicyAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	err = d.updatePolicy(ctx, data, groupIds)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update GraphQL ID", err.Error())
		return
	}

	tflog.Debug(ctx, "Updated security policy project", map[string]interface{}{
		"project":        data.Project.ValueString(),
		"policy_project": data.PolicyProject.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabProjectSecurityPolicyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

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
	`, projectIds.ProjectFullPath)
	var response SecurityProjectUnassignResponse
	_, err = api.SendGraphQLRequest(ctx, d.client, api.GraphQLQuery{Query: query}, &response)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete the group security policy attachment", err.Error())
		return
	}

	if response.Data.SecurityPolicyProjectUnassign.Errors != nil && len(response.Data.SecurityPolicyProjectUnassign.Errors) > 0 {
		resp.Diagnostics.AddError("Failed to delete the group security policy attachment", response.Data.SecurityPolicyProjectUnassign.Errors[0].Message)
		return
	}

	tflog.Debug(ctx, "Successfully deleted security policy project", map[string]interface{}{
		"project":        data.Project.ValueString(),
		"policy_project": data.PolicyProject.ValueString(),
	})

	resp.State.RemoveResource(ctx)
}

// Update the security policy associated to the group
func (d *gitlabProjectSecurityPolicyAttachmentResource) updatePolicy(ctx context.Context, data *gitlabGroupSecurityPolicyAttachmentResourceModel, ids *api.ProjectIdentifiers) error {
	// Read the policy project
	query := fmt.Sprintf(`
		mutation {
			securityPolicyProjectAssign( input: {
				fullPath:"%s",
				securityPolicyProjectId: "%s" 
			}) {
				errors
			}
		}
	`, ids.ProjectFullPath, data.PolicyProjectGraphQLId.ValueString())
	var response SecurityProjectAssignResponse
	_, err := api.SendGraphQLRequest(ctx, d.client, api.GraphQLQuery{Query: query}, &response)
	if err != nil {
		return err
	}
	if response.Data.SecurityPolicyProjectAssign.Errors != nil && len(response.Data.SecurityPolicyProjectAssign.Errors) > 0 {
		return errors.New(response.Data.SecurityPolicyProjectAssign.Errors[0].Message)
	}

	return nil
}

type SecurityProjectAssignResponse struct {
	Data struct {
		SecurityPolicyProjectAssign struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"securityPolicyProjectAssign"`
	} `json:"data"`
}

type SecurityProjectUnassignResponse struct {
	Data struct {
		SecurityPolicyProjectUnassign struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"securityPolicyProjectUnassign"`
	} `json:"data"`
}

// Get the GraphQL IDs for the project
func (d *gitlabProjectSecurityPolicyAttachmentResource) parseGraphQLIds(ctx context.Context, data *gitlabGroupSecurityPolicyAttachmentResourceModel) (*api.ProjectIdentifiers, error) {
	// Get the GraphQL of the project Id
	projectGid, err := api.GetProjectGIDFromID(ctx, d.client, data.Project.ValueString())
	if err != nil {
		return nil, err
	}
	data.ProjectGraphQLId = types.StringValue(projectGid.ProjectGQLID)

	// Get the GraphQL of the policy project ID
	if !data.PolicyProject.IsUnknown() && !data.PolicyProject.IsNull() {
		policyProjectGid, err := api.GetProjectGIDFromID(ctx, d.client, data.PolicyProject.ValueString())
		if err != nil {
			return nil, err
		}
		data.PolicyProjectGraphQLId = types.StringValue(policyProjectGid.ProjectGQLID)
	}

	return projectGid, nil
}

type GetSecurityPolicyProjectResponse struct {
	Data struct {
		Project *struct {
			SecurityPolicyProject *struct {
				ID string `json:"id"`
			} `json:"securityPolicyProject"`
			ID string `json:"id"`
		} `json:"project"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}
