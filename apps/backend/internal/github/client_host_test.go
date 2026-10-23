package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// recordingTransport answers every request with a fixed body and records the URL.
type recordingTransport struct {
	mu   sync.Mutex
	urls []string
	body string
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.urls = append(t.urls, req.URL.String())
	t.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Request:    req,
	}, nil
}

func (t *recordingTransport) lastURL() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.urls) == 0 {
		return ""
	}
	return t.urls[len(t.urls)-1]
}

var clientHostCases = []struct {
	host    string
	rest    string
	graphql string
}{
	{host: "github.com", rest: "https://api.github.com", graphql: "https://api.github.com/graphql"},
	{host: "octo.ghe.com", rest: "https://api.octo.ghe.com", graphql: "https://api.octo.ghe.com/graphql"},
	{host: "ghe.example.com", rest: "https://ghe.example.com/api/v3", graphql: "https://ghe.example.com/api/graphql"},
}

func TestTokenClientUsesHostEndpoints(t *testing.T) {
	for _, tc := range clientHostCases {
		t.Run(tc.host, func(t *testing.T) {
			transport := &recordingTransport{body: `{"login":"alice","data":{}}`}
			client := NewPATClient("token").WithHost(tc.host)
			client.httpClient.Transport = transport

			if _, err := client.GetAuthenticatedUser(context.Background()); err != nil {
				t.Fatalf("GetAuthenticatedUser: %v", err)
			}
			if got, want := transport.lastURL(), tc.rest+"/user"; got != want {
				t.Fatalf("REST URL = %q, want %q", got, want)
			}

			var out map[string]any
			if err := client.ExecuteGraphQL(context.Background(), "query{viewer{login}}", nil, &out); err != nil {
				t.Fatalf("ExecuteGraphQL: %v", err)
			}
			if got := transport.lastURL(); got != tc.graphql {
				t.Fatalf("GraphQL URL = %q, want %q", got, tc.graphql)
			}
		})
	}
}

func TestParseNextLinkStripsEnterpriseAPIBase(t *testing.T) {
	header := `<https://ghe.example.com/api/v3/repos/o/r/pulls?page=2>; rel="next"`
	if got, want := parseNextLink(header, "https://ghe.example.com/api/v3"), "/repos/o/r/pulls?page=2"; got != want {
		t.Fatalf("parseNextLink = %q, want %q", got, want)
	}
}

func TestAppClientMintsTokensOnHostAPI(t *testing.T) {
	_, pemBytes := testAppPrivateKey(t)
	for _, tc := range clientHostCases {
		t.Run(tc.host, func(t *testing.T) {
			transport := &recordingTransport{body: `{"token":"ghs_x","expires_at":"2030-01-01T00:00:00Z"}`}
			client, err := NewAppClient(tc.host, 123, pemBytes)
			if err != nil {
				t.Fatalf("NewAppClient: %v", err)
			}
			client.httpClient.Transport = transport
			_, _ = client.MintInstallationToken(context.Background(), 77, nil, nil)
			if got, want := transport.lastURL(), tc.rest+"/app/installations/77/access_tokens"; got != want {
				t.Fatalf("mint URL = %q, want %q", got, want)
			}
		})
	}
}

func TestOAuthClientExchangesCodeOnHostWeb(t *testing.T) {
	for _, tc := range clientHostCases {
		t.Run(tc.host, func(t *testing.T) {
			transport := &recordingTransport{body: `{"access_token":"ghu_x","token_type":"bearer"}`}
			client := NewGitHubOAuthClient(tc.host, "client-id", "client-secret")
			client.httpClient.Transport = transport
			_, _ = client.ExchangeUserCode(context.Background(), "code", "", "https://kandev.example/callback")
			if got, want := transport.urls[0], endpointsForHost(tc.host).Web+"/login/oauth/access_token"; got != want {
				t.Fatalf("token exchange URL = %q, want %q", got, want)
			}
		})
	}
}

func TestManifestConversionUsesHostAPI(t *testing.T) {
	for _, tc := range clientHostCases {
		t.Run(tc.host, func(t *testing.T) {
			transport := &recordingTransport{body: `{}`}
			client := NewManifestConversionClient()
			client.httpClient.Transport = transport
			_, _ = client.Convert(context.Background(), tc.host, "code")
			if got, want := transport.urls[0], tc.rest+"/app-manifests/code/conversions"; got != want {
				t.Fatalf("conversion URL = %q, want %q", got, want)
			}
		})
	}
}

func TestManifestRegistrationURLUsesHostWeb(t *testing.T) {
	for _, tc := range clientHostCases {
		web := endpointsForHost(tc.host).Web
		if got, want := manifestRegistrationURL(tc.host, ManifestOwnerUser, "alice"), web+"/settings/apps/new"; got != want {
			t.Errorf("user URL = %q, want %q", got, want)
		}
		if got, want := manifestRegistrationURL(tc.host, ManifestOwnerOrganization, "octo"), web+"/organizations/octo/settings/apps/new"; got != want {
			t.Errorf("organization URL = %q, want %q", got, want)
		}
	}
}

func TestAppInstallationURLUsesHostPath(t *testing.T) {
	tests := map[string]string{
		"github.com":      "https://github.com/apps/kandev/installations/new?state=s1",
		"octo.ghe.com":    "https://octo.ghe.com/apps/kandev/installations/new?state=s1",
		"ghe.example.com": "https://ghe.example.com/github-apps/kandev/installations/new?state=s1",
	}
	for host, want := range tests {
		if got := appInstallationURL(host, "kandev", "s1"); got != want {
			t.Errorf("appInstallationURL(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestOAuthAuthorizeURLUsesHostWeb(t *testing.T) {
	for _, tc := range clientHostCases {
		if got, want := oauthAuthorizeURL(tc.host).String(), endpointsForHost(tc.host).Web+"/login/oauth/authorize"; got != want {
			t.Errorf("oauthAuthorizeURL(%q) = %q, want %q", tc.host, got, want)
		}
	}
}

func TestAppUserTokenClientUsesHostAPI(t *testing.T) {
	transport := &recordingTransport{body: `{"login":"alice"}`}
	client := NewAppUserTokenClient("ghe.example.com", "ghu_x", 42, "alice")
	client.httpClient.Transport = transport
	_, _ = client.GetAuthenticatedUser(context.Background())
	if got, want := transport.lastURL(), "https://ghe.example.com/api/v3/user"; got != want {
		t.Fatalf("REST URL = %q, want %q", got, want)
	}
}
