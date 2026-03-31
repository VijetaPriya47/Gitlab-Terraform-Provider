package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
	_ resource.Resource                = &gitlabContainerTagProtectionResource{}
	_ resource.ResourceWithConfigure   = &gitlabContainerTagProtectionResource{}
	_ resource.ResourceWithImportState = &gitlabContainerTagProtectionResource{}

	protectionRuleRegex                      = regexp.MustCompile(`^gid://gitlab/ContainerRegistry::Protection::TagRule/(?P<id>\d+)$`)
	numberOfTagProtectionRulesReturnedOnRead = 100 // 100 is the maximum number of rules returned without pagination ; current API limits the number of rules to 5
	errRuleNotFound                          = errors.New("container tag protection rule not found")
)

func init() {
	registerResource(NewGitLabContainerTagProtectionRuleResource)
}

// NewGitLabContainerTagProtectionRuleResource is a helper function to simplify the provider implementation
func NewGitLabContainerTagProtectionRuleResource() resource.Resource {
	return &gitlabContainerTagProtectionResource{}
}

// gitlabContainerTagProtectionResource represents the resource implementation
type gitlabContainerTagProtectionResource struct {
	client *gitlab.Client
}

// gitlabContainerTagProtectionRulesModel represents a container tag protection rule from GraphQL API
type gitlabContainerTagProtectionRulesModel struct {
	ID                          types.String `tfsdk:"id"`
	Project                     types.String `tfsdk:"project"`
	ProtectionRuleID            types.Int64  `tfsdk:"protection_rule_id"`
	TagNameRegex                types.String `tfsdk:"tag_name_regex"`
	Immutable                   types.Bool   `tfsdk:"immutable"`
	MinimumAccessLevelForPush   types.String `tfsdk:"minimum_access_level_for_push"`
	MinimumAccessLevelForDelete types.String `tfsdk:"minimum_access_level_for_delete"`

	// Timeouts meta block
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata returns the resource type name
func (r *gitlabContainerTagProtectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_container_tag_protection"
}

// upperCaseAccessLevelNames returns the access level names in uppercase
func upperCaseAccessLevelNames() []string {
	upperCaseAccessLevelNames := make([]string, len(api.ValidProtectedContainerRepositoryAccessLevelNames))
	for i, accessLevelName := range api.ValidProtectedContainerRepositoryAccessLevelNames {
		upperCaseAccessLevelNames[i] = strings.ToUpper(accessLevelName)
	}
	return upperCaseAccessLevelNames
}

// Schema returns the schema for the resource
func (r *gitlabContainerTagProtectionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_container_tag_protection`" + ` resource allows managing the lifecycle of a container tag protection rule.

You can use a regular expression to protect multiple container tags with the same container protection rule.
You have to **either set** (both options are mutually exclusive):

- Only ` + "`immutable = true`" + ` which allows any user with ` + "`developer`" + ` role to create tags matching the regex but makes the object unchangeable once pushed ;
- Both ` + "`minimum_access_level_for_push`" + ` and ` + "`minimum_access_level_for_delete`" + ` attributes to mark the container tag as mutable only with regards to the given access levels.

**Upstream API**: [GitLab GraphQL API documentation](https://docs.gitlab.com/api/graphql/reference/#mutationcreatecontainerprotectiontagrule)

**Protected tags**: [General documentation](https://docs.gitlab.com/user/packages/container_registry/protected_container_tags/)

**Immutable tags**: [General documentation](https://docs.gitlab.com/user/packages/container_registry/immutable_container_tags/)
`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<protection_rule_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"protection_rule_id": schema.Int64Attribute{
				MarkdownDescription: "Unique ID of the protection rule.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or URL-encoded path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tag_name_regex": schema.StringAttribute{
				MarkdownDescription: "Container tag path pattern protected by the protection rule. Wildcard character * allowed. Tag path pattern should start with the project's full path.",
				Required:            true,
				// All fields of the mutation to update protection rules are deprecated
				// https://docs.gitlab.com/api/graphql/reference/#mutationupdatecontainerprotectiontagrule
				// and does not work with existing tag at the time of this writing
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "Whether the container tag is immutable. If set, the container tag cannot be deleted or overwritten. Conflicts with `minimum_access_level_for_push` and `minimum_access_level_for_delete`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("minimum_access_level_for_push"),
						path.MatchRelative().AtParent().AtName("minimum_access_level_for_delete"),
					),
				},
				// All fields of the mutation to update protection rules are deprecated
				// https://docs.gitlab.com/api/graphql/reference/#mutationupdatecontainerprotectiontagrule
				// and does not work with existing tag at the time of this writing
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"minimum_access_level_for_push": schema.StringAttribute{
				MarkdownDescription: "Minimum GitLab access level required to push protected container tags to the container registry. Marks the container tag as protected. Valid values are: " + utils.RenderValueListForDocs(upperCaseAccessLevelNames()) + ".",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(upperCaseAccessLevelNames()...),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("immutable")),
					stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("minimum_access_level_for_delete")),
				},
				// All fields of the mutation to update protection rules are deprecated
				// https://docs.gitlab.com/api/graphql/reference/#mutationupdatecontainerprotectiontagrule
				// and does not work with existing tag at the time of this writing
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"minimum_access_level_for_delete": schema.StringAttribute{
				MarkdownDescription: "Minimum GitLab access level required to delete protected container tags in the container registry. Marks the container tag as protected. Valid values are: " + utils.RenderValueListForDocs(upperCaseAccessLevelNames()) + ".",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(upperCaseAccessLevelNames()...),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("immutable")),
					stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("minimum_access_level_for_push")),
				},
				// All fields of the mutation to update protection rules are deprecated
				// https://docs.gitlab.com/api/graphql/reference/#mutationupdatecontainerprotectiontagrule
				// and does not work with existing tag at the time of this writing
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: "How long to wait for the container tag protection rule to be created. Defaults to 5 minutes.",
			}),
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabContainerTagProtectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

// ContainerTagProtectionRule represents a container tag protection rule from GraphQL API
type ContainerTagProtectionRule struct {
	ID                          string  `json:"id"`
	TagNameRegex                string  `json:"tagNamePattern"`
	MinimumAccessLevelForPush   *string `json:"minimumAccessLevelForPush,omitempty"`
	MinimumAccessLevelForDelete *string `json:"minimumAccessLevelForDelete,omitempty"`
	Immutable                   bool    `json:"immutable"`
}

// containerTagProtectionRuleModelToState converts a ContainerTagProtectionRule to a gitlabContainerTagProtectionRulesModel
func (r *gitlabContainerTagProtectionRulesModel) containerTagProtectionRuleModelToState(rule ContainerTagProtectionRule, project string, projectID int64) error {
	matches := protectionRuleRegex.FindStringSubmatch(rule.ID)
	if len(matches) != 2 {
		return fmt.Errorf("unexpected container tag protection rule ID can not be parsed with regex %s: %s", protectionRuleRegex.String(), rule.ID)
	}
	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid container tag protection rule ID %s can not be converted to int: %w", rule.ID, err)
	}

	r.ProtectionRuleID = types.Int64Value(id)
	if project != "" {
		r.Project = types.StringValue(project)
	} else {
		r.Project = types.StringValue(strconv.FormatInt(projectID, 10))
	}

	r.TagNameRegex = types.StringValue(rule.TagNameRegex)
	r.Immutable = types.BoolValue(rule.Immutable)
	r.MinimumAccessLevelForPush = types.StringPointerValue(rule.MinimumAccessLevelForPush)
	r.MinimumAccessLevelForDelete = types.StringPointerValue(rule.MinimumAccessLevelForDelete)

	return nil
}

// Create creates a new container tag protection rule
func (r *gitlabContainerTagProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabContainerTagProtectionRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()

	projectMetadata, _, err := r.client.Projects.GetProject(project, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}
	tagNameRegex := data.TagNameRegex.ValueString()
	input := map[string]any{
		"projectPath":                 projectMetadata.PathWithNamespace,
		"tagNamePattern":              tagNameRegex,
		"minimumAccessLevelForPush":   nil,
		"minimumAccessLevelForDelete": nil,
	}
	// There is no explicit field in the Create/Update mutation for setting the "Immutable" property.
	// Creating an immutable rule is simply done by passing in `nil` for both the push/delete levels,
	// so this block checks if immutable is set and prevents updating those two fields in the mutation.
	if !data.Immutable.ValueBool() &&
		!data.MinimumAccessLevelForPush.IsNull() &&
		!data.MinimumAccessLevelForPush.IsUnknown() &&
		!data.MinimumAccessLevelForDelete.IsNull() &&
		!data.MinimumAccessLevelForDelete.IsUnknown() {
		input["minimumAccessLevelForPush"] = data.MinimumAccessLevelForPush.ValueString()
		input["minimumAccessLevelForDelete"] = data.MinimumAccessLevelForDelete.ValueString()
	}

	query := gitlab.GraphQLQuery{
		// We do not leverage the returned data.containerProtectionTagRule object because it
		// is marked as deprecated in the API documentation
		// https://docs.gitlab.com/api/graphql/reference/#mutationcreatecontainerprotectiontagrule
		Query: `
          mutation createContainerProtectionTagRule($input: createContainerProtectionTagRuleInput!) {
            createContainerProtectionTagRule(input: $input) {
              errors
            }
          }
		`,
		Variables: map[string]any{
			"input": input,
		},
	}
	var response struct {
		Data struct {
			Errors []string `json:"errors"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	_, err = r.client.GraphQL.Do(query, &response, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create container tag protection rule: %s", err.Error()))
		return
	} else if len(response.Errors) > 0 {
		resp.Diagnostics.AddError("GitLab GraphQL API error occurred", fmt.Sprintf("GraphQL errors: %v", response.Errors))
		return
	} else if len(response.Data.Errors) > 0 {
		resp.Diagnostics.AddError("GitLab GraphQL API error occurred", fmt.Sprintf("GraphQL errors: %v", response.Data.Errors))
		return
	}

	// Instead of leveraging the deprecated data.containerProtectionTagRule object, we fetch the rule by name regex
	// https://docs.gitlab.com/api/graphql/reference/#mutationcreatecontainerprotectiontagrule
	// Because the broadcasting of the rule creation is not necessary immediate due to eventual consistency,
	// a few retries and initial delay are necessary to accomodate for latency in propagation.
	rule, err := r.waitForRuleCreation(ctx, &data, resp, projectMetadata.PathWithNamespace, project, tagNameRegex)
	if err != nil || resp.Diagnostics.HasError() {
		return
	}

	if err = data.containerTagProtectionRuleModelToState(rule, project, projectMetadata.ID); err != nil {
		resp.Diagnostics.AddError("Unexpected provider error occurred", err.Error())
		return
	}
	data.ID = types.StringValue(utils.BuildTwoPartID(
		gitlab.Ptr(strconv.FormatInt(projectMetadata.ID, 10)),
		gitlab.Ptr(strconv.FormatInt(data.ProtectionRuleID.ValueInt64(), 10)),
	))
	tflog.Debug(ctx, "created a container tag protection rule", map[string]any{
		"rule_id":        data.ProtectionRuleID.ValueInt64(),
		"project":        data.Project.ValueString(),
		"tag_name_regex": data.TagNameRegex.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the container tag protection rule from the GitLab API
func (r *gitlabContainerTagProtectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabContainerTagProtectionRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := data.Project.ValueString()

	projectID, rawRuleID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<project>:<protection_rule_id>'. Error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}
	projectMetadata, _, err := r.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get project: %s", err.Error()))
		return
	}

	ruleID, err := strconv.ParseInt(rawRuleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid container tag protection rule ID provided, container tag protection rule ID should be an Int",
			fmt.Sprintf("Unable to convert container tag protection rule ID to int: %s", err.Error()),
		)
		return
	}
	rule, err := r.getRuleByID(ctx, projectMetadata.PathWithNamespace, ruleID)
	if err == errRuleNotFound {
		tflog.Debug(ctx, "container tag protection rule does not exist, removing from state", map[string]any{
			"project": data.Project,
			"rule_id": ruleID,
		})
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to get container tag protection rule: %s", err.Error()))
		return
	}
	if err = data.containerTagProtectionRuleModelToState(rule, project, projectMetadata.ID); err != nil {
		resp.Diagnostics.AddError("Unexpected provider error occurred", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is not supported for this resource and all changes should result in a new resource being created
func (r *gitlabContainerTagProtectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields in https://docs.gitlab.com/api/graphql/reference/#mutationupdatecontainerprotectiontagrule
	// are marked as deprecated so we prefer doing delete/recreate operations instead
	resp.Diagnostics.AddError(
		"Provider Error, report upstream",
		"Somehow the resource was requested to perform an in-place update which is not expected to happen because the underlying API is deprecated: https://docs.gitlab.com/api/graphql/reference/#mutationupdatecontainerprotectiontagrule",
	)
}

// Delete deletes the container tag protection rule from the GitLab API
func (r *gitlabContainerTagProtectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabContainerTagProtectionRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, ruleID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<protection_rule_id>'. Error: %s", data.ID, err.Error()),
		)
		return
	}

	query := gitlab.GraphQLQuery{
		Query: `
          mutation deleteContainerProtectionTagRule($input: DeleteContainerProtectionTagRuleInput!) {
            deleteContainerProtectionTagRule(input: $input) {
              errors
            }
          }
        `,
		Variables: map[string]any{
			"input": map[string]any{
				"id": fmt.Sprintf("gid://gitlab/ContainerRegistry::Protection::TagRule/%s", ruleID),
			},
		},
	}
	var response struct {
		Data struct {
			DeleteContainerProtectionTagRule struct {
				Errors []string `json:"errors"`
			} `json:"deleteContainerProtectionTagRule"`
		} `json:"data"`
	}
	_, err = r.client.GraphQL.Do(query, &response, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "container tag protection rule does not exist", map[string]any{
				"project": data.Project,
				"rule_id": ruleID,
			})
		} else {
			resp.Diagnostics.AddError(
				"GitLab API Error occurred",
				fmt.Sprintf("Unable to delete container tag protection rule: %s", err.Error()),
			)
			return
		}

	} else if len(response.Data.DeleteContainerProtectionTagRule.Errors) > 0 {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("GraphQL errors: %v", response.Data.DeleteContainerProtectionTagRule.Errors),
		)
		return
	}

	resp.State.RemoveResource(ctx)
}

// ImportState imports the container tag protection rule from the GitLab API
func (r *gitlabContainerTagProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// waitForRuleCreation polls for the container tag protection rule to be created.
// Due to eventual consistency, the rule may not be immediately available after creation.
// This function retries fetching the rule every 5 seconds until it's found or the context times out.
func (r *gitlabContainerTagProtectionResource) waitForRuleCreation(ctx context.Context, data *gitlabContainerTagProtectionRulesModel, resp *resource.CreateResponse, projectPath, project, tagNameRegex string) (ContainerTagProtectionRule, error) {
	createTimeout, diags := data.Timeouts.Create(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return ContainerTagProtectionRule{}, fmt.Errorf("failed to get create timeout")
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	done := ctx.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			if ctx.Err() == context.DeadlineExceeded {
				resp.Diagnostics.AddError("Rule not found after creation", "Timeout while waiting for rule to be created")
				return ContainerTagProtectionRule{}, fmt.Errorf("Timeout while waiting for rule to be created")
			}
			return ContainerTagProtectionRule{}, ctx.Err()
		case <-ticker.C:
			rule, err := r.getRuleByTagNameRegex(ctx, projectPath, tagNameRegex)
			if err == errRuleNotFound {
				tflog.Warn(ctx, "Failed to find rule post-create, retrying to get rule", map[string]any{
					"project":        project,
					"tag_name_regex": tagNameRegex,
				})
				continue
			}
			if err != nil {
				return ContainerTagProtectionRule{}, fmt.Errorf("Unable to get container tag protection rule by name regex after creating it: %w", err)
			}

			return rule, nil
		}
	}
}

// getRuleByID gets the container tag protection rule by ID
func (r *gitlabContainerTagProtectionResource) getRuleByID(ctx context.Context, projectPath string, ruleID int64) (ContainerTagProtectionRule, error) {
	rules, err := r.listRules(ctx, projectPath)
	if err != nil {
		return ContainerTagProtectionRule{}, fmt.Errorf("failed to list container tag protection rules: %w", err)
	}

	for _, rule := range rules {
		if matches := protectionRuleRegex.FindStringSubmatch(rule.ID); len(matches) == 2 && matches[1] == strconv.FormatInt(ruleID, 10) {
			return rule, nil
		}
	}
	return ContainerTagProtectionRule{}, errRuleNotFound
}

// getRuleByTagNameRegex gets the container tag protection rule by tag name regex
func (r *gitlabContainerTagProtectionResource) getRuleByTagNameRegex(ctx context.Context, projectPath string, tagNameRegex string) (ContainerTagProtectionRule, error) {
	rules, err := r.listRules(ctx, projectPath)
	if err != nil {
		return ContainerTagProtectionRule{}, fmt.Errorf("failed to list container tag protection rules: %w", err)
	}

	for _, rule := range rules {
		if rule.TagNameRegex == tagNameRegex {
			return rule, nil
		}
	}
	return ContainerTagProtectionRule{}, errRuleNotFound
}

// listRules lists all container tag protection rules for a project
func (r *gitlabContainerTagProtectionResource) listRules(ctx context.Context, projectPath string) ([]ContainerTagProtectionRule, error) {
	query := gitlab.GraphQLQuery{
		Query: `
          query getProjectContainerProtectionTagRules($projectPath: ID!) {
            project(fullPath: $projectPath) {
              id
              containerProtectionTagRules(first: ` + strconv.Itoa(numberOfTagProtectionRulesReturnedOnRead) + `) {
                nodes {
                  tagNamePattern
                  minimumAccessLevelForPush
                  minimumAccessLevelForDelete
                  id
                  immutable
                }
              }
            }
          }
        `,
		Variables: map[string]any{
			"projectPath": projectPath,
		},
	}
	var response struct {
		Data struct {
			Project struct {
				ContainerProtectionTagRules struct {
					Nodes []ContainerTagProtectionRule `json:"nodes"`
				} `json:"containerProtectionTagRules"`
			} `json:"project"`
		} `json:"data"`
	}
	_, err := r.client.GraphQL.Do(query, &response, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GitLab API error occured: %w", err)
	}

	return response.Data.Project.ContainerProtectionTagRules.Nodes, nil
}
