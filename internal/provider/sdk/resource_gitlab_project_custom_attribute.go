package sdk

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var _ = registerResource("gitlab_project_custom_attribute", func() *schema.Resource {
	return CreateCustomAttributeResource(
		"project",
		func(client *gitlab.Client) CustomAttributeGetter {
			return client.CustomAttribute.GetCustomProjectAttribute
		},
		func(client *gitlab.Client) CustomAttributeSetter {
			return client.CustomAttribute.SetCustomProjectAttribute
		},
		func(client *gitlab.Client) CustomAttributeDeleter {
			return client.CustomAttribute.DeleteCustomProjectAttribute
		},
		`The `+"`gitlab_project_custom_attribute`"+` resource allows to manage custom attributes for a project.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/api/custom_attributes/)`,
	)
})
