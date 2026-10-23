package github

import (
	"context"
	"errors"
	"testing"
	"time"
)

const enterpriseTestHost = "octo.ghe.com"

func seedHostConnections(t *testing.T, store *Store) {
	t.Helper()
	ctx := context.Background()
	for _, connection := range []*WorkspaceConnection{
		{WorkspaceID: "ws-1", Source: ConnectionSourcePAT, GitHubHost: enterpriseTestHost, Login: "hubot",
			Status: ConnectionStatusActive, CredentialGeneration: 4},
		{WorkspaceID: "ws-1", Source: ConnectionSourceGHCLI, GitHubHost: defaultGitHubHost, Login: "octocat",
			Status: ConnectionStatusActive, CredentialGeneration: 2},
	} {
		if err := store.UpsertWorkspaceConnection(ctx, connection); err != nil {
			t.Fatalf("upsert %s connection: %v", connection.GitHubHost, err)
		}
	}
}

func TestStoreWorkspaceConnectionHostsCoexist(t *testing.T) {
	store := newTestStore(t)
	seedConnectionWorkspaces(t, store, "ws-1")
	seedHostConnections(t, store)
	ctx := context.Background()

	enterprise, err := store.GetWorkspaceConnection(ctx, "ws-1", enterpriseTestHost)
	if err != nil || enterprise == nil || enterprise.Login != "hubot" || enterprise.CredentialGeneration != 4 {
		t.Fatalf("enterprise connection = %+v, err %v", enterprise, err)
	}
	public, err := store.GetWorkspaceConnection(ctx, "ws-1", defaultGitHubHost)
	if err != nil || public == nil || public.Login != "octocat" {
		t.Fatalf("github.com connection = %+v, err %v", public, err)
	}
	missing, err := store.GetWorkspaceConnection(ctx, "ws-1", "ghe.example.com")
	if err != nil || missing != nil {
		t.Fatalf("unconnected host = %+v, err %v", missing, err)
	}
	listed, err := store.ListWorkspaceConnections(ctx, "ws-1")
	if err != nil || len(listed) != 2 ||
		listed[0].GitHubHost != defaultGitHubHost || listed[1].GitHubHost != enterpriseTestHost {
		t.Fatalf("listed connections = %+v, err %v", listed, err)
	}
}

func TestStoreDeleteWorkspaceConnectionKeepsOtherHost(t *testing.T) {
	store := newTestStore(t)
	seedConnectionWorkspaces(t, store, "ws-1")
	seedHostConnections(t, store)
	ctx := context.Background()

	if err := store.DeleteWorkspaceConnection(ctx, "ws-1", defaultGitHubHost); err != nil {
		t.Fatalf("delete github.com connection: %v", err)
	}

	listed, err := store.ListWorkspaceConnections(ctx, "ws-1")
	if err != nil || len(listed) != 1 || listed[0].GitHubHost != enterpriseTestHost ||
		listed[0].CredentialGeneration != 4 {
		t.Fatalf("connections after delete = %+v, err %v", listed, err)
	}
}

func seedHostAppConnections(t *testing.T, store *Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	enterprise := newAppRegistration("registration-enterprise", 777, "Enterprise", now)
	enterprise.GitHubHost = enterpriseTestHost
	for _, registration := range []*AppRegistration{newAppRegistration("registration-public", 776, "Public", now), enterprise} {
		if err := store.UpsertDeploymentAppRegistration(ctx, registration); err != nil {
			t.Fatalf("seed registration %s: %v", registration.ID, err)
		}
	}
	installationID := int64(42)
	for host, registrationID := range map[string]string{
		defaultGitHubHost: "registration-public", enterpriseTestHost: "registration-enterprise",
	} {
		if err := store.UpsertWorkspaceConnection(ctx, &WorkspaceConnection{
			WorkspaceID: "ws-1", Source: ConnectionSourceGitHubAppInstallation, GitHubHost: host,
			InstallationID: &installationID, InstallationAccountLogin: "acme", InstallationAccountType: "Organization",
			AppRegistrationID: registrationID, Status: ConnectionStatusActive,
		}); err != nil {
			t.Fatalf("seed %s App connection: %v", host, err)
		}
	}
}

func TestStorePersonalConnectionsAreKeyedByHost(t *testing.T) {
	store := newTestStore(t)
	seedConnectionWorkspaces(t, store, "ws-1")
	seedHostAppConnections(t, store)
	ctx := context.Background()
	for host, registrationID := range map[string]string{
		defaultGitHubHost: "registration-public", enterpriseTestHost: "registration-enterprise",
	} {
		if err := store.UpsertUserConnection(ctx, &UserConnection{
			WorkspaceID: "ws-1", UserID: "user-1", GitHubHost: host, AppRegistrationID: registrationID,
			GitHubUserID: 7, Login: "login-" + host, Status: ConnectionStatusActive,
			AccessExpiresAt: time.Now().UTC().Add(time.Hour), CredentialGeneration: 3,
		}); err != nil {
			t.Fatalf("upsert %s personal connection: %v", host, err)
		}
	}

	if err := store.DeleteUserConnection(ctx, "ws-1", "user-1", defaultGitHubHost); err != nil {
		t.Fatalf("delete github.com personal connection: %v", err)
	}

	enterprise, err := store.GetUserConnection(ctx, "ws-1", "user-1", enterpriseTestHost)
	if err != nil || enterprise == nil || enterprise.Login != "login-"+enterpriseTestHost ||
		enterprise.GitHubHost != enterpriseTestHost {
		t.Fatalf("enterprise personal connection = %+v, err %v", enterprise, err)
	}
	public, err := store.GetUserConnection(ctx, "ws-1", "user-1", defaultGitHubHost)
	if err != nil || public != nil {
		t.Fatalf("deleted github.com personal connection = %+v, err %v", public, err)
	}
	for host, want := range map[string]int64{defaultGitHubHost: 4, enterpriseTestHost: 3} {
		generation, err := store.GetUserConnectionGeneration(ctx, "ws-1", "user-1", host)
		if err != nil || generation != want {
			t.Fatalf("%s personal generation = %d, err %v; want %d", host, generation, err, want)
		}
	}
}

func TestStorePersonalConnectionRequiresAppConnectionOnSameHost(t *testing.T) {
	store := newTestStore(t)
	seedConnectionWorkspaces(t, store, "ws-1")
	seedHostAppConnections(t, store)

	err := store.UpsertUserConnection(context.Background(), &UserConnection{
		WorkspaceID: "ws-1", UserID: "user-1", GitHubHost: enterpriseTestHost, AppRegistrationID: "registration-public",
		GitHubUserID: 7, Login: "octocat", Status: ConnectionStatusActive,
		AccessExpiresAt: time.Now().UTC().Add(time.Hour),
	})

	if !errors.Is(err, ErrOAuthFlowStale) {
		t.Fatalf("personal connection with another host's registration: err = %v, want %v", err, ErrOAuthFlowStale)
	}
}
