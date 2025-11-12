package api

type GraphQLMemberRole struct {
	BaseAccessLevel struct {
		StringValue string `json:"stringValue"`
	} `json:"baseAccessLevel"`
	CreatedAt          string `json:"createdAt"`
	Description        string `json:"description"`
	EditPath           string `json:"editPath"`
	EnabledPermissions struct {
		Nodes []struct {
			Value string `json:"value"`
		} `json:"nodes"`
	} `json:"enabledPermissions"`
	ID   string `json:"id"`
	Name string `json:"name"`
}
