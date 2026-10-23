package executor

import (
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

func TestExplicitGitHubTokensAreScopedToTheirGitHubHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		env  map[string]string
		want bool
	}{
		{name: "public token for public host", host: "github.com", env: map[string]string{envGHToken: "public"}, want: true},
		{name: "enterprise token for enterprise host", host: "ghe.example", env: map[string]string{"GH_ENTERPRISE_TOKEN": "enterprise"}, want: true},
		{name: "public token does not bypass enterprise host", host: "ghe.example", env: map[string]string{envGHToken: "public"}},
		{name: "enterprise token does not bypass public host", host: "github.com", env: map[string]string{"GH_ENTERPRISE_TOKEN": "enterprise"}},
		{name: "github.com-bound token does not bypass GHE.com tenant", host: "octo.ghe.com", env: map[string]string{envGHToken: "public"}},
		{name: "tenant-bound token for its tenant", host: "octo.ghe.com", env: map[string]string{envGHToken: "tenant", "GH_HOST": "octo.ghe.com"}, want: true},
		{name: "tenant-bound fallback token for its tenant", host: "octo.ghe.com", env: map[string]string{envGitHubToken: "tenant", "GH_HOST": "octo.ghe.com"}, want: true},
		{name: "tenant-bound token does not bypass public host", host: "github.com", env: map[string]string{envGHToken: "tenant", "GH_HOST": "octo.ghe.com"}},
		{name: "tenant-bound token does not bypass another tenant", host: "other.ghe.com", env: map[string]string{envGHToken: "tenant", "GH_HOST": "octo.ghe.com"}},
		{name: "malformed GH_HOST binds no token", host: "github.com", env: map[string]string{envGHToken: "public", "GH_HOST": "https://github.com"}},
		{name: "enterprise token does not bypass GHE.com tenant", host: "octo.ghe.com", env: map[string]string{"GH_ENTERPRISE_TOKEN": "enterprise"}},
		{name: "enterprise fallback token for enterprise server", host: "ghe.example.com", env: map[string]string{"GITHUB_ENTERPRISE_TOKEN": "enterprise"}, want: true},
		{name: "server-bound enterprise token does not bypass another server", host: "ghe2.example.com",
			env: map[string]string{"GH_ENTERPRISE_TOKEN": "enterprise", "GH_HOST": "ghe.example.com"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasExplicitGitHubToken(test.env, nil, test.host); got != test.want {
				t.Fatalf("hasExplicitGitHubToken(env, %q) = %v, want %v", test.host, got, test.want)
			}
			profile := make([]models.ProfileEnvVar, 0, len(test.env))
			for key, value := range test.env {
				profile = append(profile, models.ProfileEnvVar{Key: key, Value: value})
			}
			if got := hasExplicitGitHubToken(nil, profile, test.host); got != test.want {
				t.Fatalf("hasExplicitGitHubToken(profile, %q) = %v, want %v", test.host, got, test.want)
			}
		})
	}
}

func TestExplicitGitHubTokenProfileGHHostOverridesRequestGHHost(t *testing.T) {
	env := map[string]string{envGHToken: "tenant", "GH_HOST": "github.com"}
	profile := []models.ProfileEnvVar{{Key: "GH_HOST", Value: "octo.ghe.com"}}
	if hasExplicitGitHubToken(env, profile, "github.com") {
		t.Fatal("request GH_HOST bound the token although the profile GH_HOST overrides it")
	}
	if !hasExplicitGitHubToken(env, profile, "octo.ghe.com") {
		t.Fatal("token is not bound to the profile GH_HOST")
	}
}

func TestExplicitGitHubTokenSecretBackedGHHostBindsNoHost(t *testing.T) {
	env := map[string]string{envGHToken: "token", "GH_HOST": "github.com"}
	profile := []models.ProfileEnvVar{{Key: "GH_HOST", SecretID: "secret-gh-host"}}
	for _, host := range []string{"github.com", "octo.ghe.com"} {
		if hasExplicitGitHubToken(env, profile, host) {
			t.Fatalf("token bound to %q although the profile GH_HOST is secret-backed and unresolved", host)
		}
	}
}
