package api

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/xanzy/go-gitlab"
)

// Checks if the error represents a 404 response
func Is404(err error) bool {
	// If the error is a typed response
	if errResponse, ok := err.(*gitlab.ErrorResponse); ok &&
		errResponse.Response != nil &&
		errResponse.Response.StatusCode == 404 {
		return true
	}

	// This can also come back as a string 404 from go-gitlab
	if err != nil && err.Error() == "404 Not Found" {
		return true
	}

	return false
}

// The ISO constant for parsing dates to a `gitlab.ISOTime` value
const Iso8601 = "2006-01-02"

// Checks if the error represents a 403 response
func Is403(err error) bool {
	if errResponse, ok := err.(*gitlab.ErrorResponse); ok &&
		errResponse.Response != nil &&
		errResponse.Response.StatusCode == 403 {
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

// CurrentTime returns the current time or a testing time based on an environment variable.
// If the environment variable GITLAB_TESTING_TIME is set, it will be used as the current time.
// This function is used to test time-dependent resources, so that the current time can be mocked
// if needed.
func CurrentTime() time.Time {
	testingTime, err := time.Parse(time.RFC3339, os.Getenv("GITLAB_TESTING_TIME"))
	if err == nil {
		tflog.Warn(context.Background(), "[WARNING] Use of `GITLAB_TESTING_TIME` detected. Using mocked time instead of system time. Disable for production use.", map[string]interface{}{
			"testing_time": testingTime.Format(time.RFC3339),
		})
		return testingTime
	}
	return time.Now()
}
