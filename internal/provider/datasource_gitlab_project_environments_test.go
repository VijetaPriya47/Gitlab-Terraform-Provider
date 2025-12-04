//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataProjectEnvironment_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	optsCreateEnvironmentOptions := gitlab.CreateEnvironmentOptions{
		Name:        gitlab.Ptr(acctest.RandString(10)),
		Description: gitlab.Ptr("Very best environment"),
		ExternalURL: gitlab.Ptr("example.com/env"),
		Tier:        gitlab.Ptr("other"),
	}
	environment := testutil.CreateProjectEnvironment(t, project.ID, &optsCreateEnvironmentOptions)

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_environments" "this" {
						project = %d
						states = "available"
					}
					`,
					project.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// check resource attributes
					resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "states", "available"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_environments.this", "environments.0.%"),
					// check environment attributes
					resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "environments.0.id", strconv.FormatInt(environment.ID, 10)),
					resource.TestCheckNoResourceAttr("data.gitlab_project_environments.this", "environments.0.cluster_agent_id"),
					resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "environments.0.name", environment.Name),
					resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "environments.0.description", environment.Description),
					resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "environments.0.external_url", environment.ExternalURL),
					resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "environments.0.tier", environment.Tier),
				),
			},
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_environments" "this" {
						project = "%s"
						states = "available"
					}
					`,
					project.PathWithNamespace,
				),
				Check: resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "project", project.PathWithNamespace),
			},
		},
	})
}

func TestAccDataProjectEnvironment_filter(t *testing.T) {
	project := testutil.CreateProject(t)
	optsCreateEnvironmentOptionsDev := gitlab.CreateEnvironmentOptions{
		Name:        gitlab.Ptr(acctest.RandString(10)),
		Description: gitlab.Ptr("Very best dev environment"),
		ExternalURL: gitlab.Ptr("example.com/dev"),
		Tier:        gitlab.Ptr("development"),
	}
	environmentDev := testutil.CreateProjectEnvironment(t, project.ID, &optsCreateEnvironmentOptionsDev)
	optsCreateEnvironmentOptionsProd := gitlab.CreateEnvironmentOptions{
		Name:        gitlab.Ptr(acctest.RandString(10)),
		Description: gitlab.Ptr("Very best prod environment"),
		ExternalURL: gitlab.Ptr("example.com/prod"),
		Tier:        gitlab.Ptr("production"),
	}
	environmentProd := testutil.CreateProjectEnvironment(t, project.ID, &optsCreateEnvironmentOptionsProd)

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_environments" "this" {
						project = "%d"
						name = "%s"
					}
					`,
					project.ID,
					environmentDev.Name,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.0.id",
						strconv.FormatInt(environmentDev.ID, 10),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.0.name",
						environmentDev.Name,
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.1",
					),
				),
			},
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_environments" "this" {
						project = "%d"
						search = "%s"
					}
					`,
					project.ID,
					environmentProd.Name[:5],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.0.id",
						strconv.FormatInt(environmentProd.ID, 10),
					),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.1",
					),
				),
			},
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_environments" "this" {
						project = "%d"
					}
					`,
					project.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.gitlab_project_environments.this",
						"environments.0.%"),
					resource.TestCheckResourceAttrSet(
						"data.gitlab_project_environments.this",
						"environments.1.%"),
					resource.TestCheckNoResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.2",
					),
				),
			},
		},
	})
}

func TestAccDataProjectEnvironment_clusterAgent(t *testing.T) {
	project := testutil.CreateProject(t)
	agent := testutil.CreateClusterAgents(t, project.ID, 1)[0]
	testutil.SetupUserAccess(t, project, agent)
	optsCreateEnvironmentOptionsClusterAgent := gitlab.CreateEnvironmentOptions{
		Name:                gitlab.Ptr(acctest.RandString(10)),
		ClusterAgentID:      gitlab.Ptr(agent.ID),
		KubernetesNamespace: gitlab.Ptr("flux-system"),
		FluxResourcePath:    gitlab.Ptr("helm.toolkit.fluxcd.io/v2/namespaces/gitlab-agent/helmreleases/gitlab-agent"),
	}
	environment := testutil.CreateProjectEnvironment(t, project.ID, &optsCreateEnvironmentOptionsClusterAgent)

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_environments" "this" {
						project = "%d"
						name = "%s"
					}
					`,
					project.ID,
					environment.Name,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.0.id",
						strconv.FormatInt(environment.ID, 10),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.0.name",
						environment.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.0.cluster_agent_id",
						strconv.FormatInt(environment.ClusterAgent.ID, 10),
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.0.flux_resource_path",
						environment.FluxResourcePath,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_environments.this",
						"environments.0.kubernetes_namespace",
						environment.KubernetesNamespace,
					),
				),
			},
		},
	})
}

func TestAccDataProjectEnvironment_autoStopSetting(t *testing.T) {
	project := testutil.CreateProject(t)
	optsCreateEnvironmentOptions := gitlab.CreateEnvironmentOptions{
		Name:            gitlab.Ptr(acctest.RandString(10)),
		Description:     gitlab.Ptr("Very best environment"),
		ExternalURL:     gitlab.Ptr("example.com/env"),
		Tier:            gitlab.Ptr("other"),
		AutoStopSetting: gitlab.Ptr("always"),
	}
	environment := testutil.CreateProjectEnvironment(t, project.ID, &optsCreateEnvironmentOptions)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_environments" "this" {
					  project = "%d"
					  name = "%s"
					}
					`,
					project.ID,
					environment.Name,
				),
				Check: resource.TestCheckResourceAttr("data.gitlab_project_environments.this", "environments.0.auto_stop_setting", "always"),
			},
		},
	})
}
