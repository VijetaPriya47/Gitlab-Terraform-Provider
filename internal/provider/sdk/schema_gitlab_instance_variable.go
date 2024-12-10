package sdk

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func gitlabInstanceVariableGetSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"key": {
			Description:  "The name of the variable.",
			Type:         schema.TypeString,
			ForceNew:     true,
			Required:     true,
			ValidateFunc: StringIsGitlabVariableName,
		},
		"value": {
			Description: "The value of the variable.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"description": {
			Description:      "The description of the variable. Maximum of 255 characters.",
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 255)),
		},
		"variable_type": {
			Description:      fmt.Sprintf("The type of a variable. Valid values are: %s. Default is `env_var`.", utils.RenderValueListForDocs(gitlabVariableTypeValues)),
			Type:             schema.TypeString,
			Optional:         true,
			Default:          "env_var",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(gitlabVariableTypeValues, false)),
		},
		"protected": {
			Description: "If set to `true`, the variable will be passed only to pipelines running on protected branches and tags. Defaults to `false`.",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		},
		"masked": {
			Description: "If set to `true`, the value of the variable will be hidden in job logs. The value must meet the [masking requirements](https://docs.gitlab.com/ee/ci/variables/#masked-variables). Defaults to `false`.",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		},
		"raw": {
			Description: "Whether the variable is treated as a raw string. Default: false. When true, variables in the value are not expanded.",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		},
	}
}

func gitlabInstanceVariableToStateMap(variable *gitlab.InstanceVariable) map[string]interface{} {
	stateMap := make(map[string]interface{})
	stateMap["key"] = variable.Key
	stateMap["value"] = variable.Value
	stateMap["description"] = variable.Description
	stateMap["variable_type"] = variable.VariableType
	stateMap["protected"] = variable.Protected
	stateMap["masked"] = variable.Masked
	stateMap["raw"] = variable.Raw
	return stateMap
}
