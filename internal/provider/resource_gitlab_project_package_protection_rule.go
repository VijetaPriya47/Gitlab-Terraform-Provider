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
	_ resource.Resource                = &gitlabPackageProtectionResourceRule{}
	_ resource.ResourceWithConfigure   = &gitlabPackageProtectionResourceRule{}
	_ resource.ResourceWithImportState = &gitlabPackageProtectionResourceRule{}
)

func init() {
	registerResource(NewGitLabPackageProtectionRuleResource)
}

func NewGitLabPackageProtectionRuleResource() resource.Resource {
	return &gitlabPackageProtectionResourceRule{}
}

type gitlabPackageProtectionResourceRule struct {
	client *gitlab.Client
}

type gitlabPackageProtectionModel struct {
	ID                          types.String `tfsdk:"id"`
	Project                     types.String `tfsdk:"project"`
	PackageProtectionRuleID     types.Int64  `tfsdk:"package_protection_rule_id"`
	PackageNamePattern          types.String `tfsdk:"package_name_pattern"`
	PackageType                 types.String `tfsdk:"package_type"`
	MinimumAccessLevelForPush   types.String `tfsdk:"minimum_access_level_for_push"`
	MinimumAccessLevelForDelete types.String `tfsdk:"minimum_access_level_for_delete"`
}

func (r *gitlabPackageProtectionResourceRule) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_package_protection_rule"
}

func (r *gitlabPackageProtectionResourceRule) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_package_protection_rule`" + ` resource allows managing the lifecycle of a package protection rule.

You can use a wildcard (*) to protect multiple packages with the same package protection rule.
You can apply several protection rules to the same package. A package is protected if at least one protection rule matches.

**Upstream API**: [GitLab API docs](https://docs.gitlab.com/api/project_packages_protection_rules/)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format of `<project>:<package_protection_rule_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "ID or URL-encoded path of the project.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"package_protection_rule_id": schema.Int64Attribute{
				MarkdownDescription: "Unique ID of the protection rule.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"package_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Package name pattern protected by the protection rule. For example `@scope/package-*`. Wildcard character `*` allowed.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"package_type": schema.StringAttribute{
				MarkdownDescription: "Package type protected by the protection rule. For example npm.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"minimum_access_level_for_push": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Minimum GitLab access level required to push packages to the package registry. Valid values are: %s. Must be provided when `minimum_access_level_for_delete` is not set.", utils.RenderValueListForDocs(api.ValidPackageProtectionPushRuleAccessLevelNames)),
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(api.ValidPackageProtectionPushRuleAccessLevelNames...),
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("minimum_access_level_for_delete"), path.MatchRelative().AtParent().AtName("minimum_access_level_for_push")),
				},
			},
			"minimum_access_level_for_delete": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Minimum GitLab access level required to delete packages from the package registry. Valid values are: %s. Must be provided when `minimum_access_level_for_push` is not set.", utils.RenderValueListForDocs(api.ValidPackageProtectionPushRuleAccessLevelNames)),
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(api.ValidPackageProtectionPushRuleAccessLevelNames...),
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("minimum_access_level_for_push"), path.MatchRelative().AtParent().AtName("minimum_access_level_for_delete")),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *gitlabPackageProtectionResourceRule) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabPackageProtectionModel) packageProtectionRuleModelToState(rule *gitlab.PackageProtectionRule, projectID string) {
	r.PackageProtectionRuleID = types.Int64Value(rule.ID)
	r.Project = types.StringValue(projectID)
	r.PackageNamePattern = types.StringValue(rule.PackageNamePattern)
	r.PackageType = types.StringValue(rule.PackageType)

	r.MinimumAccessLevelForDelete = types.StringNull()
	if rule.MinimumAccessLevelForDelete != "" {
		r.MinimumAccessLevelForDelete = types.StringValue(rule.MinimumAccessLevelForDelete)
	}

	r.MinimumAccessLevelForPush = types.StringNull()
	if rule.MinimumAccessLevelForPush != "" {
		r.MinimumAccessLevelForPush = types.StringValue(rule.MinimumAccessLevelForPush)
	}
}

