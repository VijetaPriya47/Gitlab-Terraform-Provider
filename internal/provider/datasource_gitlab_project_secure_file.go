package provider

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ datasource.DataSource              = &gitlabProjectSecureFileDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabProjectSecureFileDataSource{}
)

func init() {
	registerDataSource(NewGitlabProjectSecureFileDataSource)
}

func NewGitlabProjectSecureFileDataSource() datasource.DataSource {
	return &gitlabProjectSecureFileDataSource{}
}

type gitlabProjectSecureFileDataSource struct {
	client *gitlab.Client
}

type gitlabProjectSecureFileDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	SecureFileID      types.Int64  `tfsdk:"secure_file_id"`
	Project           types.String `tfsdk:"project"`
	Content           types.String `tfsdk:"content"`
	Name              types.String `tfsdk:"name"`
	Checksum          types.String `tfsdk:"checksum"`
	ChecksumAlgorithm types.String `tfsdk:"checksum_algorithm"`
	CreatedAt         types.String `tfsdk:"created_at"`
	ExpiresAt         types.String `tfsdk:"expires_at"`
	Metadata          types.Object `tfsdk:"metadata"`
}

type gitlabProjectSecureFileDataSourceModelMetadata struct {
	ID        types.String `tfsdk:"id"`
	Issuer    types.Object `tfsdk:"issuer"`
	Subject   types.Object `tfsdk:"subject"`
	ExpiresAt types.String `tfsdk:"expires_at"`
}

func (m gitlabProjectSecureFileDataSourceModelMetadata) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":         types.StringType,
		"expires_at": types.StringType,
		"issuer":     types.ObjectType{AttrTypes: gitlabProjectSecureFileDataSourceModelIssuer{}.AttributeTypes()},
		"subject":    types.ObjectType{AttrTypes: gitlabProjectSecureFileDataSourceModelSubject{}.AttributeTypes()},
	}
}

type gitlabProjectSecureFileDataSourceModelSubject struct {
	C   types.String `tfsdk:"c"`
	O   types.String `tfsdk:"o"`
	CN  types.String `tfsdk:"cn"`
	OU  types.String `tfsdk:"ou"`
	UID types.String `tfsdk:"uid"`
}

func (m gitlabProjectSecureFileDataSourceModelSubject) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"c":   types.StringType,
		"o":   types.StringType,
		"cn":  types.StringType,
		"ou":  types.StringType,
		"uid": types.StringType,
	}
}

type gitlabProjectSecureFileDataSourceModelIssuer struct {
	C  types.String `tfsdk:"c"`
	O  types.String `tfsdk:"o"`
	CN types.String `tfsdk:"cn"`
	OU types.String `tfsdk:"ou"`
}

func (m gitlabProjectSecureFileDataSourceModelIssuer) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"c":  types.StringType,
		"o":  types.StringType,
		"cn": types.StringType,
		"ou": types.StringType,
	}
}

func (d *gitlabProjectSecureFileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_secure_file"
}

func (d *gitlabProjectSecureFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_project_secure_file`" + ` data source allows the contents of a secure file to be retrieved by either Name or ID.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/secure_files/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this data source. In the format `<project:id>`",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project the secure file resides.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"secure_file_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the secure file in gitlab",
				Computed:            true,
				Optional:            true,
				Validators:          []validator.Int64{int64validator.ExactlyOneOf(path.MatchRoot("name"))},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name for the secure file, unique per project",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(path.MatchRoot("secure_file_id")),
				},
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The contents of the secure file",
				Computed:            true,
			},
			"checksum": schema.StringAttribute{
				MarkdownDescription: "The checksum of the file",
				Computed:            true,
			},
			"checksum_algorithm": schema.StringAttribute{
				MarkdownDescription: "The checksum algorithm used",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time the secure file was uploaded",
				Computed:            true,
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "The time the secure file will expire",
				Computed:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "metadata returned by the gitlab api about the secure file",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Certificate ID",
						Computed:            true,
					},
					"expires_at": schema.StringAttribute{
						MarkdownDescription: "Certificate expiration date",
						Computed:            true,
					},
					"issuer": schema.SingleNestedAttribute{
						MarkdownDescription: "Certificate issuer information",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"c": schema.StringAttribute{
								MarkdownDescription: "Country",
								Computed:            true,
							},
							"o": schema.StringAttribute{
								MarkdownDescription: "Organization",
								Computed:            true,
							},
							"cn": schema.StringAttribute{
								MarkdownDescription: "Common Name",
								Computed:            true,
							},
							"ou": schema.StringAttribute{
								MarkdownDescription: "Organizational Unit",
								Computed:            true,
							},
						},
					},
					"subject": schema.SingleNestedAttribute{
						MarkdownDescription: "Certificate subject information",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"c": schema.StringAttribute{
								MarkdownDescription: "Country",
								Computed:            true,
							},
							"o": schema.StringAttribute{
								MarkdownDescription: "Organization",
								Computed:            true,
							},
							"cn": schema.StringAttribute{
								MarkdownDescription: "Common Name",
								Computed:            true,
							},
							"ou": schema.StringAttribute{
								MarkdownDescription: "Organizational Unit",
								Computed:            true,
							},
							"uid": schema.StringAttribute{
								MarkdownDescription: "User ID",
								Computed:            true,
							},
						},
					},
				},
			},
		},
	}
}

