package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfig_CustomHeaders(t *testing.T) {
	t.Parallel()

	headerValue := "test-value"
	calls := 0

	// Create a mock "current user" endpoint to receive the early auth calls
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		// Check if our header value is applied
		if r.Header.Get("X-Custom-Header") != headerValue {
			t.Errorf("Expected X-Custom-Header to be '%s', got '%s'", headerValue, r.Header.Get("X-Custom-Header"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1, "username": "test-user"}`)) // nolint - don't need to err check writing the response in the test
	}))
	defer mockServer.Close()

	// Create a test configuration with the mock server URL
	testConfig := &Config{
		Token:         "test-token",
		BaseURL:       mockServer.URL,
		EarlyAuthFail: true,
		Headers: map[string]any{
			"X-Custom-Header": headerValue,
		},
	}

	// Create a new GitLab client using the test configuration
	_, err := testConfig.NewGitLabClient(context.Background())
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// There should only be one call because the early auth should have been successful, and only call once.
	if calls != 1 {
		t.Errorf("Expected 1 call to the mock server, got %d", calls)
	}
}
