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
	_ datasource.DataSource              = &gitLabProjectProtectedTagDataSource{}
	_ datasource.DataSourceWithConfigure = &gitLabProjectProtectedTagDataSource{}
)

func init() {
	registerDataSource(NewGitLabProjectProtectedTagDataSource)
}

// NewGitLabProjectProtectedTagDataSource is a helper function to simplify the provider implementation.
func NewGitLabProjectProtectedTagDataSource() datasource.DataSource {
	return &gitLabProjectProtectedTagDataSource{}
}

// gitLabProjectProtectedTagDataSource is the data source implementation.
type gitLabProjectProtectedTagDataSource struct {
	client *gitlab.Client
}

// gitLabProjectProtectedTagDataSourceModel describes the data source data model.
type gitLabProjectProtectedTagDataSourceModel struct {
	Id                 types.String                                                        `tfsdk:"id"`
	Project            types.String                                                        `tfsdk:"project"`
	Tag                types.String                                                        `tfsdk:"tag"`
	CreateAccessLevels []*gitLabProjectProtectedTagDataSourceCreateAccessLevelsObjectModel `tfsdk:"create_access_levels"`
}

// gitLabProjectProtectedTagDataSourceCreateAccessLevelsObjectModel describes the allowed to block data model.
type gitLabProjectProtectedTagDataSourceCreateAccessLevelsObjectModel struct {
	Id                     types.Int64  `tfsdk:"id"`
	AccessLevel            types.String `tfsdk:"access_level"`
	AccessLevelDescription types.String `tfsdk:"access_level_description"`
	UserId                 types.Int64  `tfsdk:"user_id"`
	GroupId                types.Int64  `tfsdk:"group_id"`
}

// Metadata returns the data source type name.
func (d *gitLabProjectProtectedTagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_protected_tag"
}

// GetSchema defines the schema for the data source.
func (d *gitLabProjectProtectedTagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_protected_tag`" + ` data source allows details of a protected tag to be retrieved by its name and the project it belongs to.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/protected_tags.html#get-a-single-protected-tag-or-wildcard-protected-tag)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource. In the format of `<tag>`.",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The integer or path with namespace that uniquely identifies the project.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"tag": schema.StringAttribute{
				MarkdownDescription: "The name of the protected tag.",
				Required:            true,
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
							Description: "Readable description of access level.",
							Computed:    true,
						},
						"user_id": schema.Int64Attribute{
							Description: "The ID of a GitLab user allowed to perform the relevant action.",
							Optional:    true,
						},
						"group_id": schema.Int64Attribute{
							Description: "The ID of a GitLab group allowed to perform the relevant action.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitLabProjectProtectedTagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitLabProjectProtectedTagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitLabProjectProtectedTagDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// call protected repository tag, read Gitlab API
	protectedTag, _, err := d.client.ProtectedTags.GetProtectedTag(state.Project.ValueString(), state.Tag.ValueString(), gitlab.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("GitLab API error occured", fmt.Sprintf("Unable to read protected tag details: %s", err.Error()))
		return
	}

	state.Id = types.StringValue(protectedTag.Name)
	state.CreateAccessLevels = populateTagCreateAccessLevelsObjectList(protectedTag.CreateAccessLevels)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func populateTagCreateAccessLevelsObjectList(access_levels []*gitlab.TagAccessDescription) []*gitLabProjectProtectedTagDataSourceCreateAccessLevelsObjectModel {
	allowedTosData := make([]*gitLabProjectProtectedTagDataSourceCreateAccessLevelsObjectModel, len(access_levels))
	for i, v := range access_levels {
		allowedToData := gitLabProjectProtectedTagDataSourceCreateAccessLevelsObjectModel{
			Id:                     types.Int64Value(int64(v.ID)),
			AccessLevelDescription: types.StringValue(v.AccessLevelDescription),
		}
		allowedToData.AccessLevel = types.StringValue(api.AccessLevelValueToName[v.AccessLevel])
		if v.UserID != 0 {
			allowedToData.UserId = types.Int64Value(int64(v.UserID))
		}
		if v.GroupID != 0 {
			allowedToData.GroupId = types.Int64Value(int64(v.GroupID))
		}
		allowedTosData[i] = &allowedToData
	}

	return allowedTosData

}