func (d *gitlabProjectSecureFileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

func (d *gitlabProjectSecureFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *gitlabProjectSecureFileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var secureFile *gitlab.SecureFile
	var err error
	project := data.Project.ValueString()

	switch {
	case !data.SecureFileID.IsNull() && !data.SecureFileID.IsUnknown():
		// Get secure file by id
		tflog.Info(ctx, "Reading Gitlab Secure File by ID")
		// Get details about the securefile
		secureFile, _, err = d.client.SecureFiles.ShowSecureFileDetails(project, data.SecureFileID.ValueInt64(), gitlab.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("Error fetching secure file details", err.Error())
			return
		}

		tflog.Debug(ctx, fmt.Sprintf("Fetched Secure File info: %d", secureFile.ID))
	case !data.Name.IsNull() && !data.Name.IsUnknown():
		// Get secure file by name
		tflog.Info(ctx, "Reading Gitlab Secure File by Name")
		secureFile = d.findSecureFileByName(ctx, data, &resp.Diagnostics)
	default:
		resp.Diagnostics.AddError("Missing required parameter", "One and only one of secure_file_id or name must be set")
		return
	}

	if secureFile == nil {
		resp.Diagnostics.AddError("Secure File not found", "No Secure File was found matching the specified criteria")
		return
	}

	// Download the actual securefile
	tflog.Debug(ctx, fmt.Sprintf("Downloading Secure File %d from project %s", secureFile.ID, project))
	secureFileContent, _, err := d.client.SecureFiles.DownloadSecureFile(project, secureFile.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error downloading secure file", err.Error())
		return
	}

	contentBytes, err := io.ReadAll(secureFileContent)
	if err != nil {
		resp.Diagnostics.AddError("Error reading secure file content", err.Error())
		return
	}

	data.Content = types.StringValue(string(contentBytes))
	data.Name = types.StringValue(secureFile.Name)
	data.Checksum = types.StringValue(secureFile.Checksum)
	data.ChecksumAlgorithm = types.StringValue(secureFile.ChecksumAlgorithm)
	data.CreatedAt = types.StringValue(secureFile.CreatedAt.String())

	secureFileId := fmt.Sprintf("%d", secureFile.ID)
	data.SecureFileID = types.Int64Value(int64(secureFile.ID))
	data.ID = types.StringValue(utils.BuildTwoPartID(data.Project.ValueStringPointer(), &secureFileId))

	if secureFile.Metadata == nil {
		data.Metadata = types.ObjectNull(gitlabProjectSecureFileDataSourceModelMetadata{}.AttributeTypes())
	} else {
		// Create issuer object
		issuerModel := gitlabProjectSecureFileDataSourceModelIssuer{
			C:  types.StringValue(secureFile.Metadata.Issuer.C),
			O:  types.StringValue(secureFile.Metadata.Issuer.O),
			CN: types.StringValue(secureFile.Metadata.Issuer.CN),
			OU: types.StringValue(secureFile.Metadata.Issuer.OU),
		}

		issuerObj, diags := types.ObjectValueFrom(ctx, issuerModel.AttributeTypes(), issuerModel)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		// Create subject object
		subjectModel := gitlabProjectSecureFileDataSourceModelSubject{
			C:   types.StringValue(secureFile.Metadata.Subject.C),
			O:   types.StringValue(secureFile.Metadata.Subject.O),
			CN:  types.StringValue(secureFile.Metadata.Subject.CN),
			OU:  types.StringValue(secureFile.Metadata.Subject.OU),
			UID: types.StringValue(secureFile.Metadata.Subject.UID),
		}

		subjectObj, diags := types.ObjectValueFrom(ctx, subjectModel.AttributeTypes(), subjectModel)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		// Create metadata object
		metadataModel := gitlabProjectSecureFileDataSourceModelMetadata{
			ID:      types.StringValue(secureFile.Metadata.ID),
			Issuer:  issuerObj,
			Subject: subjectObj,
		}

		if secureFile.Metadata.ExpiresAt != nil {
			metadataModel.ExpiresAt = types.StringValue(secureFile.Metadata.ExpiresAt.Format(time.RFC3339))
		} else {
			metadataModel.ExpiresAt = types.StringNull()
		}

		metadataObj, diags := types.ObjectValueFrom(ctx, metadataModel.AttributeTypes(), metadataModel)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		data.Metadata = metadataObj
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *gitlabProjectSecureFileDataSource) findSecureFileByName(ctx context.Context, data *gitlabProjectSecureFileDataSourceModel, diags *diag.Diagnostics) *gitlab.SecureFile {
	options := &gitlab.ListProjectSecureFilesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 20,
		},
	}
	project := data.Project.ValueString()
	name := data.Name.ValueString()

	for {
		secureFiles, resp, err := d.client.SecureFiles.ListProjectSecureFiles(project, options, gitlab.WithContext(ctx))
		if err != nil {
			diags.AddError("GitLab API error occurred", fmt.Sprintf("Unable to list secure files: %s", err.Error()))
			return nil
		}

		for _, secureFile := range secureFiles {
			if secureFile.Name == name {
				return secureFile
			}
		}

		if resp.NextPage == 0 {
			break
		}

		options.Page = resp.NextPage
	}

	return nil
}
