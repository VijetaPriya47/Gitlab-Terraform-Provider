<a href="https://terraform.io">
    <img src=".gitlab/terraform_logo.svg" alt="Terraform logo" title="Terraform" align="right" height="50" />
</a>

# Terraform Provider for GitLab

- [Documentation](https://www.terraform.io/docs/providers/gitlab/index.html)
- [Issues](https://gitlab.com/gitlab-org/terraform-provider-gitlab/issues)
- [Discord Server](https://discord.gg/gitlab)

The Terraform GitLab Provider is a plugin for Terraform that allows for the full lifecycle management of
GitLab resources, like users, groups and projects.

## Contributing

Check out the [CONTRIBUTING.md](/CONTRIBUTING.md) guide for tips on how to contribute and develop the provider.

If you'd like to reach out for some help or a pair coding session, please mention Patrick or Timo on the Discord server, and we'll work with you to set up a pair coding session!

## Troubleshooting support

This is a community maintained project. If you have a paid GitLab subscription, please note that GitLab Terraform Provider is not packaged as a part of GitLab, and falls outside of the scope of support. For more information, see GitLab's [Statement of Support](https://about.gitlab.com/support/statement-of-support.html).

We support the following versions:

- Latest 3 patch releases within a major release. For example, if current is 17.8, we support 17.6-17.8. Or if current is 18.1, we support 18.0-18.1.
- We introduce any breaking changes on major releases only. For example, 17.0 or 18.0.
- We run tests against the latest 3 patch releases regardless of whether these cross a major release boundary. For example, if current is 17.8, we test 17.6-17.8. Or if current is 18.1, we test 17.11-18.1.

All other versions are best effort support.

We do not support experimental GitLab features until they are enabled by default or made Generally Available.

Note, that the compatibility between a provider release and GitLab itself **cannot** be inferred from the
release version. New features added to GitLab may not be added to the provider until later versions.
Equally, features removed or deprecated in GitLab may not be removed or deprecated from the provider until later versions.

Please [fill out an issue](https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues) in this project's issue tracker and someone from the community will respond as soon as they are available to help you.
