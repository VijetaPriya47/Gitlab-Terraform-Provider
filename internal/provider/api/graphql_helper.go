package api

import (
	"context"
	"fmt"

	"github.com/xanzy/go-gitlab"
)

// Helper method for modifying client requests appropriately for sending a GraphQL call instead of a REST call.
func SendGraphQLRequest(ctx context.Context, client *gitlab.Client, query GraphQLQuery, response interface{}) (interface{}, error) {
	request, err := client.NewRequest("POST", "", query, nil)
	if err != nil {
		return nil, err
	}
	// Overwrite the path of the existing request, as otherwise the go-gitlab client appends /api/v4 instead.
	request.URL.Path = "/api/graphql"
	if _, err = client.Do(request, response); err != nil {
		return nil, err
	}
	return response, nil
}

// Represents a GraphQL call to the API. All GraphQL calls are a string passed to the "query" parameter, so they should be included here.
type GraphQLQuery struct {
	Query string `json:"query"`
}

// Returns a GraphQL ID from project ID or Path
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
