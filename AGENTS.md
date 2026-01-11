# Agent Guide for GitLab Terraform Provider

This guide is designed for AI agents working on the GitLab Terraform Provider codebase. It provides essential information about the repository structure, development workflows, testing procedures, and issue triage processes.

## Essential Reading Order

Before making any changes to this repository, read the following documents in order:

1. **[README.md](README.md)** - Start here for project overview, support policy, and general information
2. **[CONTRIBUTING.md](CONTRIBUTING.md)** - Read this for development setup, guidelines, and contribution requirements
3. **[docs/development/CreatingANewResource.md](docs/development/CreatingANewResource.md)** - Essential guide for creating new resources using the Terraform Plugin Framework

## Repository Structure

- **`internal/provider/`** - New resources and data sources using Terraform Plugin Framework
- **`internal/provider/sdk/`** - Legacy resources using Terraform Plugin SDK (maintenance only)
- **`docs/`** - Auto-generated documentation (do not edit manually)
- **`examples/`** - Example configurations for resources and data sources
- **`scripts/`** - Helper scripts for development and CI/CD
- **`GNUmakefile`** - Build and test automation

## Development Workflow

### Before Making Changes

Always run before committing:

```bash
make reviewable
```

This command runs:

- `make build` - Builds the provider
- `make fmt` - Formats code and fixes issues
- `make generate` - Generates documentation
- `make test` - Runs unit tests

**CRITICAL** Always run a least the tests for your modified resources
before you commit changes!

### Building the Provider

```bash
make build
```

The provider binary will be installed to `$GOPATH/bin`.

## Running Tests

### Unit Tests

Run all unit tests:

```bash
make test
```

Run specific tests using the `RUN` variable:

```bash
make test RUN=TestAccGitlabGroup
```

### Acceptance Tests

Acceptance tests run against a real GitLab instance. There are two options:

#### Option 1: Local GitLab Container (Recommended)

1. **Start GitLab container** (takes ~5 minutes to become healthy):

   ```bash
   unset GITLAB_TOKEN && make testacc-up
   ```

   **CRITICAL** if `Gitlab-license.txt` is present in the root of the repository,
   use this to start tests instead:

   ```bash
   unset GITLAB_TOKEN && SERVICE=gitlab-ee make testacc-up
   ```

2. **Run acceptance tests** (full suite takes ~60-90 minutes):

   ```bash
   unset GITLAB_TOKEN && make testacc
   ```

3. **Run specific acceptance tests**:

   ```bash
   unset GITLAB_TOKEN && make testacc RUN=TestAccGitlabGroup
   ```

4. **Run specific test function**:

   ```bash
   unset GITLAB_TOKEN && make testacc RUN=TestAccGitlabGroup_basic
   ```

5. **Stop GitLab container**:

   ```bash
   unset GITLAB_TOKEN && make testacc-down
   ```

#### Option 2: External GitLab Instance

```bash
make testacc GITLAB_TOKEN=your_token GITLAB_BASE_URL=https://your-gitlab.com/api/v4
```

**Note:** The token must have admin privileges.

#### GitLab Enterprise Edition Tests

If you have a `Gitlab-license.txt` file in the repository root:

```bash
make testacc-up SERVICE=gitlab-ee
```

#### Testing Against Specific GitLab Versions

```bash
make testacc-up GITLAB_CE_VERSION=15.0.0-ce.0
```

#### Test Tags

Different test suites can be run using the `TESTACCTAG` variable:

- `acceptance` (default) - Normal acceptance tests
- `flakey` - Tests with higher failure rates
- `settings` - Tests that permanently alter instance settings

```bash
make testacc TESTACCTAG=flakey
```

### Debugging Tests in an IDE

1. Start the GitLab container:

   ```bash
   make testacc-up
   ```

2. Configure your IDE's run configuration with these environment variables:

   ```bash
   GITLAB_BASE_URL=http://127.0.0.1:8085/api/v4
   TF_ACC=1
   ```

3. Run the test directly from your IDE

## Issue Triage Workflow

When triaging issues, especially bug reports, follow this systematic approach:

### 0. Understand the changes made if on a branch

If working on a branch, use a `git diff` between the current branch and `main`
to understand what changes exist on the branch. This helps context when determining
why the tests may be failing.

### 1. Reproduce the Issue Locally

First, verify the issue exists by running the relevant tests:

```bash
# Start GitLab container if not already running
make testacc-up

# Run the specific test related to the issue
make testacc RUN=TestAccGitlab<ResourceName>
```

**Expected outcome:** The test should fail, confirming the bug exists locally.

### 2. Investigate the Root Cause

