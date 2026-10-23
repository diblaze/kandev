package github

import "testing"

func TestNormalizeGitHubHost(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty defaults to github.com", raw: "", want: "github.com"},
		{name: "github.com", raw: "github.com", want: "github.com"},
		{name: "www alias", raw: "www.github.com", want: "github.com"},
		{name: "GHE.com tenant", raw: "octo.ghe.com", want: "octo.ghe.com"},
		{name: "enterprise server", raw: "ghe.example.com", want: "ghe.example.com"},
		{name: "uppercase and padding", raw: "  GHE.Example.COM ", want: "ghe.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeGitHubHost(tt.raw)
			if err != nil {
				t.Fatalf("normalizeGitHubHost(%q) error = %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeGitHubHost(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestNormalizeGitHubHostRejectsUnsafeForms(t *testing.T) {
	for _, raw := range []string{"https://x", "x/y", "x:443", "a@b", "-x", "a b", "a\tb"} {
		t.Run(raw, func(t *testing.T) {
			if got, err := normalizeGitHubHost(raw); err == nil {
				t.Fatalf("normalizeGitHubHost(%q) = %q, want error", raw, got)
			}
		})
	}
}

func TestEndpointsForHost(t *testing.T) {
	tests := []struct {
		host string
		want githubEndpoints
	}{
		{host: "github.com", want: githubEndpoints{
			Web: "https://github.com", REST: "https://api.github.com",
			GraphQL: "https://api.github.com/graphql", AppInstallPrefix: "/apps/",
		}},
		{host: "", want: githubEndpoints{
			Web: "https://github.com", REST: "https://api.github.com",
			GraphQL: "https://api.github.com/graphql", AppInstallPrefix: "/apps/",
		}},
		{host: "octo.ghe.com", want: githubEndpoints{
			Web: "https://octo.ghe.com", REST: "https://api.octo.ghe.com",
			GraphQL: "https://api.octo.ghe.com/graphql", AppInstallPrefix: "/apps/",
		}},
		{host: "ghe.example.com", want: githubEndpoints{
			Web: "https://ghe.example.com", REST: "https://ghe.example.com/api/v3",
			GraphQL: "https://ghe.example.com/api/graphql", AppInstallPrefix: "/github-apps/",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := endpointsForHost(tt.host); got != tt.want {
				t.Fatalf("endpointsForHost(%q) = %+v, want %+v", tt.host, got, tt.want)
			}
		})
	}
}

func TestWebHostFromURL(t *testing.T) {
	valid := map[string]string{
		"https://github.com/o/r/pull/1":      "github.com",
		"https://octo.ghe.com/o/r/issues/2":  "octo.ghe.com",
		"https://GHE.Example.com/o/r/pull/3": "ghe.example.com",
	}
	for raw, want := range valid {
		if got, err := WebHostFromURL(raw); err != nil || got != want {
			t.Errorf("WebHostFromURL(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
}

func TestWebHostFromURLRejectsMalformedURL(t *testing.T) {
	for _, raw := range []string{
		"http://ghe.example.com/o/r", "https://user@ghe.example.com/o/r",
		"https://ghe.example.com:8443/o/r", "not a url", "",
	} {
		if got, err := WebHostFromURL(raw); err == nil {
			t.Errorf("WebHostFromURL(%q) = %q, want error", raw, got)
		}
	}
}

func TestHostFromProviderHost(t *testing.T) {
	valid := map[string]string{
		"https://github.com":      "github.com",
		"https://octo.ghe.com":    "octo.ghe.com",
		"https://ghe.example.com": "ghe.example.com",
		"ghe.example.com":         "ghe.example.com",
		"OCTO.ghe.com":            "octo.ghe.com",
	}
	for raw, want := range valid {
		if got, err := HostFromProviderHost(raw); err != nil || got != want {
			t.Errorf("HostFromProviderHost(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
}

func TestHostFromProviderHostRejectsUnknownIdentity(t *testing.T) {
	for _, raw := range []string{"", "https://x/y", "http://ghe.example.com", "https://ghe.example.com:8443"} {
		if got, err := HostFromProviderHost(raw); err == nil {
			t.Errorf("HostFromProviderHost(%q) = %q, want error", raw, got)
		}
	}
}
