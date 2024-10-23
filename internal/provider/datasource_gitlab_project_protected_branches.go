package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitLabProjectProtectedBranchesDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabProjectProtectedBranchesDataSource{}
)

func init() {
	registerDataSource(NewGitLabProjectProtectedBranchesDataSource)
}

// NewGitLabApplicationDataSource is a helper function to simplify the provider implementation.
func NewGitLabProjectProtectedBranchesDataSource() datasource.DataSource {
	return &gitLabProjectProtectedBranchesDataSource{}
}

// gitlabMetadataDataSource is the data source implementation.
type gitLabProjectProtectedBranchesDataSource struct {
	client *gitlab.Client
}

// gitLabMetadataDataSourceModel describes the data source data model.
type gitLabProjectProtectedBranchesDataSourceModel struct {
	Id                types.Int64                                           `tfsdk:"id"`
	ProjectId         types.String                                          `tfsdk:"project_id"`
	ProtectedBranches []gitLabProjectProtectedBranchesObjectDataSourceModel `tfsdk:"protected_branches"`
}

// gitLabMetadataDataSourceModel describes the data source data model.
type gitLabProjectProtectedBranchesObjectDataSourceModel struct {
	Name                      types.String                                      `tfsdk:"name"`
	Id                        types.Int64                                       `tfsdk:"id"`
	PushAccessLevels          []*gitlabBranchProtectionAllowedToPushObjectModel `tfsdk:"push_access_levels"`
	MergeAccessLevels         []*gitlabBranchProtectionAllowedToObjectModel     `tfsdk:"merge_access_levels"`
	AllowForcePush            types.Bool                                        `tfsdk:"allow_force_push"`
	CodeOwnerApprovalRequired types.Bool                                        `tfsdk:"code_owner_approval_required"`
}

// Metadata returns the data source type name.
func (d *gitLabProjectProtectedBranchesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_protected_branches"
}

// GetSchema defines the schema for the data source.
func (d *gitLabProjectProtectedBranchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_protected_branches`" + ` data source allows details of the protected branches of a given project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/protected_branches.html#list-protected-branches)`,

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
		},
		Blocks: map[string]schema.Block{
			"protected_branches": schema.ListNestedBlock{
				MarkdownDescription: "A list of protected branches, as defined below.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The ID of this resource.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the protected branch.",
							Computed:            true,
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabProjectProtectedBranchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*gitlab.Client)
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabProjectProtectedBranchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabProjectProtectedBranchesDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project_id := state.ProjectId.ValueString()

	// call project details retrieval Gitlab API
	projectDetails, _, err := d.client.Projects.GetProject(project_id, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read project details: %s", err.Error()))
		return
	}

	var allProtectedBranches []*gitlab.ProtectedBranch
	totalPages := -1
	opts := &gitlab.ListProtectedBranchesOptions{}
	for opts.Page = 0; opts.Page != totalPages; opts.Page++ {
		// Get protected branch by project ID/path and branch name
		protectedBranches, response, err := d.client.ProtectedBranches.ListProtectedBranches(project_id, opts, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to list protected branches for project: %s. Error message: %s",
				project_id, err.Error()))
			return
		}
		totalPages = response.TotalPages
		allProtectedBranches = append(allProtectedBranches, protectedBranches...)
	}

	state.ProjectId = types.StringValue(project_id)
	state.ProtectedBranches = populateProtectedBranches(allProtectedBranches)
	state.Id = types.Int64Value(int64((projectDetails.ID)))

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func populateProtectedBranches(pbs []*gitlab.ProtectedBranch) (values []gitLabProjectProtectedBranchesObjectDataSourceModel) {
	protectedBranches := make([]gitLabProjectProtectedBranchesObjectDataSourceModel, len(pbs))

	for i, protectedBranch := range pbs {
		pb := gitLabProjectProtectedBranchesObjectDataSourceModel{}
		pb.Id = types.Int64Value(int64(protectedBranch.ID))
		pb.Name = types.StringValue(protectedBranch.Name)
		pb.AllowForcePush = types.BoolValue(protectedBranch.AllowForcePush)
		pb.CodeOwnerApprovalRequired = types.BoolValue(protectedBranch.CodeOwnerApprovalRequired)
		pb.PushAccessLevels = populateAllowedToPushObjectList(protectedBranch.PushAccessLevels)
		pb.MergeAccessLevels = populateAllowedToObjectList(protectedBranch.MergeAccessLevels)
		protectedBranches[i] = pb
	}
	return protectedBranches
}
