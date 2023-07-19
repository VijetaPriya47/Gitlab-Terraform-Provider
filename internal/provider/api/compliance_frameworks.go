package api

type GraphQLComplianceFramework struct {
	ID                            string `json:"id"` // This comes back as a globally unique ID
	Name                          string `json:"name"`
	Description                   string `json:"description"`
	Color                         string `json:"color"`
	DefaultFramework              bool   `json:"default"`
	PipelineConfigurationFullPath string `json:"pipelineConfigurationFullPath"`
}
