package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectTargetBranchRule{}
	_ resource.ResourceWithConfigure   = &gitlabProjectTargetBranchRule{}
	_ resource.ResourceWithImportState = &gitlabProjectTargetBranchRule{}
)

func init() {
	registerResource(newGitLabProjectTargetBranchRule)
}

func newGitLabProjectTargetBranchRule() resource.Resource {
	return &gitlabProjectTargetBranchRule{}
}

func (r *gitlabProjectTargetBranchRule) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_target_branch_rule"
}

type gitlabProjectTargetBranchRule struct {
	client *gitlab.Client
}

type gitlabProjectTargetBranchRuleModel struct {
	Id                  types.String `tfsdk:"id"`
	Project             types.String `tfsdk:"project"`
	SourceBranchPattern types.String `tfsdk:"source_branch_pattern"`
	TargetBranchName    types.String `tfsdk:"target_branch_name"`
}

func (r *gitlabProjectTargetBranchRule) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_project_target_branch_rule`" + ` resource manages default target branch rules when creating merge requests.

**Upstream API**: [GitLab GraphQL API docs](https://docs.gitlab.com/api/graphql/reference/#mutationprojecttargetbranchrulecreate)`),

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<target-branch-rule-id>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or URL-encoded path of the project.",
				Required:            true,
			},
			"source_branch_pattern": schema.StringAttribute{
				MarkdownDescription: "A pattern matching the branch name for which the merge request should have a default target branch configured.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_branch_name": schema.StringAttribute{
				MarkdownDescription: "The name of the branch to which the merge request should be addressed.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *gitlabProjectTargetBranchRule) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectTargetBranchRule) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectTargetBranchRuleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	projectID := data.Project.ValueString()

	projectGID, err := api.GetProjectGIDFromID(ctx, r.client, projectID)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project GID: %s", err.Error()))
		return
	}

	tflog.Trace(ctx, "found project", map[string]any{
		"project": gitlab.Stringify(projectGID),
	})

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				projectTargetBranchRuleCreate(input:{projectId:"%s", name:"%s", targetBranch:"%s"}) {
					errors
					targetBranchRule {
					id
					name
					}
				}
			}`, projectGID.ProjectGQLID, data.SourceBranchPattern.ValueString(), data.TargetBranchName.ValueString()),
	}

	tflog.Debug(ctx, "executing GraphQL Query to create gitlab_project_target_branch_rule", map[string]any{
		"query": query.Query,
	})
	var response getProjectTargetBranchRuleCreateResponse
	_, err = r.client.GraphQL.Do(query, &response)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create gitlab_project_target_branch_rule: %s", err.Error()))
		return
	}

	if len(response.Data.ProjectTargetBranchRuleCreate.Errors) > 0 {
		for _, respErrors := range response.Data.ProjectTargetBranchRuleCreate.Errors {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create gitlab_project_target_branch_rule: %s", respErrors))
			return
		}
	}

	tflog.Debug(ctx, "response from GraphQL Query to create gitlab_project_target_branch_rule", map[string]any{
		"response": response,
	})

	data.Id = types.StringValue(utils.BuildTwoPartID(&projectID, &response.Data.ProjectTargetBranchRuleCreate.TargetBranchRule.Id))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectTargetBranchRule) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectTargetBranchRuleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	projectID, targetBranchRuleID, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<target-branch-rule-id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	project, _, err := r.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project does not exist, removing resource from state", map[string]any{
				"project": data.Project.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project details: %s", err.Error()))
		return
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			query {
				project(fullPath:"%s") {
					targetBranchRules {
						nodes {
							id
							name
							targetBranch
						}
					}
				}
			}`, project.PathWithNamespace),
	}

	tflog.Debug(ctx, "executing GraphQL Query to read gitlab_project_target_branch_rule", map[string]any{
		"query": query.Query,
	})

	var response getProjectTargetBranchRuleReadResponse
	_, err = r.client.GraphQL.Do(query, &response)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read gitlab_project_target_branch_rule: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, "response from GraphQL Query to read gitlab_project_target_branch_rule", map[string]any{
		"response": response,
	})

	var value targetBranchRule
	for _, rule := range response.Data.Project.TargetBranchRules.Nodes {
		if rule.Id == targetBranchRuleID {
			value = rule
		}
	}

	if value.Id != targetBranchRuleID {
		tflog.Debug(ctx, "targetBranchRuleID not found in project, removing resource from state", map[string]any{
			"project": data.Project.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.Project = types.StringValue(projectID)
	data.SourceBranchPattern = types.StringValue(value.Name)
	data.TargetBranchName = types.StringValue(value.TargetBranch)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectTargetBranchRule) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Provider Error, report upstream",
		"Somehow the resource was requested to perform an in-place upgrade which is not possible.",
	)
}

func (r *gitlabProjectTargetBranchRule) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectTargetBranchRuleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, targetBranchRuleID, err := utils.ParseTwoPartID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<target-branch-rule-id>'. Error: %s", data.Id.ValueString(), err.Error()),
		)
		return
	}

	query := gitlab.GraphQLQuery{
		Query: fmt.Sprintf(`
			mutation {
				projectTargetBranchRuleDestroy(
					input:{id:"%[1]s", clientMutationId: "projectTargetBranchRuleDestroy"}) {
    				errors
					clientMutationId
				}
			}`, targetBranchRuleID),
	}

	tflog.Debug(ctx, "executing GraphQL Query to delete gitlab_project_target_branch_rule", map[string]any{
		"query": query.Query,
	})

	var response getProjectTargetBranchRuleDeleteResponse
	_, err = r.client.GraphQL.Do(query, &response)
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete gitlab_project_target_branch_rule: %s", err.Error()))
		return
	}

	if response.Data.ProjectTargetBranchRuleDestroy.ClientMutationID != "projectTargetBranchRuleDestroy" {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("No mutation id was returned by GraphQL API: %s", map[string]any{
			"response": response,
		}))
		return
	}

	if len(response.Data.ProjectTargetBranchRuleDestroy.Errors) > 0 {
		for _, respErrors := range response.Data.ProjectTargetBranchRuleDestroy.Errors {
			resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to delete gitlab_project_target_branch_rule: %s", respErrors))
			return
		}
	}

	tflog.Debug(ctx, "response from GraphQL Query to delete gitlab_project_target_branch_rule", map[string]any{
		"response": response,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectTargetBranchRule) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type getProjectTargetBranchRuleCreateResponse struct {
	Data struct {
		ProjectTargetBranchRuleCreate struct {
			Errors           []string         `json:"errors"`
			TargetBranchRule targetBranchRule `json:"targetBranchRule"`
		} `json:"projectTargetBranchRuleCreate"`
	} `json:"data"`
}

type getProjectTargetBranchRuleReadResponse struct {
	Data struct {
		Project struct {
			TargetBranchRules struct {
				Nodes []targetBranchRule `json:"nodes"`
			} `json:"targetBranchRules"`
		} `json:"project"`
	} `json:"data"`
}

type getProjectTargetBranchRuleDeleteResponse struct {
	Data struct {
		ProjectTargetBranchRuleDestroy struct {
			Errors           []string `json:"errors"`
			ClientMutationID string   `json:"clientMutationId"`
		} `json:"projectTargetBranchRuleDestroy"`
	} `json:"data"`
}

type targetBranchRule struct {
	CreatedAt    string `json:"createdAt"`
	Id           string `json:"id"`
	Name         string `json:"name"`
	TargetBranch string `json:"targetBranch"`
}
