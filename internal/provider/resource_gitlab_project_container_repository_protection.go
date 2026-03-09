package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	_ resource.Resource                = &gitlabContainerRepositoryProtectionResource{}
	_ resource.ResourceWithConfigure   = &gitlabContainerRepositoryProtectionResource{}
	_ resource.ResourceWithImportState = &gitlabContainerRepositoryProtectionResource{}
)

func init() {
	registerResource(NewGitLabContainerRepositoryProtectionRuleResource)
}

func NewGitLabContainerRepositoryProtectionRuleResource() resource.Resource {
	return &gitlabContainerRepositoryProtectionResource{}
}

type gitlabContainerRepositoryProtectionResource struct {
	client *gitlab.Client
}

type gitlabContainerRepositoryProtectionModel struct {
	ID                          types.String `tfsdk:"id"`
	Project                     types.String `tfsdk:"project"`
	ProtectionRuleID            types.Int64  `tfsdk:"protection_rule_id"`
	RepositoryPathPattern       types.String `tfsdk:"repository_path_pattern"`
	MinimumAccessLevelForPush   types.String `tfsdk:"minimum_access_level_for_push"`
	MinimumAccessLevelForDelete types.String `tfsdk:"minimum_access_level_for_delete"`
}

func (r *gitlabContainerRepositoryProtectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_container_repository_protection"
}

func (r *gitlabContainerRepositoryProtectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_container_repository_protection`" + ` resource allows managing the lifecycle of a container repository protection rule.

You can use a wildcard (*) to protect multiple container repositories with the same container protection rule.
You can apply several protection rules to the same container repository. A container repository is protected if at least one protection rule matches.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/container_repository_protection_rules/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<protection_rule_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or URL-encoded path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"protection_rule_id": schema.Int64Attribute{
				MarkdownDescription: "Unique ID of the protection rule.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"repository_path_pattern": schema.StringAttribute{
				MarkdownDescription: "Container repository path pattern protected by the protection rule. Wildcard character * allowed. Repository path pattern should start with the project's full path",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"minimum_access_level_for_push": schema.StringAttribute{
				MarkdownDescription: "Minimum GitLab access level required to push container images to the container registry. For example maintainer, owner or admin. Must be provided when `minimum_access_level_for_delete` is not set.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(api.ValidProtectedContainerRepositoryAccessLevelNames...),
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("minimum_access_level_for_push"), path.MatchRelative().AtParent().AtName("minimum_access_level_for_delete")),
				},
			},
			"minimum_access_level_for_delete": schema.StringAttribute{
				MarkdownDescription: "Minimum GitLab access level required to delete container images in the container registry. For example maintainer, owner, admin. Must be provided when `minimum_access_level_for_push` is not set.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(api.ValidProtectedContainerRepositoryAccessLevelNames...),
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("minimum_access_level_for_delete"), path.MatchRelative().AtParent().AtName("minimum_access_level_for_push")),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabContainerRepositoryProtectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabContainerRepositoryProtectionModel) containerRepositoryProtectionRuleModelToState(rule *gitlab.ContainerRegistryProtectionRule, projectID string) {
	r.ProtectionRuleID = types.Int64Value(int64(rule.ID))
	r.Project = types.StringValue(projectID)
	r.RepositoryPathPattern = types.StringValue(rule.RepositoryPathPattern)

	if rule.MinimumAccessLevelForDelete != "" {
		r.MinimumAccessLevelForDelete = types.StringValue(string(rule.MinimumAccessLevelForDelete))
	}

	if rule.MinimumAccessLevelForPush != "" {
		r.MinimumAccessLevelForPush = types.StringValue(string(rule.MinimumAccessLevelForPush))
	}
}

func (r *gitlabContainerRepositoryProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabContainerRepositoryProtectionModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Project.ValueString()

	options := &gitlab.CreateContainerRegistryProtectionRuleOptions{
		RepositoryPathPattern: gitlab.Ptr(data.RepositoryPathPattern.ValueString()),
	}

	if !data.MinimumAccessLevelForPush.IsNull() && !data.MinimumAccessLevelForPush.IsUnknown() {
		options.MinimumAccessLevelForPush = gitlab.Ptr(gitlab.ProtectionRuleAccessLevel(data.MinimumAccessLevelForPush.ValueString()))
	}
	if !data.MinimumAccessLevelForDelete.IsNull() && !data.MinimumAccessLevelForDelete.IsUnknown() {
		options.MinimumAccessLevelForDelete = gitlab.Ptr(gitlab.ProtectionRuleAccessLevel(data.MinimumAccessLevelForDelete.ValueString()))
	}

	rule, _, err := r.client.ContainerRegistryProtectionRules.CreateContainerRegistryProtectionRule(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create container repository protection rule: %s", err.Error()))
		return
	}

	data.containerRepositoryProtectionRuleModelToState(rule, projectID)
	data.ID = types.StringValue(utils.BuildTwoPartID(gitlab.Ptr(projectID), gitlab.Ptr(strconv.Itoa(int(data.ProtectionRuleID.ValueInt64())))))

	// Log the creation of the resource
	tflog.Debug(ctx, "created a container repository protection rule", map[string]any{
		"project":                 data.Project.ValueString(),
		"repository_path_pattern": data.RepositoryPathPattern.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabContainerRepositoryProtectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabContainerRepositoryProtectionModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, rawRuleID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<project>:<protection_rule_id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	ruleID, err := strconv.ParseInt(rawRuleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid container repository protection rule ID provided, container repository protection rule ID should be an Int",
			fmt.Sprintf("Unable to convert container repository protection rule ID to int: %s", err.Error()),
		)
		return
	}

	rules, _, err := r.client.ContainerRegistryProtectionRules.ListContainerRegistryProtectionRules(projectID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project does not exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read container repository protection rules: %s", err.Error()))
		return
	}

	found, rule := r.containsID(rules, ruleID)

	if found {
		data.containerRepositoryProtectionRuleModelToState(rule, projectID)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	} else {
		tflog.Debug(ctx, "container repository protection rule does not exist, removing from state", map[string]any{
			"project": data.Project,
			"rule_id": ruleID,
		})
		resp.State.RemoveResource(ctx)
	}
}

func (r *gitlabContainerRepositoryProtectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabContainerRepositoryProtectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.UpdateContainerRegistryProtectionRuleOptions{
		RepositoryPathPattern: gitlab.Ptr(data.RepositoryPathPattern.ValueString()),
	}

	if !data.MinimumAccessLevelForPush.IsNull() && !data.MinimumAccessLevelForPush.IsUnknown() {
		options.MinimumAccessLevelForPush = gitlab.Ptr(gitlab.ProtectionRuleAccessLevel(data.MinimumAccessLevelForPush.ValueString()))
	}
	if !data.MinimumAccessLevelForDelete.IsNull() && !data.MinimumAccessLevelForDelete.IsUnknown() {
		options.MinimumAccessLevelForDelete = gitlab.Ptr(gitlab.ProtectionRuleAccessLevel(data.MinimumAccessLevelForDelete.ValueString()))
	}

	projectID := data.Project.ValueString()
	ruleID := data.ProtectionRuleID.ValueInt64()
	rule, _, err := r.client.ContainerRegistryProtectionRules.UpdateContainerRegistryProtectionRule(projectID, ruleID, options, gitlab.WithContext(ctx))

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating container repository protection rule for project %s", data.Project),
			err.Error(),
		)
		return
	}

	data.containerRepositoryProtectionRuleModelToState(rule, projectID)

	tflog.Debug(ctx, "updated container repository protection rule", map[string]interface{}{
		"project": data.Project.ValueString(), "repository_path_pattern": data.RepositoryPathPattern.ValueString(),
	})

	// Set our plan object into state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabContainerRepositoryProtectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabContainerRepositoryProtectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, rawRuleID, err := utils.ParseTwoPartID(data.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<protection_rule_id>'. Error: %s", data.ID, err.Error()),
		)
		return
	}

	ruleID, err := strconv.ParseInt(rawRuleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid container repository protection rule ID provided, container repository protection rule ID should be an Int",
			fmt.Sprintf("Unable to convert container repository protection rule ID to int: %s", err.Error()),
		)
		return
	}

	_, err = r.client.ContainerRegistryProtectionRules.DeleteContainerRegistryProtectionRule(projectID, ruleID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete container repository protection rule: %s", err.Error()),
		)
	}

	resp.State.RemoveResource(ctx)
}

func (r *gitlabContainerRepositoryProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabContainerRepositoryProtectionResource) containsID(rules []*gitlab.ContainerRegistryProtectionRule, ruleID int64) (bool, *gitlab.ContainerRegistryProtectionRule) {
	for _, rule := range rules {
		if rule.ID == ruleID {
			return true, rule
		}
	}
	return false, nil
}