func (r *gitlabPackageProtectionResourceRule) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gitlabPackageProtectionModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.Project.ValueString()

	options := &gitlab.CreatePackageProtectionRulesOptions{
		PackageNamePattern: new(data.PackageNamePattern.ValueString()),
		PackageType:        new(data.PackageType.ValueString()),
	}

	// On creation, we can skip the null check because nothing exists yet.
	if !data.MinimumAccessLevelForPush.IsNull() && !data.MinimumAccessLevelForPush.IsUnknown() {
		options.MinimumAccessLevelForPush = gitlab.NewNullableWithValue(gitlab.ProtectionRuleAccessLevel(data.MinimumAccessLevelForPush.ValueString()))
	}
	if !data.MinimumAccessLevelForDelete.IsNull() && !data.MinimumAccessLevelForDelete.IsUnknown() {
		options.MinimumAccessLevelForDelete = gitlab.NewNullableWithValue(gitlab.ProtectionRuleAccessLevel(data.MinimumAccessLevelForDelete.ValueString()))
	}

	rule, _, err := r.client.ProtectedPackages.CreatePackageProtectionRules(projectID, options, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to create package protection rule: %s", err.Error()))
		return
	}

	data.packageProtectionRuleModelToState(rule, projectID)
	data.ID = types.StringValue(utils.BuildTwoPartID(new(projectID), new(strconv.Itoa(int(data.PackageProtectionRuleID.ValueInt64())))))

	// Log the creation of the resource
	tflog.Debug(ctx, "created a package protection rule", map[string]any{
		"project":              data.Project.ValueString(),
		"package_name_pattern": data.PackageNamePattern.ValueString(),
		"package_type":         data.PackageType.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPackageProtectionResourceRule) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gitlabPackageProtectionModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, rawRuleID, err := utils.ParseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format in Read. It should be '<project>:<package_protection_rule_id>'. Error: %s", data.ID, err.Error()),
		)
		return
	}

	ruleID, err := strconv.ParseInt(rawRuleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid package protection rule ID provided, package protection rule ID should be an Int",
			fmt.Sprintf("Unable to convert package protection rule ID to int: %s", err.Error()),
		)
		return
	}

	rules, _, err := r.client.ProtectedPackages.ListPackageProtectionRules(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, "project does not exist, removing from state", map[string]any{
				"project": data.Project,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read package protection rules: %s", err.Error()))
		return
	}

	found, rule := r.containsID(rules, ruleID)

	if found {
		data.packageProtectionRuleModelToState(rule, projectID)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	} else {
		tflog.Debug(ctx, "package protection rule does not exist, removing from state", map[string]any{
			"project": data.Project,
			"rule_id": ruleID,
		})
		resp.State.RemoveResource(ctx)
	}
}

func (r *gitlabPackageProtectionResourceRule) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gitlabPackageProtectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.UpdatePackageProtectionRulesOptions{
		PackageNamePattern: new(data.PackageNamePattern.ValueString()),
		PackageType:        new(data.PackageType.ValueString()),
	}

	if !data.MinimumAccessLevelForPush.IsUnknown() {
		if data.MinimumAccessLevelForPush.IsNull() {
			options.MinimumAccessLevelForPush = gitlab.NewNullNullable[gitlab.ProtectionRuleAccessLevel]()
		} else {
			options.MinimumAccessLevelForPush = gitlab.NewNullableWithValue(gitlab.ProtectionRuleAccessLevel(data.MinimumAccessLevelForPush.ValueString()))
		}
	}
	if !data.MinimumAccessLevelForDelete.IsUnknown() {
		if data.MinimumAccessLevelForDelete.IsNull() {
			options.MinimumAccessLevelForDelete = gitlab.NewNullNullable[gitlab.ProtectionRuleAccessLevel]()
		} else {
			options.MinimumAccessLevelForDelete = gitlab.NewNullableWithValue(gitlab.ProtectionRuleAccessLevel(data.MinimumAccessLevelForDelete.ValueString()))
		}
	}

	projectID := data.Project.ValueString()
	ruleID := data.PackageProtectionRuleID.ValueInt64()
	rule, _, err := r.client.ProtectedPackages.UpdatePackageProtectionRules(projectID, ruleID, options, gitlab.WithContext(ctx))

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating package protection rule for project %s", data.Project),
			err.Error(),
		)
		return
	}

	data.packageProtectionRuleModelToState(rule, projectID)

	tflog.Debug(ctx, "updated package protection rule", map[string]interface{}{
		"project": data.Project.ValueString(), "package_name_pattern": data.PackageNamePattern.ValueString(),
	})

	// Set our plan object into state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabPackageProtectionResourceRule) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gitlabPackageProtectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectID, rawRuleID, err := utils.ParseTwoPartID(data.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID format",
			fmt.Sprintf("The resource ID '%s' has an invalid format. It should be '<project>:<package_protection_rule_id>'. Error: %s", data.ID, err.Error()),
		)
		return
	}

	ruleID, err := strconv.ParseInt(rawRuleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid package protection rule ID provided, package protection rule ID should be an Int",
			fmt.Sprintf("Unable to convert package protection rule ID to int: %s", err.Error()),
		)
		return
	}

	_, err = r.client.ProtectedPackages.DeletePackageProtectionRules(projectID, ruleID, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API Error occurred",
			fmt.Sprintf("Unable to delete package protection rule: %s", err.Error()),
		)
	}

	resp.State.RemoveResource(ctx)
}

func (r *gitlabPackageProtectionResourceRule) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *gitlabPackageProtectionResourceRule) containsID(rules []*gitlab.PackageProtectionRule, ruleID int64) (bool, *gitlab.PackageProtectionRule) {
	for _, rule := range rules {
		if rule.ID == ruleID {
			return true, rule
		}
	}
	return false, nil
}
