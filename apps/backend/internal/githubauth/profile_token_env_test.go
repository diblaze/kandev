package githubauth

import "testing"

func TestProfileTokenEnvForHostBindsGitHubTokenToGHHost(t *testing.T) {
	tests := []struct {
		name, ghHost, host, primary, fallback string
	}{
		{name: "unset GH_HOST binds github.com", host: "github.com", primary: "GH_TOKEN", fallback: "GITHUB_TOKEN"},
		{name: "github.com-bound token never reaches a tenant", host: "octo.ghe.com"},
		{name: "tenant GH_HOST binds the tenant", ghHost: "octo.ghe.com", host: "octo.ghe.com", primary: "GH_TOKEN", fallback: "GITHUB_TOKEN"},
		{name: "tenant-bound token never reaches github.com", ghHost: "octo.ghe.com", host: "github.com"},
		{name: "tenant-bound token never reaches another tenant", ghHost: "octo.ghe.com", host: "other.ghe.com"},
		{name: "tenant-bound GH_TOKEN never reaches a server", ghHost: "octo.ghe.com", host: "ghe.example.com",
			primary: "GH_ENTERPRISE_TOKEN", fallback: "GITHUB_ENTERPRISE_TOKEN"},
		{name: "enterprise token answers any server without a server GH_HOST", host: "ghe.example.com",
			primary: "GH_ENTERPRISE_TOKEN", fallback: "GITHUB_ENTERPRISE_TOKEN"},
		{name: "server GH_HOST binds the enterprise token to it", ghHost: "ghe.example.com", host: "ghe.example.com",
			primary: "GH_ENTERPRISE_TOKEN", fallback: "GITHUB_ENTERPRISE_TOKEN"},
		{name: "server-bound enterprise token never reaches another server", ghHost: "ghe.example.com", host: "ghe2.example.com"},
		{name: "server-bound GH_TOKEN never reaches github.com", ghHost: "ghe.example.com", host: "github.com"},
		{name: "GH_HOST is normalized", ghHost: " OCTO.GHE.COM ", host: "octo.ghe.com", primary: "GH_TOKEN", fallback: "GITHUB_TOKEN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary, fallback := ProfileTokenEnvForHost(tt.ghHost, tt.host)
			if primary != tt.primary || fallback != tt.fallback {
				t.Errorf("ProfileTokenEnvForHost(%q, %q) = %q, %q; want %q, %q",
					tt.ghHost, tt.host, primary, fallback, tt.primary, tt.fallback)
			}
		})
	}
}

func TestProfileTokenEnvForHostMalformedGHHostBindsNoToken(t *testing.T) {
	for _, ghHost := range []string{"https://octo.ghe.com", "octo.ghe.com/acme", "octo.ghe.com:8443", "-octo.ghe.com", "user@github.com", "octo ghe.com"} {
		for _, host := range []string{"github.com", "octo.ghe.com", "ghe.example.com"} {
			if primary, fallback := ProfileTokenEnvForHost(ghHost, host); primary != "" || fallback != "" {
				t.Errorf("ProfileTokenEnvForHost(%q, %q) = %q, %q; want none", ghHost, host, primary, fallback)
			}
		}
	}
}
