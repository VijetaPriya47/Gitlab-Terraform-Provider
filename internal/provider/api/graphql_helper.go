package api

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Returns a GraphQL ID from the project ID or Path
func GetProjectGIDFromID(ctx context.Context, client *gitlab.Client, projectId string) (*ProjectIdentifiers, error) {
	var project *gitlab.Project

	project, _, err := client.Projects.GetProject(projectId, nil)
	if err != nil {
		return nil, err
	}

	// Call the GraphQL Project API to get the GID
	var response getProjectIDStruct
	_, err = client.GraphQL.Do(gitlab.GraphQLQuery{Query: fmt.Sprintf(`query { project(fullPath: "%s") { id } }`, project.PathWithNamespace)}, &response)

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
	_, err = client.GraphQL.Do(gitlab.GraphQLQuery{Query: fmt.Sprintf(`query { group(fullPath: "%s") { id } }`, group.FullPath)}, &response)

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
