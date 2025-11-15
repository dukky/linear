package client

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
	"github.com/andreasholley/linear-cli/internal/auth"
)

const linearAPIURL = "https://api.linear.app/graphql"

// authedTransport wraps an HTTP RoundTripper with authentication
type authedTransport struct {
	wrapped http.RoundTripper
	apiKey  string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return t.wrapped.RoundTrip(req)
}

// NewClient creates a new Linear API client
func NewClient() (graphql.Client, error) {
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &authedTransport{
			wrapped: http.DefaultTransport,
			apiKey:  apiKey,
		},
	}

	return graphql.NewClient(linearAPIURL, httpClient), nil
}
