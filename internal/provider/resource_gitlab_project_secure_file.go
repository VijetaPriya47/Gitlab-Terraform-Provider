package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var (
	_ resource.Resource                = &gitlabProjectSecureFileResource{}
	_ resource.ResourceWithConfigure   = &gitlabProjectSecureFileResource{}
	_ resource.ResourceWithImportState = &gitlabProjectSecureFileResource{}
)

func init() {
	registerResource(NewGitlabProjectSecureFileResource)
}

func NewGitlabProjectSecureFileResource() resource.Resource {
	return &gitlabProjectSecureFileResource{}
}

type gitlabProjectSecureFileResourceModel struct {
	ID                types.String `tfsdk:"id"`
	SecureFileID      types.Int64  `tfsdk:"secure_file_id"`
	Project           types.String `tfsdk:"project"`
	Content           types.String `tfsdk:"content"`
	Name              types.String `tfsdk:"name"`
	Checksum          types.String `tfsdk:"checksum"`
	ChecksumAlgorithm types.String `tfsdk:"checksum_algorithm"`
	CreatedAt         types.String `tfsdk:"created_at"`
	ExpiresAt         types.String `tfsdk:"expires_at"`
}

type gitlabProjectSecureFileResource struct {
	client *gitlab.Client
}

func (r *gitlabProjectSecureFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_secure_file"
}

func (r *gitlabProjectSecureFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(`The ` + "`gitlab_project_secure_file`" + ` resource allows users to manage the lifecycle of a secure file in gitlab.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/secure_files/)`),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource. In the format `<project:id>",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"secure_file_id": schema.Int64Attribute{
				MarkdownDescription: "The id of the secure file in gitlab",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The ID or full path of the project to environment is created for.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name for the secure file, unique per project",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The contents of the secure file",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
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
		},
	}
}

func (r *gitlabProjectSecureFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	resourceData := req.ProviderData.(*GitLabResourceData)
	r.client = resourceData.Client
}

func (r *gitlabProjectSecureFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *gitlabProjectSecureFileResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := data.Project.ValueString()
	contentReader := strings.NewReader(data.Content.ValueString())
	options := &gitlab.CreateSecureFileOptions{
		Name: data.Name.ValueStringPointer(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Project %s creating secure file %q", project, *options.Name))

	secureFile, _, err := r.client.SecureFiles.CreateSecureFile(project, contentReader, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitLab secure file", err.Error())
		return
	}

	// Save successful creation to state
	secureFileID := fmt.Sprintf("%d", secureFile.ID)
	data.ID = types.StringValue(utils.BuildTwoPartID(&project, &secureFileID))
	resp.Diagnostics.Append(data.modelToStateModel(ctx, project, secureFile)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Debug(ctx, fmt.Sprintf("Project %s created secure file with id %s", project, secureFileID))
}

func (r *gitlabProjectSecureFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *gitlabProjectSecureFileResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, secureFileID, err := resourceGitLabSecureFileParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing GitLab project secure file ID", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Project %s read secure file %d", project, secureFileID))

	secureFile, _, err := r.client.SecureFiles.ShowSecureFileDetails(project, secureFileID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading GitLab project secure file", err.Error())
		return
	}

	resp.Diagnostics.Append(data.modelToStateModel(ctx, project, secureFile)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gitlabProjectSecureFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No-op since all attributes require replacement
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"This resource does not support updates. All changes require replacement.",
	)
}

func (r *gitlabProjectSecureFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *gitlabProjectSecureFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, secureFileID, err := resourceGitLabSecureFileParseId(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing GitLab project secure file ID", err.Error())
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Project %s deleting secure file %s", project, data.Name))

	_, err = r.client.SecureFiles.RemoveSecureFile(project, secureFileID)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting secure file", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *gitlabProjectSecureFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *gitlabProjectSecureFileResourceModel) modelToStateModel(ctx context.Context, project string, secureFile *gitlab.SecureFile) diag.Diagnostics {
	secureFileId := fmt.Sprintf("%d", secureFile.ID)
	d.ID = types.StringValue(utils.BuildTwoPartID(&project, &secureFileId))

	d.Project = types.StringValue(project)
	d.Name = types.StringValue(secureFile.Name)
	d.SecureFileID = types.Int64Value(int64(secureFile.ID))
	d.Checksum = types.StringValue(secureFile.Checksum)
	d.ChecksumAlgorithm = types.StringValue(secureFile.ChecksumAlgorithm)
	d.CreatedAt = types.StringValue(secureFile.CreatedAt.String())

	if secureFile.ExpiresAt == nil {
		d.ExpiresAt = types.StringNull()
	} else {
		d.ExpiresAt = types.StringValue(secureFile.CreatedAt.String())
	}

	return nil
}

func resourceGitLabSecureFileParseId(id string) (string, int64, error) {
	projectID, secureFileID, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	secureFileIID, err := strconv.Atoi(secureFileID)
	if err != nil {
		return "", 0, err
	}

	return projectID, int64(secureFileIID), nil
}
