package github

import (
	"context"
	"testing"
)

func newTwoHostCLIDefaultsService(t *testing.T) *Service {
	t.Helper()
	service, _ := newWorkspaceConnectionService(t, "operator")
	logins := map[string]string{defaultGitHubHost: "octocat", enterpriseTestHost: "hubot"}
	service.tokenClientFactory = func(host, _ string) Client {
		client := NewMockClient()
		client.SetUser(logins[host])
		return client
	}
	service.resolver.ghToken = func(_ context.Context, host, _ string) (string, error) {
		return "gho_" + host, nil
	}
	service.ghAccountLister = func(context.Context) ([]GHAccount, error) {
		return []GHAccount{
			{Host: defaultGitHubHost, Login: "octocat", Active: true, State: "active"},
			{Host: enterpriseTestHost, Login: "other", Active: false, State: "active"},
			{Host: enterpriseTestHost, Login: "hubot", Active: true, State: "active"},
		}, nil
	}
	return service
}

func TestWorkspaceDefaultsHostsBindEveryActiveCLIHost(t *testing.T) {
	service := newTwoHostCLIDefaultsService(t)

	if err := service.InitializeWorkspaceDefaults(context.Background(), "ws-1"); err != nil {
		t.Fatalf("InitializeWorkspaceDefaults: %v", err)
	}

	connections, err := service.store.ListWorkspaceConnections(context.Background(), "ws-1")
	if err != nil || len(connections) != 2 ||
		connections[0].GitHubHost != defaultGitHubHost || connections[0].Login != "octocat" ||
		connections[1].GitHubHost != enterpriseTestHost || connections[1].Login != "hubot" ||
		connections[1].Source != ConnectionSourceGHCLI {
		t.Fatalf("connections = %+v, err %v", connections, err)
	}
}

func TestWorkspaceDefaultsHostsKeepExistingHostAndContinueAfterFailure(t *testing.T) {
	service := newTwoHostCLIDefaultsService(t)
	ctx := context.Background()
	if err := service.store.UpsertWorkspaceConnection(ctx, &WorkspaceConnection{
		WorkspaceID: "ws-1", Source: ConnectionSourcePAT, GitHubHost: enterpriseTestHost, Login: "existing",
		Status: ConnectionStatusActive, CredentialGeneration: 1,
	}); err != nil {
		t.Fatalf("seed enterprise connection: %v", err)
	}
	service.ghAccountLister = func(context.Context) ([]GHAccount, error) {
		return []GHAccount{
			{Host: "ghe.broken.example", Login: "broken", Active: true, State: "active"},
			{Host: enterpriseTestHost, Login: "hubot", Active: true, State: "active"},
			{Host: defaultGitHubHost, Login: "octocat", Active: true, State: "active"},
		}, nil
	}

	if err := service.InitializeWorkspaceDefaults(ctx, "ws-1"); err != nil {
		t.Fatalf("InitializeWorkspaceDefaults: %v", err)
	}

	connections, err := service.store.ListWorkspaceConnections(ctx, "ws-1")
	if err != nil || len(connections) != 2 ||
		connections[0].GitHubHost != defaultGitHubHost || connections[0].Source != ConnectionSourceGHCLI ||
		connections[1].GitHubHost != enterpriseTestHost || connections[1].Login != "existing" {
		t.Fatalf("connections = %+v, err %v", connections, err)
	}
}
