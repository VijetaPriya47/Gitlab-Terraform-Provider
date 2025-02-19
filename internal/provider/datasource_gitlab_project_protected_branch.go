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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitLabProjectProtectedBranchDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabProjectProtectedBranchDataSource{}
)

func init() {
	registerDataSource(NewGitLabProjectProtectedBranchDataSource)
}

// NewGitLabApplicationDataSource is a helper function to simplify the provider implementation.
func NewGitLabProjectProtectedBranchDataSource() datasource.DataSource {
	return &gitLabProjectProtectedBranchDataSource{}
}

// gitlabMetadataDataSource is the data source implementation.
type gitLabProjectProtectedBranchDataSource struct {
	client *gitlab.Client
}

// gitLabMetadataDataSourceModel describes the data source data model.
type gitLabProjectProtectedBranchDataSourceModel struct {
	ProjectId                 types.String                                      `tfsdk:"project_id"`
	Name                      types.String                                      `tfsdk:"name"`
	Id                        types.Int64                                       `tfsdk:"id"`
	PushAccessLevels          []*gitlabBranchProtectionAllowedToPushObjectModel `tfsdk:"push_access_levels"`
	MergeAccessLevels         []*gitlabBranchProtectionAllowedToObjectModel     `tfsdk:"merge_access_levels"`
	AllowForcePush            types.Bool                                        `tfsdk:"allow_force_push"`
	CodeOwnerApprovalRequired types.Bool                                        `tfsdk:"code_owner_approval_required"`
}

// Metadata returns the data source type name.
func (d *gitLabProjectProtectedBranchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_protected_branch"
}

// GetSchema defines the schema for the data source.
func (d *gitLabProjectProtectedBranchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_protected_branch`" + ` data source allows details of a protected branch to be retrieved by its name and the project it belongs to.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/protected_branches/#get-a-single-protected-branch-or-wildcard-protected-branch)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The integer or path with namespace that uniquely identifies the project.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the protected branch.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"allow_force_push": schema.BoolAttribute{
				MarkdownDescription: "Whether force push is allowed.",
				Computed:            true,
			},
			"code_owner_approval_required": schema.BoolAttribute{
				MarkdownDescription: "Reject code pushes that change files listed in the CODEOWNERS file.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"push_access_levels":  schemaAllowedToPushBlock(api.ValidProtectedBranchTagAccessLevelNames),
			"merge_access_levels": schemaAllowedToBlock("merge", api.ValidProtectedBranchTagAccessLevelNames),
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabProjectProtectedBranchDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabProjectProtectedBranchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabProjectProtectedBranchDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// call protected repository branch, read Gitlab API
	protectedBranch, _, err := d.client.ProtectedBranches.GetProtectedBranch(state.ProjectId.ValueString(), state.Name.ValueString(), gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read protected branch details: %s", err.Error()))
		return
	}

	state.Id = types.Int64Value(int64(protectedBranch.ID))
	state.AllowForcePush = types.BoolValue(protectedBranch.AllowForcePush)
	state.CodeOwnerApprovalRequired = types.BoolValue(protectedBranch.CodeOwnerApprovalRequired)
	state.PushAccessLevels = populateAllowedToPushObjectList(protectedBranch.PushAccessLevels)
	state.MergeAccessLevels = populateAllowedToObjectList(protectedBranch.MergeAccessLevels)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
