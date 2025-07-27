package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/api/client-go/config"
	"golang.org/x/oauth2"
)

// Config is per-provider, specifies where to connect to gitlab
type Config struct {
	Token               string
	BaseURL             string
	Insecure            bool
	CACertFile          string
	ClientCert          string
	ClientKey           string
	EarlyAuthFail       bool
	Retries             int
	Headers             map[string]any
	Context             string
	ConfigFile          string
	EnableAutoCISupport bool
}

// Client returns a *gitlab.Client to interact with the configured gitlab instance
func (c *Config) NewGitLabClient(ctx context.Context) (*gitlab.Client, error) {
	var client *gitlab.Client

	opts := []gitlab.ClientOptionFunc{}

	if c.Headers != nil {
		stringMap := make(map[string]string, len(c.Headers))
		for k, v := range c.Headers {
			stringMap[k] = fmt.Sprintf("%v", v)
		}
		opts = append(opts, gitlab.WithRequestOptions(gitlab.WithHeaders(stringMap)))
	}

	if c.Token == "" {
		// Assuming that the configuration is made via file
		options := []config.ConfigOption{}
		if c.ConfigFile != "" {
			options = append(options, config.WithPath(c.ConfigFile))
		}
		if c.EnableAutoCISupport {
			options = append(options, config.WithAutoCISupport())
		}

		cfg := config.New(options...)
		if err := cfg.Load(); err != nil {
			return nil, err
		}

		if c.Context != "" {
			cl, err := cfg.NewClientForContext(c.Context, opts...)
			if err != nil {
				return nil, err
			}
			client = cl
		} else {
			cl, err := cfg.NewClient(opts...)
			if err != nil {
				return nil, err
			}
			client = cl
		}
	} else {
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

		opts = append(opts,
			gitlab.WithHTTPClient(
				&http.Client{
					Transport: logging.NewSubsystemLoggingHTTPTransport("GitLab", t),
				},
			),
		)

		if c.BaseURL != "" {
			opts = append(opts, gitlab.WithBaseURL(c.BaseURL))
		}

		opts = append(opts, gitlab.WithCustomRetryMax(c.Retries))

		// The OAuth method is also compatible with project/group/personal access and job tokens because they are all usable as Bearer tokens.
		// Although the job token API access is very limited.
		// see https://docs.gitlab.com/api/rest/authentication/
		cl, err := gitlab.NewAuthSourceClient(gitlab.OAuthTokenSource{
			TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token}),
		}, opts...)
		if err != nil {
			return nil, err
		}
		client = cl
	}

	// Test the credentials by checking we can get information about the authenticated user.
	var err error
	if c.EarlyAuthFail {
		_, _, err = client.Users.CurrentUser(gitlab.WithContext(ctx))
	}

	return client, err
}
