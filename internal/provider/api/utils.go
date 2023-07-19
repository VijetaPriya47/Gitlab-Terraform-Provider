package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"
)

func Is404(err error) bool {
	if errResponse, ok := err.(*gitlab.ErrorResponse); ok &&
		errResponse.Response != nil &&
		errResponse.Response.StatusCode == 404 {
		return true
	}
	return false
}

// extractIIDFromGlobalID extracts the internal model ID from a global GraphQL ID.
//
// e.g. 'gid://gitlab/User/1' -> 1 or 'gid://gitlab/Project/42' -> 42
//
// see https://docs.gitlab.com/ee/development/api_graphql_styleguide.html#global-ids
func ExtractIIDFromGlobalID(globalID string) (int, error) {
	parts := strings.Split(globalID, "/")
	iid, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, fmt.Errorf("unable to extract iid from global id %q. Was looking for an integer after the last slash (/).", globalID)
	}
	return iid, nil
}
