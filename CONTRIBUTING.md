# Contributing to GitLab Terraform Provider

Thank you for contributing to this provider! :tada: :heart: :trophy:

Generally we accept any change that adds or changes a Terraform resource that is in line with the [GitLab API](https://docs.gitlab.com/ee/api/api_resources.html).
It is always best to [open an issue](https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/new) before starting on a change.

## Getting Started

Use HashiCorp's [Plugin Development](https://www.terraform.io/plugin) guide as a reference, especially the [Provider Design Principles](https://www.terraform.io/plugin/hashicorp-provider-design-principles).

See the [Developing The Provider](#developing-the-provider) section below for specifics about this GitLab provider.

The GitLab Terraform Provider is implemented partly with the [Terraform Plugin SDK](https://developer.hashicorp.com/terraform/plugin/sdkv2)
and partly with the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).
The Terraform Plugin Framework is the new and shiny thing and is what we use to implement
all new resources and data sources. We consider the SDK in this provider as _legacy_.

The new resources and data sources are located in `internal/provider` and the _legacy_ ones
in `internal/provider/sdk`.

> If you're just getting started, and looking to create a new resource, check out the ["Creating A New Resource"
tutorial available here](docs/development/CreatingANewResource.md)!

## Before Committing

Run `make reviewable` and include any generated files in your commit. This will help ensure your PR passes automated checks.

### PR Checklist

<!--
These are the most common issues in PRs which we have not automated.
-->

For a smooth review process, please run through this checklist before submitting a PR.

1. Resource attributes match 1:1 the names and structure of the API resource in the [GitLab API documentation](https://docs.gitlab.com/ee/api/api_resources.html).
1. [Examples](/examples) are updated with:
   1. A \*.tf file for the resource/s with at least one usage example
   1. A \*.sh file for the resource/s with an import example (if applicable)
1. New resources have at minimum a basic test with three steps:
   1. Create the resource
   1. Update the attributes
   1. Import the resource
1. No new `//lintignore` comments that came from copied code. Linter rules are meant to be enforced on new code.

## Guidelines

<!--
These guidelines are specific callouts for issues that come up occasionally but are somewhat niche.
-->

#### Resource ID and Import

Terraform resources have a unique ID (`d.SetId("")`). The ID should be comprised from the URL path variables of the `GET` API for the resource in the [GitLab API documentation](https://docs.gitlab.com/ee/api/api_resources.html), separated by `:`.

For example, a resource for the `GET /projects/:id/environments/:environment_id` API would have the ID `123:456` where `123` is the project ID and `456` is the environment ID.

This makes it very easy to add import functionality, by using

```go
Importer: &schema.ResourceImporter{
    StateContext: schema.ImportStatePassthroughContext,
},
```

See the [importer state function docs](https://www.terraform.io/plugin/sdkv2/resources/import#importer-state-function) for more details.

#### Inline test configuration

We prefer to inline the test Terraform configuration in acceptance tests instead of returning the configuration from a dedicated function.

We make exceptions for large resource configurations and in places where there is a lot of configuration reuse.
However, because we prefer to give each configuration a new function, there is very little reuse at present.

| Pros                                                              | Cons                                     |
| ----------------------------------------------------------------- | ---------------------------------------- |
| Easier to review when the configuration is near the related tests | Large configurations can be hard to read |

**Desired:**

```go
Steps: []resource.TestStep{
  {
    Config: fmt.Sprintf(`
      resource "gitlab_instance_variable" "test" {
        key           = "key_%[1]s"
        value         = "value-%[1]s"
        variable_type = "file"
        masked        = false
      }`,
      rString,
    ),
    Check: ... // snip
  }
}
```

**Undesired:**

```go
Steps: []resource.TestStep{
  {
    Config: testAccGitlabInstanceVariableConfig(rString),
    Check: ... // snip
  },
}

func testAccGitlabInstanceVariableConfig(rString string) string {
  return fmt.Sprintf(`
    resource "gitlab_instance_variable" "test" {
      key           = "key_%[1]s"
      value         = "value-%[1]s"
      variable_type = "file"
      masked        = false
    }`,
    rString,
  )
}
```

#### Add only resource or data source being tested to test configuration

We prefer to add only the resource or data source being tested to the acceptance test configuration instead of
adding all required resources to the test configuration.

| Pros                                                        | Cons                                                                                 |
| ----------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Allows us to use CheckDestroy correctly (see #205 (closed)) | Integration of multiple resources not tested (low risk, because of clear interfaces) |
| Less code to review                                         |                                                                                      |
| Isolates issues                                             |                                                                                      |
| Faster runtime (fewer API calls)                            |

**Desired:**

```hcl
resource "gitlab_pipeline_schedule" "test" {
  project     = %d
  ref         = "%s"
  cron        = "0 1 * * *"
}
```

The `testGitLabClient` can be used to create a test project. The [`helper_test.go`](internal/provider/helper_test.go) file
contains many functions to help create test resources. For the example above, a test project you can create the required test project with the following code:

```go
testProject := testAccCreateProject(t)

// now you can inject the project id using fmt.Sprintf and testProject.ID
```

**Undesired:**

```hcl
resource "gitlab_project" "test" {
  name = "foo-%d"
  description = "Terraform acceptance tests"
}

resource "gitlab_pipeline_schedule" "test" {
  project     = "${gitlab_project.test.id}"
  ref         = "%s"
  cron        = "0 1 * * *"
}
```

#### Use Import Verification test steps

We use `Import Verify` test steps whenever possible to check if the created resource can be imported.
The benefit of this approach is that the Terraform Acceptance Testing framework does all the validation, so we don't have to do any manual checks.

The framework does the following:

1. Re-runs `terraform plan` after each step and fails if the plan is not empty for the same config [(source)](https://www.terraform.io/plugin/sdkv2/best-practices/testing#built-in-patterns).
1. Using `ImportStateVerify`, the framework ensures that the state is the same before and after importing,
   bypassing any DiffSuppressFunc or CustomizeDiff [(source)](https://github.com/hashicorp/terraform-plugin-sdk/blob/2c03a32a9d1be63a12eb18aaf12d2c5270c42346/helper/resource/testing.go#L482). This is equivalent to checking the state and upstream resource manually.

| Pros                                                                                                                                                               | Cons                                             |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------ |
| No need to write a dedicated "CheckExists" function for each resource                                                                                              | Please [open an issue](/issues) if you find one! |
| No need to write a dedicated "CheckAttributes" function for each resource (a common sore spot for copy-paste bugs and neglecting new attributes)                   |                                                  |
| Encourages pragmatic testing using the framework's builtin check functions where necessary, especially for checking computed attributes which we routinely neglect |                                                  |
| Faster runtime (fewer API calls)                                                                                                                                   |                                                  |

**Desired**:

```go
{
  Config: `<config>`,
  // Check computed and default attributes.
  Check: resource.TestCheckResourceAttrSet("gitlab_foo.test", "bar"),
},
{
  ResourceName:      "gitlab_foo.test",
  ImportState:       true,
  ImportStateVerify: true,
},
```

**Undesired:**

```go
{
  Config: `<config>`,
  Check: resource.ComposeTestCheckFunc(
    testAccCheckGitlabFooExists("gitlab_foo.foo", &foo),
    testAccCheckGitlabFooAttributes(&foo, &testAccGitlabFooExpectedAttributes{
      Foo: "bar",
      Baz:  123,
    }),
  ),
},
```

#### Documentation

Documentation in [/docs](/docs) is auto-generated by [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) based on code descriptions and [examples](/examples). This auto-generation is run by executing the `make reviewable` command. You can use the [Terraform doc preview tool](https://registry.terraform.io/tools/doc-preview) if you would like to preview the generated documentation featuring your changes.

## Developing The Provider

### Set Up Your Local Environment

You'll first need [Go](http://www.golang.org) installed on your machine (version 1.20+ is _required_).

1. Clone the git repository.

   ```sh
   git clone git@gitlab.com:gitlab-org/terraform-provider-gitlab
   cd terraform-provider-gitlab
   ```

2. Build the provider with `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

   ```sh
   make build
   ```

To do some experiments you can use the setup in the `playground` folder.
It contains a prepared `.terraformrc` file which can be used to leverage the previously
built provider:

```sh
cd playground
export TF_CLI_CONFIG_FILE=.terraformrc
terraform plan
```

You can also install the provider to the system so that you can use from any configuration
without the `.terraformrc`.
Run one of the following `make` commands depending on what is correct for your platform:

```sh
# Default: suitable for macOS
make local

# Set the terraform platform directory
# e.g. for linux amd64:
make local TERRAFORM_PLATFORM_DIR=linux_amd64

# Set the terraform plugin dir, see https://www.terraform.io/cli/config/config-file#implied-local-mirror-directories
# e.g. for Windows
make local TERRAFORM_PLUGIN_DIR=%APPDATA%/terraform.d/plugins
```

The you can use it in your provider config like this:

```hcl
terraform {
  required_providers {
    gitlab = {
      version = "99.99.99"
      source  = "gitlab.local/x/gitlab"
    }
  }
}
```

### M1/M2 Environments

The apple silicon environments currently have an issue where the local GitLab container will not start to run tests. We recommend that you use GitPod for developing the
provider until this issue is resolved. If you manage to get a mac environment working properly, please let us know by creating an issue!

### Use a Remote Environment via GitPod

You can choose to use your own development environment if desired, however a `.gitpod.yml` file is included within the repository to allow the use of [GitPod](https://gitpod.io/) easily.
This will allow you to use GitPod's integration with GitLab to quickly start a web-based development environment including Go and Docker, which are necessary
for running tests. To use GitPod's integration, you have two different options described below. After you've completed one of the two options, your development environment
will be ready within a minute or two. As part of starting up, your development environment will automatically start up the `gitlab-ce` container necessary for running
tests, as described in the "Running Tests/Option 1" section below.

#### Option 1: Manually navigate to GitPod

You can manually sign in and open your workspace within GitPod by following these steps:

1. Navigate to [GitPod](https://gitpod.io/)
1. Click [Login](https://gitpod.io/login/) if you have an account, or [Sign Up](https://www.gitpod.io/#get-started) if you do not.
1. Click on "Continue with GitLab" and authorize GitPod to access your account.
1. After you've signed in, select "Projects" along the top menu, click on your forked `terraform-provider-gitlab` project
1. Hover over either the main branch for your fork or the branch you created for your fork, and click "New Workspace"

#### Option 2: Open your GitPod Workspace directly via URL

1. Navigate to your fork of the `terraform-provider-gitlab` project in GitLab
1. Select the branch you want to develop
1. Add `https://gitpod.io/#` to the front of your URL

Your workspace will automatically open the repository and branch that you selected in GitLab.

#### Option 3: Use the GitLab integration for GitPod

1. [Enable](https://docs.gitlab.com/ee/integration/gitpod.html#enable-gitpod-in-your-user-settings) the GitPod integration in your user settings.
1. [Launch](https://docs.gitlab.com/ee/integration/gitpod.html#launch-gitpod-in-gitlab) GitPod from the `terraform-provider-gitlab` fork you want to use.

### Running Tests

The acceptance tests can run against a Gitlab instance where you have a token with administrator permissions (likely not gitlab.com).

The GitLab CI pipeline test against the three latest GitLab releases.

#### Option 1: Run tests against a local Gitlab container with docker-compose

This option is the easiest and requires [docker-compose](https://docs.docker.com/compose/install/) (version 1.13+) to be installed on your machine.

Note that you need an up-to-date version of GNU make and an up-to-date version of openssl / libressl.

1. Start the Gitlab container. It will take about 5 minutes for the container to become healthy.

```sh
make testacc-up
```

2. Run the acceptance tests. The full suite takes 10-20 minutes to run.

```sh
make testacc
```

3. Stop the Gitlab container.

```sh
make testacc-down
```

> :memo: Compose v2 and podman-compose are also supported by setting the CONTAINER_ENGINE and CONTAINER_COMPOSE_ENGINE env variables:
>
> `CONTAINER_COMPOSE_ENGINE='docker compose' make testacc-up`
> 
> `CONTAINER_COMPOSE_ENGINE=podman-compose CONTAINER_ENGINE=podman make testacc-up`

#### Option 2: Run tests against your own Gitlab instance

If you have your own hosted Gitlab instance, you can run the tests against it directly.

```sh
make testacc GITLAB_TOKEN=example123 GITLAB_BASE_URL=https://example.com/api/v4
```

`GITLAB_TOKEN` must be a valid token for an account with admin privileges.

#### Testing Tips

- **Gitlab Community Edition and Gitlab Enterprise Edition:**

  This module supports both Gitlab CE and Gitlab EE. We run tests on Gitlab EE,
  but can't run them on pull requests from forks.

  Features that only work on one flavour can use the following helpers as
  SkipFunc: `isRunningInEE` and `isRunningInCE`. You can see an example of this
  for [gitlab_project_level_mr_approvals](internal/provider/resource_gitlab_project_level_mr_approvals_test.go)
  tests.

- **Run EE tests:**

  If you have a `Gitlab-license.txt` you can run Gitlab EE, which will enable the full suite of tests:

  ```sh
  make testacc-up SERVICE=gitlab-ee
  ```

- **Run tests against specific GitLab version:**

  Specify the GitLab release in the `GITLAB_CE_VERSION` or `GITLAB_EE_VERSION`, e.g.:

  ```sh
  make testacc-up GITLAB_CE_VERSION=15.0.0-ce.0
  ```

- **Run a single test:**

  You can pass a pattern to the `RUN` variable to run a reduced number of tests. For example:

  ```sh
  make testacc RUN=TestAccGitlabGroup
  ```

  ...will run all tests for the `gitlab_group` resource.

- **Debug a test in an IDE:**

  First start the Gitlab container with `make testacc-up`.
  Then run the desired Go test as you would normally from your IDE, but configure your run configuration to set these environment variables:

  ```
  GITLAB_TOKEN=glpat-ACCTEST1234567890123
  GITLAB_BASE_URL=http://127.0.0.1:8085/api/v4
  TF_ACC=1
  ```

- **Useful HashiCorp documentation:**

  Refer to [HashiCorp's testing guide](https://www.terraform.io/docs/extend/testing/index.html)
  and [HashiCorp's testing best practices](https://www.terraform.io/docs/extend/best-practices/testing.html).

## Release Workflow

After the migration of the GitLab Terraform Provider to GitLab,
[we've decided](https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/1331) to
use the same release cadence as [GitLab](https://about.gitlab.com/releases/).

Which means that:

- Every 22nd of the month a new minor (`X.Y+1`) release is published.
- Security and Bug Fix releases (`X.Y.Z+1`) will be publish on demand.
- Once a year on 22nd of May a new major (`X+1`) release is published.

Note, that the compatibility between a provider release and GitLab itself **cannot** be inferred from the
release version. That is, a release `15.7` of the provider might be compatible with `15.4`, `15.5`, `15.6`
and even future GitLab releases.

This workflow has been introduced with the GitLab %15.7 Milestone in December 2022.

### Release Scoping

The actions in the bullet points below allow us to plan the scope of releases and track changes over time.

- Issues and Merge Requests are assigned with a Milestone that they were implemented in.
- Every [Release](https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/releases) has a Milestone assigned.

## Release Setup

This chapter describes the setup we have in place to successfully release a Terraform Provider release built on GitLab.com to the official [Terraform Provider Registry](https://registry.terraform.io/).

### Rational

The "normal" process to release and publish a Terraform Provider to the Terraform Registry is documented by HashiCorp [here](https://developer.hashicorp.com/terraform/tutorials/providers/provider-release-publish).
The problem we are facing is that publishing to the Terraform Registry is _only_ possible from GitHub, because the Terraform Registry fetches the release assets from a GitHub release and partly from the source code (the documentation).

The GitLab Terraform Provider is naturally hosted on GitLab.com, thus we are not able to use the "normal" process from HashiCorp, but a little additional workaround.

### Setup

As outlined in the [Rational](#Retional) only GitHub is supported to publish to the Terraform Registry.
Given that limitation, we still use the old GitHub repository at <https://github.com/gitlabhq/terraform-provider-gitlab> as a proxy to release the provider.
The setup looks like this:

<div class="center">

```mermaid
sequenceDiagram
    GitLab->>GitHub: push release and docs
    GitHub->>Registry: webhook to notify about new release
    Registry-->>GitHub: fetch release and docs and make available
```

</div>

#### GitLab repository

The GitLab repository is a complete Terraform Provider source code repository, including the provider code itself,
acceptance tests and a GitLab pipeline configuration to lint, build, test and release the provider.
During a provider release not only a release in the GitLab project is created, but it's also pushed via REST API to GitHub.
In addition, the documentation is committed and pushed to the GitHub repository.
The GitLab pipeline is authenticated using the `GITHUB_TOKEN_FOR_SYNC` environment variable.
It contains a [fine-grainted personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#creating-a-fine-grained-personal-access-token)
which only has `Contents` `read-write` permissions to the GitHub repository.

#### GitHub repository

The GitHub repository only contains a [`README.md`](https://gitlab.com/gitlab-org/terraform-provider-gitlab/blob/main/README.md)
file with an information that the actual provider is located on GitLab, a GitHub Action workflow to lock down the repository,
and the committed release documentation. As described in the [GitLab repository](#gitlab-repository) section the GitHub repository
also contains proxy releases and has a webhook installed to notify the Terraform Registry about new releases.

##### GitHub repository lock down

The GitHub repository is locked down which means that generally every option to contribute to it is disabled.
Unfortunately, Pull Requests cannot be disabled so that we have to use the [`repo-lockdown`](https://github.com/marketplace/actions/repo-lockdown) action.
It'll auto-close every PR with a comment that we only accept contributions in our GitLab.com repository.

## GitLab Ultimate License for testing

The GitLab pipeline tests against an ephemeral GitLab Enterprise omnibus instance 
which requires an Ultimate license.
This license must be renewed every three months using an Access Request.

### Renew process

This process can only be followed by GitLab Team Members.

1. Create a new related issue from this AR: https://gitlab.com/gitlab-com/team-member-epics/access-requests/-/issues/18758 (internal).
1. Choose the `GitLab_Team_Member_License_request` template.
1. Make sure to check `Ultimate` with a seats of at least `100`.
1. Follow the template instructions.
1. Update the license that is sent to you in the `GITLAB_LICENSE` CI/CD variable of this project.