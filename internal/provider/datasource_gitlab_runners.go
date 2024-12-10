package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/gitlab-org/api/client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &gitlabRunnersDataSource{}
	_ datasource.DataSourceWithConfigure = &gitlabRunnersDataSource{}
)

func init() {
	registerDataSource(NewGitLabRunnersDataSource)
}

// NewGitLabRunnersDataSource is a helper function to simplify the provider implementation.
func NewGitLabRunnersDataSource() datasource.DataSource {
	return &gitlabRunnersDataSource{}
}

// gitlabRunnersDataSource is the data source implementation.
type gitlabRunnersDataSource struct {
	client *gitlab.Client
}

// gitLabRunnersDataSourceModel describes the data source data model.
type gitLabRunnersDataSourceModel struct {
	ID      types.String     `tfsdk:"id"`
	Paused  types.Bool       `tfsdk:"paused"`
	Status  types.String     `tfsdk:"status"`
	TagList types.Set        `tfsdk:"tag_list"`
	Type    types.String     `tfsdk:"type"`
	Runners []*gitlabRunners `tfsdk:"runners"`
}

type gitlabRunners struct {
	ID          types.Int64  `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Paused      types.Bool   `tfsdk:"paused"`
	IsShared    types.Bool   `tfsdk:"is_shared"`
	RunnerType  types.String `tfsdk:"runner_type"`
	Online      types.Bool   `tfsdk:"online"`
	Status      types.String `tfsdk:"status"`
}

// Metadata returns the data source type name.
func (d *gitlabRunnersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runners"
}

// Schema defines the schema for the data source.
func (d *gitlabRunnersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	allowedStatusValues := []string{"online", "offline", "stale", "never_contacted"}
	allowedTypeValues := []string{"instance_type", "group_type", "project_type"}

	resp.Schema = schema.Schema{
		MarkdownDescription: `The ` + "`gitlab_runners`" + ` data source retrieves information about all gitlab runners.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/runners.html#list-all-runners)`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this Terraform resource.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Filters for runners with the given status. Valid Values are `online`, `offline`, `stale`, and `never_contacted`.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf(allowedStatusValues...)},
			},
			"paused": schema.BoolAttribute{
				MarkdownDescription: "Filters for runners with the given paused value",
				Optional:            true,
			},
			"tag_list": schema.SetAttribute{
				MarkdownDescription: "Filters for runners with all of the given tags",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of runner to return. Valid values are `instance_type`, `group_type` and `project_type`",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf(allowedTypeValues...)},
			},
			"runners": schema.ListNestedAttribute{
				MarkdownDescription: "The list of runners.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The runner id.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "The description of the runner.",
							Computed:            true,
						},
						"paused": schema.BoolAttribute{
							MarkdownDescription: "Indicates if the runner is accepting or ignoring new jobs.",
							Computed:            true,
						},
						"is_shared": schema.BoolAttribute{
							MarkdownDescription: "Indicates if this is a shared runner",
							Computed:            true,
						},
						"runner_type": schema.StringAttribute{
							MarkdownDescription: "The runner type. Values are `instance_type`, `group_type` and `project_type`.",
							Computed:            true,
						},
						"online": schema.BoolAttribute{
							MarkdownDescription: "The connectivity status of the runner.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The status of the runner. Values can be `online`, `offline`, `stale`, and `never_contacted`.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *gitlabRunnersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	datasource := req.ProviderData.(*GitLabDatasourceData)
	d.client = datasource.Client
}

// Read refreshes the Terraform state with the latest data.
func (d *gitlabRunnersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitLabRunnersDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	options := &gitlab.ListRunnersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	if !config.Paused.IsNull() && !config.Paused.IsUnknown() {
		options.Paused = config.Paused.ValueBoolPointer()
	}

	if !config.Status.IsNull() && !config.Status.IsUnknown() {
		options.Status = config.Status.ValueStringPointer()
	}

	if !config.TagList.IsNull() && !config.TagList.IsUnknown() {
		var tags []string
		config.TagList.ElementsAs(ctx, &tags, true)
		options.TagList = &tags
	}

	if !config.Type.IsNull() && !config.Type.IsUnknown() {
		options.Type = config.Type.ValueStringPointer()
	}

	runners, err := d.getAllRunners(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError(
			"GitLab API error occurred",
			err.Error(),
		)
		return
	}

	for _, runner := range runners {
		config.Runners = append(config.Runners, &gitlabRunners{
			ID:          types.Int64Value(int64(runner.ID)),
			Description: types.StringValue(runner.Description),
			Paused:      types.BoolValue(runner.Paused),
			IsShared:    types.BoolValue(runner.IsShared),
			RunnerType:  types.StringValue(runner.RunnerType),
			Online:      types.BoolValue(runner.Online),
			Status:      types.StringValue(runner.Status),
		})
	}

	diags := resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

func (d *gitlabRunnersDataSource) getAllRunners(ctx context.Context, options *gitlab.ListRunnersOptions) ([]*gitlab.Runner, error) {
	var runners []*gitlab.Runner
	for options.Page != 0 {
		// Make API call to read gitlab runners
		r, resp, err := d.client.Runners.ListAllRunners(options, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("Unable to read runners: %s", err.Error())
		}

		runners = append(runners, r...)

		options.Page = resp.NextPage
	}
	return runners, nil
}
