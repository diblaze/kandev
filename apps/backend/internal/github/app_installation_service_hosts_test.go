package github

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAppInstallationCompleteConnectsRegistrationHost(t *testing.T) {
	flows := NewOAuthFlowManager(&oauthFlowMemoryStore{})
	flows.random = strings.NewReader(strings.Repeat("h", oauthRandomBytes))
	store := &appInstallationMemoryStore{}
	service := NewAppInstallationService(
		AppInstallationConfig{
			RegistrationID: "registration-enterprise", Host: enterpriseTestHost, Slug: "kandev-app",
			CallbackURL: "https://kandev.example/callback",
		},
		flows, store,
		&fakeAppInstallationVerifier{installation: AppInstallation{ID: 42, AccountLogin: "acme", AccountType: "Organization"}},
		&fakeGitHubAppOAuth{
			tokens: GitHubOAuthTokens{AccessToken: "temporary-user-token"},
			user:   GitHubOAuthUser{ID: 11, Login: "octocat"}, canAccess: true,
		},
	)
	service.installationAccessRetryDelays = []time.Duration{0}
	started, err := service.Start(context.Background(), "workspace-1", "user-1")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if parsed, _ := url.Parse(started.URL); parsed == nil || parsed.Host != enterpriseTestHost {
		t.Fatalf("install URL = %q, want host %s", started.URL, enterpriseTestHost)
	}

	if _, err := service.Complete(context.Background(), AppInstallationCallback{
		State: started.State, Code: "oauth-code", InstallationID: 42,
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if store.connection == nil || store.connection.GitHubHost != enterpriseTestHost {
		t.Fatalf("persisted connection = %+v, want host %s", store.connection, enterpriseTestHost)
	}
}
