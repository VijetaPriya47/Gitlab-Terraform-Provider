package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/oauth2"
)

// Config is per-provider, specifies where to connect to gitlab
type Config struct {
	Token         string
	BaseURL       string
	Insecure      bool
	CACertFile    string
	ClientCert    string
	ClientKey     string
	EarlyAuthFail bool
	Retries       int
	Headers       map[string]any
}

// Client returns a *gitlab.Client to interact with the configured gitlab instance
func (c *Config) NewGitLabClient(ctx context.Context) (*gitlab.Client, error) {
	if c.Token == "" {
		return nil, errors.New("No GitLab token configured, either use the `token` provider argument or set it as `GITLAB_TOKEN` environment variable")
	}

	// Configure TLS/SSL
	tlsConfig := &tls.Config{}

	// If a CACertFile has been specified, use that for cert validation
	if c.CACertFile != "" {
		caCert, err := os.ReadFile(c.CACertFile)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	// If configured as insecure, turn off SSL verification
	if c.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}

	// add client cert and key to connection
	if c.ClientCert != "" && c.ClientKey != "" {
		clientPair, err := tls.LoadX509KeyPair(c.ClientCert, c.ClientKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{clientPair}
	}

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = tlsConfig
	t.MaxIdleConnsPerHost = 100

	opts := []gitlab.ClientOptionFunc{
		gitlab.WithHTTPClient(
			&http.Client{
				Transport: logging.NewSubsystemLoggingHTTPTransport("GitLab", t),
			},
		),
	}

	if c.BaseURL != "" {
		opts = append(opts, gitlab.WithBaseURL(c.BaseURL))
	}

	if c.Headers != nil {
		stringMap := make(map[string]string, len(c.Headers))
		for k, v := range c.Headers {
			stringMap[k] = fmt.Sprintf("%v", v)
		}
		opts = append(opts, gitlab.WithRequestOptions(
			gitlab.WithHeaders(stringMap),
		))
	}

	opts = append(opts, gitlab.WithCustomRetryMax(c.Retries))

	// The OAuth method is also compatible with project/group/personal access and job tokens because they are all usable as Bearer tokens.
	// Although the job token API access is very limited.
	// see https://docs.gitlab.com/api/rest/authentication/
	client, err := gitlab.NewAuthSourceClient(gitlab.OAuthTokenSource{
		TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token}),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Test the credentials by checking we can get information about the authenticated user.
	if c.EarlyAuthFail {
		_, _, err = client.Users.CurrentUser(gitlab.WithContext(ctx))
	}

	return client, err
}
