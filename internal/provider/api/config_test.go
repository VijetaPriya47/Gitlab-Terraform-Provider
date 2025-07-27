package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gitlab.com/gitlab-org/api/client-go/config"
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

func TestConfig_NewGitLabClient_TokenPrecedence_ConfigFileUnset(t *testing.T) {
	t.Parallel()

	// GIVEN
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer any-value" {
			t.Fatalf("Expected authorization header value 'Bearer any-value', got %q", authHeader)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	config := &Config{
		Token:         "any-value",
		BaseURL:       testServer.URL,
		Insecure:      false,
		EarlyAuthFail: false,
		Context:       "",
		ConfigFile:    "",
	}

	// WHEN
	client, err := config.NewGitLabClient(t.Context())
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// We have to create a request to obtain the token used to configure the client
	req, err := client.NewRequest(http.MethodGet, "some-path", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Failed to Do request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestConfig_NewGitLabClient_TokenPrecedence_ConfigFileSet(t *testing.T) {
	t.Parallel()

	// GIVEN
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// THEN
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer any-value" {
			t.Fatalf("Expected authorization header value 'Bearer any-value', got %q", authHeader)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	config := &Config{
		Token:         "any-value",
		BaseURL:       testServer.URL,
		Insecure:      false,
		EarlyAuthFail: false,
		Context:       "",
		ConfigFile:    filepath.Join(t.TempDir(), "config.yaml"),
	}

	// WHEN
	client, err := config.NewGitLabClient(t.Context())
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// We have to create a request to obtain the token used to configure the client
	req, err := client.NewRequest(http.MethodGet, "some-path", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Failed to Do request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestConfig_NewGitLabClient_ConfigFile_CurrentContext(t *testing.T) {
	t.Parallel()

	// GIVEN
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// THEN
		authHeader := r.Header.Get("Private-Token")
		if authHeader != "any-value-from-config-current-context" {
			t.Fatalf("Expected authorization header value 'any-value-from-config', got %q", authHeader)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	configPath := setupTestConfig(t, testServer.URL)

	config := &Config{
		Token:         "",
		BaseURL:       "",
		Insecure:      false,
		EarlyAuthFail: false,
		Context:       "",
		ConfigFile:    configPath,
	}

	// WHEN
	client, err := config.NewGitLabClient(t.Context())
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// We have to create a request to obtain the token used to configure the client
	req, err := client.NewRequest(http.MethodGet, "some-path", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Failed to Do request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestConfig_NewGitLabClient_ConfigFile_ExplicitContext(t *testing.T) {
	t.Parallel()

	// GIVEN
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// THEN
		authHeader := r.Header.Get("Private-Token")
		if authHeader != "any-value-from-config-second-context" {
			t.Fatalf("Expected authorization header value 'any-value-from-config', got %q", authHeader)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	configPath := setupTestConfig(t, testServer.URL)

	config := &Config{
		Token:         "",
		BaseURL:       "",
		Insecure:      false,
		EarlyAuthFail: false,
		Context:       "example-com-2",
		ConfigFile:    configPath,
	}

	// WHEN
	client, err := config.NewGitLabClient(t.Context())
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// We have to create a request to obtain the token used to configure the client
	req, err := client.NewRequest(http.MethodGet, "some-path", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Failed to Do request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestConfig_NewGitLabClient_ConfigFile_FromDefaultLocation(t *testing.T) {
	// GIVEN
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// THEN
		authHeader := r.Header.Get("Private-Token")
		if authHeader != "any-value-from-config-current-context" {
			t.Fatalf("Expected authorization header value 'any-value-from-config', got %q", authHeader)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	configPath := setupTestConfig(t, testServer.URL)

	// NOTE: we can't test the default location in the users home here, so we just overwrite it
	// with our test location.
	t.Setenv("GITLAB_CONFIG", configPath)

	config := &Config{
		Token:         "",
		BaseURL:       "",
		Insecure:      false,
		EarlyAuthFail: false,
		Context:       "",
		ConfigFile:    "",
	}

	// WHEN
	client, err := config.NewGitLabClient(t.Context())
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// We have to create a request to obtain the token used to configure the client
	req, err := client.NewRequest(http.MethodGet, "some-path", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Failed to Do request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func setupTestConfig(t *testing.T, baseURL string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yaml")

	c, err := config.NewFromString(fmt.Sprintf(`
version: gitlab.com/config/v1beta1
instances:
  - name: example-com
    server: %s
auths:
  - name: example-com
    auth-info:
      personal-access-token:
        token: any-value-from-config-current-context
  - name: example-com-2
    auth-info:
      personal-access-token:
        token: any-value-from-config-second-context
contexts:
  - name: example-com
    instance: example-com
    auth: example-com
  - name: example-com-2
    instance: example-com
    auth: example-com-2
current-context: example-com`, baseURL), config.WithPath(configPath))

	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	err = c.Save()
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	return configPath
}