- Review the test output and error messages
- Check the resource implementation in `internal/provider/`
- Review the GitLab API documentation at <https://docs.gitlab.com/api/>
- Compare the provider's behavior with the API's expected behavior

### 3. Implement the Fix

- Make necessary changes to the resource or data source
- Ensure changes follow the patterns in [CONTRIBUTING.md](CONTRIBUTING.md)
- Run `make reviewable` to format code and generate documentation

### 4. Verify the Fix

Re-run the previously failing test to ensure it now passes:

```bash
make testacc RUN=TestAccGitlab<ResourceName>
```

**Expected outcome:** The test should pass, confirming the fix works.

### 5. Run Related Tests

Run all tests for the affected resource to ensure no regressions:

```bash
make testacc RUN=TestAccGitlab<ResourceName>
```

### 6. Clean Up

```bash
make testacc-down
```

## Creating New Resources

When creating a new resource, follow the comprehensive guide at [docs/development/CreatingANewResource.md](docs/development/CreatingANewResource.md).

**Key requirements:**

1. Use the Terraform Plugin Framework (not SDK)
2. Place new resources in `internal/provider/`
3. Resource attributes must match GitLab API 1:1
4. Include examples in `/examples` directory
5. Write acceptance tests with at least:
   - Create step
   - Update step
   - Import step
6. Use inline test configurations (preferred over separate functions)
7. Use Import Verification test steps

## Migrating Datasources and Resources from SDK to Framework

If you are migrating a datasource or resource from the SDK to Terraform Framework Plugin, follow the guide at [Migration.md](Migration.md) for specific instructions.

## Code Guidelines

### Resource ID and Import

Resource IDs should be comprised from URL path variables in the GitLab API, separated by `:`.

Example: For `GET /projects/:id/environments/:environment_id`, the ID would be `123:456`.

### Test Configuration Style

**Preferred:** Inline test configuration

```go
Steps: []resource.TestStep{
  {
    Config: fmt.Sprintf(`
      resource "gitlab_instance_variable" "test" {
        key   = "key_%[1]s"
        value = "value-%[1]s"
      }`,
      rString,
    ),
  },
}
```

**Avoid:** Separate configuration functions (unless configuration is very large or heavily reused)

### Test Resource Creation

Use `testutil` helpers to create test resources instead of including them in test configurations:

```go
import "gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"

testProject := testutil.CreateProject(t)
// Use testProject.ID in your test configuration
```

## Common Commands Reference

| Command | Purpose |
|---------|---------|
| `make reviewable` | Run before committing (build, fmt, generate, test) |
| `make build` | Build the provider binary |
| `make test` | Run unit tests |
| `make test RUN=<pattern>` | Run specific unit tests |
| `make testacc-up` | Start local GitLab container |
| `make testacc` | Run all acceptance tests |
| `make testacc RUN=<pattern>` | Run specific acceptance tests |
| `make testacc-down` | Stop local GitLab container |
| `make fmt` | Format code and fix linting issues |
| `make generate` | Generate documentation |
| `make local` | Install provider locally for testing |

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `GITLAB_BASE_URL` | GitLab API base URL | `http://127.0.0.1:8085/api/v4` |
| `TF_ACC` | Enable acceptance tests | - |
| `RUN` | Test pattern filter | - |
| `TESTACCTAG` | Test tag to run | `acceptance` |
| `SERVICE` | GitLab service (ce/ee) | `gitlab-ce` |
| `GITLAB_CE_VERSION` | GitLab CE version | Latest |
| `GITLAB_EE_VERSION` | GitLab EE version | Latest |

## Important Notes

- **New resources must use Terraform Plugin Framework**, not SDK
- **Documentation is auto-generated** - do not edit files in `/docs` directly
- **Resource attributes must match GitLab API** naming and structure
- **All acceptance tests require admin token** on the GitLab instance
- **GitLab container takes ~5 minutes** to become healthy after starting
- **Full acceptance test suite takes 10-20 minutes** to complete
- **Always run `make reviewable`** before committing changes

## Getting Help

- **Issues:** <https://gitlab.com/gitlab-org/terraform-provider-gitlab/issues>
- **Discord:** <https://discord.gg/gitlab> (mention Patrick or Timo)
- **API Documentation:** <https://docs.gitlab.com/api/>
- **Terraform Plugin Framework:** <https://developer.hashicorp.com/terraform/plugin/framework>
- **Provider Design Principles:** <https://www.terraform.io/plugin/hashicorp-provider-design-principles>

## Support Policy

The provider supports:

- Latest 3 patch releases within a major release
- Breaking changes only on major releases
- Tests run against latest 3 patch releases
- Experimental GitLab features are not supported until Generally Available

For more details, see [README.md](README.md).
