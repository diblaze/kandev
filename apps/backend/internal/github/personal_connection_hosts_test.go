package github

import (
	"context"
	"testing"
	"time"
)

func newTwoHostPersonalService(t *testing.T) *Service {
	t.Helper()
	service, _ := newWorkspaceConnectionService(t, "octocat")
	seedHostAppConnections(t, service.store)
	for host, registrationID := range map[string]string{
		defaultGitHubHost: "registration-public", enterpriseTestHost: "registration-enterprise",
	} {
		connection := &UserConnection{
			WorkspaceID: "ws-1", UserID: DefaultUserID, GitHubHost: host, AppRegistrationID: registrationID,
			GitHubUserID: 7, Login: "login-" + host, Status: ConnectionStatusActive,
			AccessExpiresAt: time.Now().UTC().Add(time.Hour), CredentialGeneration: 1,
		}
		if err := service.personalConnections.ReplacePersonalConnection(context.Background(), connection, GitHubOAuthTokens{
			AccessToken: "access-" + host, RefreshToken: "refresh-" + host, AccessExpiresAt: connection.AccessExpiresAt,
		}, 0); err != nil {
			t.Fatalf("seed %s personal connection: %v", host, err)
		}
	}
	return service
}

func assertPersonalConnectionHosts(t *testing.T, service *Service, want map[string]bool) {
	t.Helper()
	for host, present := range want {
		connection, err := service.store.GetUserConnection(context.Background(), "ws-1", DefaultUserID, host)
		if err != nil || (connection != nil) != present {
			t.Fatalf("%s personal connection = %+v, err %v; want present %v", host, connection, err, present)
		}
	}
}

func TestDisconnectPersonalAuthRemovesOnlyRequestedHost(t *testing.T) {
	service := newTwoHostPersonalService(t)

	if err := service.DisconnectPersonalAuth(context.Background(), "ws-1", DefaultUserID, enterpriseTestHost); err != nil {
		t.Fatalf("disconnect enterprise personal connection: %v", err)
	}

	assertPersonalConnectionHosts(t, service, map[string]bool{defaultGitHubHost: true, enterpriseTestHost: false})
}

func TestReplacingHostAppConnectionRevokesOnlyThatHostsPersonalConnections(t *testing.T) {
	service := newTwoHostPersonalService(t)

	connectWorkspacePAT(t, service, enterpriseTestHost, "ghp_enterprise")

	assertPersonalConnectionHosts(t, service, map[string]bool{defaultGitHubHost: true, enterpriseTestHost: false})
}
