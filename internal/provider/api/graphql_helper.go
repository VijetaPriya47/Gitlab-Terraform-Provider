package api

import (
	"context"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/gitlab-org/api/client-go"
)

// Helper method for modifying client requests appropriately for sending a GraphQL call instead of a REST call.
func SendGraphQLRequest(ctx context.Context, client *gitlab.Client, query GraphQLQuery, response interface{}) (interface{}, error) {
	request, err := client.NewRequest("POST", "", query, nil)
	if err != nil {
		return nil, err
	}
	// Overwrite the path of the existing request, as otherwise client-go appends /api/v4 instead.
	request.URL.Path = "/api/graphql"
	resp, err := client.Do(request, response)
	if err != nil {
		// Read the body of the request so we can log it
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		tflog.Debug(ctx, "GraphQL request failed", map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return nil, err
	}
	return response, nil
}

// Represents a GraphQL call to the API. All GraphQL calls are a string passed to the "query" parameter, so they should be included here.
type GraphQLQuery struct {
	Query string `json:"query"`
}

// Returns a GraphQL ID from the project ID or Path
func GetProjectGIDFromID(ctx context.Context, client *gitlab.Client, projectId string) (*ProjectIdentifiers, error) {
	var project *gitlab.Project

	project, _, err := client.Projects.GetProject(projectId, nil)
	if err != nil {
		return nil, err
	}

	// Call the GraphQL Project API to get the GID
	var response getProjectIDStruct
	_, err = SendGraphQLRequest(ctx, client, GraphQLQuery{Query: fmt.Sprintf(`query { project(fullPath: "%s") { id } }`, project.PathWithNamespace)}, &response)

	return &ProjectIdentifiers{
		ProjectID:       project.ID,
		ProjectFullPath: project.PathWithNamespace,
		ProjectGQLID:    response.Data.Project.ID,
	}, err
}

type getProjectIDStruct struct {
	Data struct {
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	} `json:"data"`
}

// A type including all the relevant idenfiers for a project
type ProjectIdentifiers struct {
	ProjectID       int
	ProjectFullPath string
	ProjectGQLID    string
}

// Returns a GraphQL ID from the group ID or Path
func GetGroupGIDFromID(ctx context.Context, client *gitlab.Client, groupId string) (*GroupIdentifiers, error) {
	var group *gitlab.Group

	group, _, err := client.Groups.GetGroup(groupId, nil)
	if err != nil {
		return nil, err
	}

	// Call the GraphQL Project API to get the GID
	var response getGroupIDStruct
	_, err = SendGraphQLRequest(ctx, client, GraphQLQuery{Query: fmt.Sprintf(`query { group(fullPath: "%s") { id } }`, group.FullPath)}, &response)

	return &GroupIdentifiers{
		GroupID:       group.ID,
		GroupFullPath: group.FullPath,
		GroupGQLID:    response.Data.Group.ID,
	}, err
}

type getGroupIDStruct struct {
	Data struct {
		Group struct {
			ID string `json:"id"`
		} `json:"group"`
	} `json:"data"`
}

// A type including all the relevant idenfiers for a project
type GroupIdentifiers struct {
	GroupID       int
	GroupFullPath string
	GroupGQLID    string
}
