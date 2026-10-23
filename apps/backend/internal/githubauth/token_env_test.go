package githubauth

import (
	"slices"
	"testing"
)

func TestTokenEnvForHostFollowsGitHubCLIHostRule(t *testing.T) {
	tests := []struct {
		host, primary, fallback string
	}{
		{host: "github.com", primary: "GH_TOKEN", fallback: "GITHUB_TOKEN"},
		{host: "GitHub.com", primary: "GH_TOKEN", fallback: "GITHUB_TOKEN"},
		{host: "octo.ghe.com", primary: "GH_TOKEN", fallback: "GITHUB_TOKEN"},
		{host: "ghe.example.com", primary: "GH_ENTERPRISE_TOKEN", fallback: "GITHUB_ENTERPRISE_TOKEN"},
		{host: "ghe.com", primary: "GH_ENTERPRISE_TOKEN", fallback: "GITHUB_ENTERPRISE_TOKEN"},
	}
	for _, tt := range tests {
		primary, fallback := TokenEnvForHost(tt.host)
		if primary != tt.primary || fallback != tt.fallback {
			t.Errorf("TokenEnvForHost(%q) = %q, %q; want %q, %q", tt.host, primary, fallback, tt.primary, tt.fallback)
		}
	}
}

func TestEnterpriseHostsParsesTheListedHosts(t *testing.T) {
	got := EnterpriseHosts(" OCTO.ghe.com,ghe.example.com,,github.com,octo.ghe.com ")
	want := []string{"octo.ghe.com", "ghe.example.com"}
	if !slices.Equal(got, want) {
		t.Fatalf("EnterpriseHosts() = %q, want %q", got, want)
	}
}

func TestEnterpriseHostsDropsUnsafeEntries(t *testing.T) {
	raw := "-flag.example.com,ghe.example.com:8443,user@ghe.example.com,ghe.example.com/path," +
		"$(id).example.com,ghe example.com,ghe.example.com;rm,'quoted'.example.com"
	if got := EnterpriseHosts(raw); len(got) != 0 {
		t.Fatalf("EnterpriseHosts() = %q, want none", got)
	}
}
