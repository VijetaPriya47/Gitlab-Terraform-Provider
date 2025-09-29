package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	_ datasource.DataSource              = &gitLabProjectApprovalRulesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabProjectApprovalRulesDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectApprovalRulesDataSource)
}

// NewGitlabProjectApprovalRulesDataSource is a helper function to simplify the provider implementation.
func NewGitlabProjectApprovalRulesDataSource() datasource.DataSource {
	return &gitLabProjectApprovalRulesDataSource{}
}

// gitLabProjectApprovalRulesDataSource is the data source implementation.
type gitLabProjectApprovalRulesDataSource struct {
	client *gitlab.Client
}

// gitLabProjectApprovalRulesDataSourceModel describes the data source data model.
type gitLabProjectApprovalRulesDataSourceModel struct {
	ID            types.String                                       `tfsdk:"id"`
	Project       types.String                                       `tfsdk:"project"`
	ApprovalRules []*gitLabProjectApprovalRulesObjectDataSourceModel `tfsdk:"approval_rules"`
}

type gitLabProjectApprovalRulesObjectDataSourceModel struct {
	ID                            types.Int64   `tfsdk:"id"`
	Name                          types.String  `tfsdk:"name"`
	RuleType                      types.String  `tfsdk:"rule_type"`
	ReportType                    types.String  `tfsdk:"report_type"`
	ApprovalsRequired             types.Int64   `tfsdk:"approvals_required"`
	EligibleApproverIDs           []types.Int64 `tfsdk:"eligible_approver_ids"`
	UserIDs                       []types.Int64 `tfsdk:"user_ids"`
	GroupIDs                      []types.Int64 `tfsdk:"group_ids"`
	ProtectedBranchIDs            []types.Int64 `tfsdk:"protected_branch_ids"`
	AppliesToAllProtectedBranches types.Bool    `tfsdk:"applies_to_all_protected_branches"`
}

// Metadata returns the data source type name.
func (d *gitLabProjectApprovalRulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_approval_rules"
}

// GetSchema defines the schema for the data source.
func (d *gitLabProjectApprovalRulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_approval_rules`" + ` data source retrieves all approval rules of a given project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/merge_request_approvals/#list-all-approval-rules-for-a-project)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or path with namespace that identifies the project.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		Blocks: map[string]schema.Block{
			"approval_rules": schema.ListNestedBlock{
				MarkdownDescription: "A list of project approval rules, as defined below.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of the approval rule.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the approval rule.",
							Computed:            true,
						},
						"rule_type": schema.StringAttribute{
							MarkdownDescription: "The type of the approval rule. Can be `any_approver`, `regular` or `report_approver`.",
							Computed:            true,
						},
						"report_type": schema.StringAttribute{
							MarkdownDescription: "The report type. Required when the rule type is `report_approver`. The supported report types are `license_scanning` and `code_coverage`.",
							Computed:            true,
						},
						"approvals_required": schema.Int64Attribute{
							MarkdownDescription: "The number of approvals required for this rule.",
							Computed:            true,
						},
						"eligible_approver_ids": schema.ListAttribute{
							MarkdownDescription: "List of all approver IDs that are eligible to approve this rule.",
							Computed:            true,
							ElementType:         types.Int64Type,
						},
						"user_ids": schema.ListAttribute{
							MarkdownDescription: "List of user IDs that are eligible to approve this rule.",
							Computed:            true,
							ElementType:         types.Int64Type,
						},
						"group_ids": schema.ListAttribute{
							MarkdownDescription: "List of group IDs that are eligible to approve this rule.",
							Computed:            true,
							ElementType:         types.Int64Type,
						},
						"protected_branch_ids": schema.ListAttribute{
							MarkdownDescription: "List of protected branch IDs that this rule applies to.",
							Computed:            true,
							ElementType:         types.Int64Type,
						},
						"applies_to_all_protected_branches": schema.BoolAttribute{
							MarkdownDescription: "If true, applies the rule to all protected branches, ignoring the protected branches attribute.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabProjectApprovalRulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabProjectApprovalRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabProjectApprovalRulesDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := state.Project.ValueString()
	options := &gitlab.GetProjectApprovalRulesListsOptions{}

	// Make API call to read the project approval rules
	approvalRules, err := gitlab.ScanAndCollect(func(pagination gitlab.PaginationOptionFunc) ([]*gitlab.ProjectApprovalRule, *gitlab.Response, error) {
		return d.client.Projects.GetProjectApprovalRules(project, options, pagination, gitlab.WithContext(ctx))
	})
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occurred", fmt.Sprintf("Unable to read project approval rules: %s", err.Error()))
		return
	}

	state.ID = types.StringValue(project)
	state.ApprovalRules = populateApprovalRules(approvalRules)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func populateApprovalRules(rules []*gitlab.ProjectApprovalRule) []*gitLabProjectApprovalRulesObjectDataSourceModel {
	approvalRules := make([]*gitLabProjectApprovalRulesObjectDataSourceModel, len(rules))

	for i, rule := range rules {
		approvalRule := gitLabProjectApprovalRulesObjectDataSourceModel{
			ID:                            types.Int64Value(int64(rule.ID)),
			Name:                          types.StringValue(rule.Name),
			RuleType:                      types.StringValue(rule.RuleType),
			ApprovalsRequired:             types.Int64Value(int64(rule.ApprovalsRequired)),
			AppliesToAllProtectedBranches: types.BoolValue(rule.AppliesToAllProtectedBranches),
		}

		if rule.ReportType != "" {
			approvalRule.ReportType = types.StringValue(rule.ReportType)
		}

		// Get eligible approver IDs
		approverIDs := make([]types.Int64, len(rule.EligibleApprovers))
		for j, approver := range rule.EligibleApprovers {
			approverIDs[j] = types.Int64Value(int64(approver.ID))
		}
		approvalRule.EligibleApproverIDs = approverIDs

		// Get user IDs
		userIDs := make([]types.Int64, len(rule.Users))
		for j, user := range rule.Users {
			userIDs[j] = types.Int64Value(int64(user.ID))
		}
		approvalRule.UserIDs = userIDs

		// Get group IDs
		groupIDs := make([]types.Int64, len(rule.Groups))
		for j, group := range rule.Groups {
			groupIDs[j] = types.Int64Value(int64(group.ID))
		}
		approvalRule.GroupIDs = groupIDs

		// Get protected branch IDs
		protectedBranchIDs := make([]types.Int64, len(rule.ProtectedBranches))
		for j, branch := range rule.ProtectedBranches {
			protectedBranchIDs[j] = types.Int64Value(int64(branch.ID))
		}
		approvalRule.ProtectedBranchIDs = protectedBranchIDs

		approvalRules[i] = &approvalRule
	}

	return approvalRules
}
