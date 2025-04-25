package sdk

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_project_cluster", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_project_cluster` + "`" + ` resource allows to manage the lifecycle of a project cluster.

~> This is deprecated GitLab feature since 14.5

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/project_clusters/)`,

		CreateContext: resourceGitlabProjectClusterCreate,
		ReadContext:   resourceGitlabProjectClusterRead,
		UpdateContext: resourceGitlabProjectClusterUpdate,
		DeleteContext: resourceGitlabProjectClusterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"project": {
				Description: "The id of the project to add the cluster to.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Description: "The name of cluster.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"domain": {
				Description: "The base domain of the cluster.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"enabled": {
				Description: "Determines if cluster is active or not. Defaults to `true`. This attribute cannot be read.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
			},
			"managed": {
				Description: "Determines if cluster is managed by gitlab or not. Defaults to `true`. This attribute cannot be read.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
			},
			"created_at": {
				Description: "Create time.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"provider_type": {
				Description: "Provider type.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"platform_type": {
				Description: "Platform type.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"environment_scope": {
				Description: "The associated environment to the cluster. Defaults to `*`.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "*",
			},
			"cluster_type": {
				Description: "Cluster type.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"kubernetes_api_url": {
				Description: "The URL to access the Kubernetes API.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"kubernetes_token": {
				Description: "The token to authenticate against Kubernetes.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"kubernetes_ca_cert": {
				Description: "TLS certificate (needed if API is using a self-signed TLS certificate).",
				Type:        schema.TypeString,
				Optional:    true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"kubernetes_namespace": {
				Description: "The unique namespace related to the project.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"kubernetes_authorization_type": {
				Description:  "The cluster authorization type. Valid values are `rbac`, `abac`, `unknown_authorization`. Defaults to `rbac`.",
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "rbac",
				ValidateFunc: validation.StringInSlice([]string{"rbac", "abac", "unknown_authorization"}, false),
			},
			"management_project_id": {
				Description: "The ID of the management project for the cluster.",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}
})

func resourceGitlabProjectClusterCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)

	pk := gitlab.AddPlatformKubernetesOptions{
		APIURL: gitlab.Ptr(d.Get("kubernetes_api_url").(string)),
		Token:  gitlab.Ptr(d.Get("kubernetes_token").(string)),
	}

	if v, ok := d.GetOk("kubernetes_ca_cert"); ok {
		pk.CaCert = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("kubernetes_namespace"); ok {
		pk.Namespace = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("kubernetes_authorization_type"); ok {
		pk.AuthorizationType = gitlab.Ptr(v.(string))
	}

	options := &gitlab.AddClusterOptions{
		Name:               gitlab.Ptr(d.Get("name").(string)),
		Enabled:            gitlab.Ptr(d.Get("enabled").(bool)),
		Managed:            gitlab.Ptr(d.Get("managed").(bool)),
		PlatformKubernetes: &pk,
	}

	if v, ok := d.GetOk("domain"); ok {
		options.Domain = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("environment_scope"); ok {
		options.EnvironmentScope = gitlab.Ptr(v.(string))
	}

	if v, ok := d.GetOk("management_project_id"); ok {
		options.ManagementProjectID = gitlab.Ptr(v.(string))
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] create gitlab project cluster %q/%q", project, *options.Name))

	cluster, _, err := client.ProjectCluster.AddCluster(project, options, gitlab.WithContext(ctx)) //nolint:staticcheck

	if err != nil {
		return diag.FromErr(err)
	}

	clusterIdString := fmt.Sprintf("%d", cluster.ID)
	d.SetId(utils.BuildTwoPartID(&project, &clusterIdString))

	return resourceGitlabProjectClusterRead(ctx, d, meta)
}

func resourceGitlabProjectClusterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project, clusterId, err := projectIdAndClusterIdFromId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] read gitlab project cluster %q/%d", project, clusterId))

	cluster, _, err := client.ProjectCluster.GetCluster(project, clusterId, gitlab.WithContext(ctx)) //nolint:staticcheck
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, fmt.Sprintf("[DEBUG] gitlab project cluster not found %s/%d", project, clusterId))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("project", project)
	d.Set("name", cluster.Name)
	d.Set("domain", cluster.Domain)
	d.Set("created_at", cluster.CreatedAt.String())
	d.Set("provider_type", cluster.ProviderType)
	d.Set("platform_type", cluster.PlatformType)
	d.Set("environment_scope", cluster.EnvironmentScope)
	d.Set("cluster_type", cluster.ClusterType)

	d.Set("kubernetes_api_url", cluster.PlatformKubernetes.APIURL)
	d.Set("kubernetes_ca_cert", cluster.PlatformKubernetes.CaCert)
	d.Set("kubernetes_namespace", cluster.PlatformKubernetes.Namespace)
	d.Set("kubernetes_authorization_type", cluster.PlatformKubernetes.AuthorizationType)

	if cluster.ManagementProject == nil {
		d.Set("management_project_id", "")
	} else {
		d.Set("management_project_id", strconv.Itoa(cluster.ManagementProject.ID))
	}

	return nil
}

func resourceGitlabProjectClusterUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project, clusterId, err := projectIdAndClusterIdFromId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.EditClusterOptions{}

	if d.HasChange("name") {
		options.Name = gitlab.Ptr(d.Get("name").(string))
	}

	if d.HasChange("domain") {
		options.Domain = gitlab.Ptr(d.Get("domain").(string))
	}

	if d.HasChange("environment_scope") {
		options.EnvironmentScope = gitlab.Ptr(d.Get("environment_scope").(string))
	}

	pk := &gitlab.EditPlatformKubernetesOptions{}

	if d.HasChange("kubernetes_api_url") {
		pk.APIURL = gitlab.Ptr(d.Get("kubernetes_api_url").(string))
	}

	if d.HasChange("kubernetes_token") {
		pk.Token = gitlab.Ptr(d.Get("kubernetes_token").(string))
	}

	if d.HasChange("kubernetes_ca_cert") {
		pk.CaCert = gitlab.Ptr(d.Get("kubernetes_ca_cert").(string))
	}

	if d.HasChange("kubernetes_namespace") {
		pk.Namespace = gitlab.Ptr(d.Get("kubernetes_namespace").(string))
	}

	if *pk != (gitlab.EditPlatformKubernetesOptions{}) {
		options.PlatformKubernetes = pk
	}

	if d.HasChange("management_project_id") {
		options.ManagementProjectID = gitlab.Ptr(d.Get("management_project_id").(string))
	}

	if *options != (gitlab.EditClusterOptions{}) {
		tflog.Debug(ctx, fmt.Sprintf("[DEBUG] update gitlab project cluster %q/%d", project, clusterId))
		_, _, err := client.ProjectCluster.EditCluster(project, clusterId, options, gitlab.WithContext(ctx)) //nolint:staticcheck
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceGitlabProjectClusterRead(ctx, d, meta)
}

func resourceGitlabProjectClusterDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project, clusterId, err := projectIdAndClusterIdFromId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] delete gitlab project cluster %q/%d", project, clusterId))

	_, err = client.ProjectCluster.DeleteCluster(project, clusterId, gitlab.WithContext(ctx)) //nolint:staticcheck
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func projectIdAndClusterIdFromId(id string) (string, int, error) {
	project, clusterIdString, err := utils.ParseTwoPartID(id)
	if err != nil {
		return "", 0, err
	}

	clusterId, err := strconv.Atoi(clusterIdString)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get clusterId: %v", err)
	}

	return project, clusterId, nil
}
