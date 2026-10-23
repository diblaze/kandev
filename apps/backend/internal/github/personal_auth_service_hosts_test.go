package github

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPersonalAuthCompleteOnEnterpriseRegistrationKeepsGitHubComIdentity(t *testing.T) {
	store := newTestStore(t)
	seedConnectionWorkspaces(t, store, "ws-1")
	seedHostAppConnections(t, store)
	secrets := newFakeConnectionSecrets()
	repo := NewStorePersonalConnectionRepository(store, secrets)
	now := time.Now().UTC()
	public := &UserConnection{
		WorkspaceID: "ws-1", UserID: "user-1", GitHubHost: defaultGitHubHost, AppRegistrationID: "registration-public",
		GitHubUserID: 5, Login: "public-login", Status: ConnectionStatusActive,
		AccessExpiresAt: now.Add(time.Hour), CredentialGeneration: 1,
	}
	if err := repo.ReplacePersonalConnection(context.Background(), public, GitHubOAuthTokens{
		AccessToken: "public-access", RefreshToken: "public-refresh", AccessExpiresAt: public.AccessExpiresAt,
	}, 0); err != nil {
		t.Fatalf("seed github.com personal connection: %v", err)
	}
	flows := NewOAuthFlowManager(store)
	flows.random = strings.NewReader(strings.Repeat("q", oauthRandomBytes*2))
	service := NewPersonalAuthService(PersonalAuthConfig{
		RegistrationID: "registration-enterprise", Host: enterpriseTestHost, ClientID: "Iv1.client",
		CallbackURL: "https://kandev.example/personal/callback",
	}, flows, repo, &personalOAuthRemote{fakeGitHubAppOAuth: fakeGitHubAppOAuth{
		tokens: GitHubOAuthTokens{AccessToken: "enterprise-access", RefreshToken: "enterprise-refresh",
			AccessExpiresAt: now.Add(time.Hour)},
		user: GitHubOAuthUser{ID: 11, Login: "hubot"}, canAccess: true,
	}})
	started, err := service.Start(context.Background(), "ws-1", "user-1")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if _, err := service.Complete(context.Background(), PersonalAuthCallback{
		State: started.State, Code: "oauth-code",
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	enterprise, err := store.GetUserConnection(context.Background(), "ws-1", "user-1", enterpriseTestHost)
	if err != nil || enterprise == nil || enterprise.Login != "hubot" || enterprise.GitHubHost != enterpriseTestHost {
		t.Fatalf("enterprise personal connection = %+v, err %v", enterprise, err)
	}
	if secrets.values[UserAccessTokenSecretKey("ws-1", "user-1", enterpriseTestHost)] != "enterprise-access" ||
		secrets.values[UserAccessTokenSecretKey("ws-1", "user-1", defaultGitHubHost)] != "public-access" {
		t.Fatalf("personal secrets = %v", secrets.values)
	}
	kept, err := store.GetUserConnection(context.Background(), "ws-1", "user-1", defaultGitHubHost)
	if err != nil || kept == nil || kept.Login != "public-login" {
		t.Fatalf("github.com personal connection = %+v, err %v", kept, err)
	}
}
