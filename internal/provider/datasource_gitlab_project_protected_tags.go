package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitLabProjectProtectedTagsDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabProjectProtectedTagsDataSource{}
)

func init() {
	registerDataSource(NewGitLabProjectProtectedTagsDataSource)
}

// NewGitLabProjectProtectedTagsDataSource is a helper function to simplify the provider implementation.
func NewGitLabProjectProtectedTagsDataSource() datasource.DataSource {
	return &gitLabProjectProtectedTagsDataSource{}
}

// gitLabProjectProtectedTagsDataSource is the data source implementation.
type gitLabProjectProtectedTagsDataSource struct {
	client *gitlab.Client
}

// gitLabProjectProtectedTagsDataSourceModel describes the data source data model.
type gitLabProjectProtectedTagsDataSourceModel struct {
	Id            types.String                                      `tfsdk:"id"`
	Project       types.String                                      `tfsdk:"project"`
	ProtectedTags []gitLabProjectProtectedTagsObjectDataSourceModel `tfsdk:"protected_tags"`
}

// gitLabProjectProtectedTagsObjectDataSourceModel describes the data source data model.
type gitLabProjectProtectedTagsObjectDataSourceModel struct {
	Tag                types.String                                                        `tfsdk:"tag"`
	CreateAccessLevels []*gitLabProjectProtectedTagDataSourceCreateAccessLevelsObjectModel `tfsdk:"create_access_levels"`
}

// Metadata returns the data source type name.
func (d *gitLabProjectProtectedTagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_protected_tags"
}

// GetSchema defines the schema for the data source.
func (d *gitLabProjectProtectedTagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_protected_tags`" + ` data source allows details of the protected tags of a given project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/protected_tags/#list-protected-tags)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The integer or path with namespace that uniquely identifies the project.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			// },
			"protected_tags": schema.ListNestedAttribute{
				MarkdownDescription: "A list of protected tags, as defined below.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"tag": schema.StringAttribute{
							MarkdownDescription: "The name of the protected tag.",
							Computed:            true,
							Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"create_access_levels": schema.ListNestedAttribute{
							MarkdownDescription: "Array of access levels/user(s)/group(s) allowed to create protected tags.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										MarkdownDescription: "The ID of the create access level.",
										Computed:            true,
									},
									"access_level": schema.StringAttribute{
										MarkdownDescription: "Access level allowed to create protected tags.",
										Computed:            true,
									},
									"access_level_description": schema.StringAttribute{
										MarkdownDescription: "Readable description of access level.",
										Computed:            true,
									},
									"user_id": schema.Int64Attribute{
										MarkdownDescription: "The ID of a GitLab user allowed to perform the relevant action.",
										Optional:            true,
									},
									"group_id": schema.Int64Attribute{
										MarkdownDescription: "The ID of a GitLab group allowed to perform the relevant action.",
										Optional:            true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabProjectProtectedTagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabProjectProtectedTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabProjectProtectedTagsDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := state.Project.ValueString()

	// call project details retrieval Gitlab API
	projectDetails, _, err := d.client.Projects.GetProject(project, nil, gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read project details: %s", err.Error()))
		return
	}

	var allProtectedTags []*gitlab.ProtectedTag
	totalPages := -1
	opts := &gitlab.ListProtectedTagsOptions{}
	for opts.Page = 0; opts.Page != totalPages; opts.Page++ {
		// Get protected tag by project ID/path and tag name
		protectedTags, response, err := d.client.ProtectedTags.ListProtectedTags(project, opts, gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to list protected tags for project: %s. Error message: %s",
				project, err.Error()))
			return
		}
		totalPages = response.TotalPages
		allProtectedTags = append(allProtectedTags, protectedTags...)
	}

	state.ProtectedTags = populateProtectedTags(allProtectedTags)
	state.Id = types.StringValue(strconv.Itoa(projectDetails.ID))

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func populateProtectedTags(pts []*gitlab.ProtectedTag) (values []gitLabProjectProtectedTagsObjectDataSourceModel) {
	protectedTags := make([]gitLabProjectProtectedTagsObjectDataSourceModel, len(pts))

	for i, protectedTag := range pts {
		pt := gitLabProjectProtectedTagsObjectDataSourceModel{}
		pt.Tag = types.StringValue(protectedTag.Name)
		pt.CreateAccessLevels = populateTagCreateAccessLevelsObjectList(protectedTag.CreateAccessLevels)

		protectedTags[i] = pt
	}
	return protectedTags
}
