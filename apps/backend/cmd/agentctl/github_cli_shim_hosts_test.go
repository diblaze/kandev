package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var githubCLIShimTokenVariables = []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

func envHasKey(env []string, key string) bool {
	for _, entry := range env {
		if name, _, _ := strings.Cut(entry, "="); name == key {
			return true
		}
	}
	return false
}

func TestGitHubCLIShimHostsRunGHOnTheLeaseHostWithItsTokenVariable(t *testing.T) {
	tests := []struct {
		host     string
		tokenKey string
	}{
		{host: "github.com", tokenKey: "GH_TOKEN"},
		{host: "octo.ghe.com", tokenKey: "GH_TOKEN"},
		{host: "ghe.example.com", tokenKey: "GH_ENTERPRISE_TOKEN"},
	}
	for _, test := range tests {
		t.Run(test.host, func(t *testing.T) {
			var got githubBrokerResolveRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Errorf("decode request: %v", err)
				}
				_, _ = io.WriteString(w, `{"username":"x-access-token","password":"lease-host-token"}`)
			}))
			t.Cleanup(server.Close)
			env := githubCredentialTestEnv(server.URL)
			env["PATH"] = "/shim:/usr/bin"
			env["GH_REPO"] = test.host + "/acme/backend"
			for _, key := range githubCLIShimTokenVariables {
				env[key] = "parent-" + strings.ToLower(key)
			}
			env[envGitHubCredentialScopes] = `[
				{"lease":"public-lease","task_id":"task-1","session_id":"session-1","repository_id":"repo-1","owner":"acme","repo":"backend","host":"github.com"},
				{"lease":"host-lease","task_id":"task-1","session_id":"session-1","repository_id":"repo-2","owner":"acme","repo":"backend","host":"` + test.host + `"}
			]`
			var childEnv []string
			err := runGitHubCLIShim(
				context.Background(), []string{"pr", "view"}, strings.NewReader(""), io.Discard, io.Discard,
				lookupEnv(env), func() []string { return envMap(env) }, server.Client(), "/shim",
				func(string, string) (string, error) { return "/usr/bin/gh", nil },
				func(_ context.Context, _ string, _ []string, child []string, _ io.Reader, _, _ io.Writer) error {
					childEnv = child
					return nil
				},
			)
			if err != nil {
				t.Fatalf("runGitHubCLIShim() error = %v", err)
			}

			if got.Host != test.host {
				t.Fatalf("broker scope host = %q, want %q", got.Host, test.host)
			}
			if value := envValue(childEnv, "GH_HOST"); value != test.host {
				t.Fatalf("child GH_HOST = %q, want %q", value, test.host)
			}
			if value := envValue(childEnv, test.tokenKey); value != "lease-host-token" {
				t.Fatalf("child %s = %q, want the lease credential", test.tokenKey, value)
			}
			for _, key := range githubCLIShimTokenVariables {
				if key != test.tokenKey && envHasKey(childEnv, key) {
					t.Fatalf("child %s = %q, want removed for %s", key, envValue(childEnv, key), test.host)
				}
			}
		})
	}
}

func TestGitHubCLIShimHostsRefuseAHostnameOtherThanTheLeaseHost(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantRun bool
	}{
		{name: "another tenant", args: []string{"api", "--hostname", "evil.ghe.com", "/user"}},
		{name: "another tenant inline", args: []string{"api", "--hostname=octo.ghe.com", "/user"}},
		{name: "missing value", args: []string{"api", "/user", "--hostname"}},
		{name: "lease host", args: []string{"api", "--hostname", "GitHub.com", "/user"}, wantRun: true},
		{name: "API URL on another host", args: []string{"api", "https://ghe.example.com/api/v3/user"}},
		{name: "URL after the flag terminator", args: []string{"api", "--", "https://evil.example.com/user"}},
		{name: "lookalike host suffix", args: []string{"api", "https://api.evilgithub.com/user"}},
		{name: "API URL on the lease host", args: []string{"api", "https://api.github.com/user"}, wantRun: true},
		{name: "pull request URL on the lease host", args: []string{"pr", "view", "https://GitHub.com/o/r/pull/1"}, wantRun: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"username":"x-access-token","password":"public-lease-token"}`)
			}))
			t.Cleanup(server.Close)
			env := githubCredentialTestEnv(server.URL)
			env["PATH"] = "/shim:/usr/bin"
			ran := false
			err := runGitHubCLIShim(
				context.Background(), test.args, strings.NewReader(""), io.Discard, io.Discard,
				lookupEnv(env), func() []string { return envMap(env) }, server.Client(), "/shim",
				func(string, string) (string, error) { return "/usr/bin/gh", nil },
				func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
					ran = true
					return nil
				},
			)
			if ran != test.wantRun || (err == nil) != test.wantRun {
				t.Fatalf("runGitHubCLIShim(%q) ran = %v, error = %v; want ran = %v", test.args, ran, err, test.wantRun)
			}
		})
	}
}
