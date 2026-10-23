package github

import (
	"context"
	"errors"
	"testing"
)

func TestResolveGitCredentialRoutesByHost(t *testing.T) {
	t.Parallel()

	connections := &fakeConnectionReader{hostWorkspaces: map[string]*WorkspaceConnection{
		"workspace-a|github.com": {
			WorkspaceID: "workspace-a", GitHubHost: defaultGitHubHost, Source: ConnectionSourcePAT,
			Status: ConnectionStatusActive, CredentialGeneration: 1,
		},
		"workspace-a|octo.ghe.com": {
			WorkspaceID: "workspace-a", GitHubHost: "octo.ghe.com", Source: ConnectionSourcePAT,
			Status: ConnectionStatusActive, CredentialGeneration: 1,
		},
	}}
	service := &Service{resolver: NewCredentialResolver(connections, fakeAuthSecrets{
		WorkspacePATSecretKey("workspace-a", defaultGitHubHost): "dotcom-token",
		WorkspacePATSecretKey("workspace-a", "octo.ghe.com"):    "tenant-token",
	})}

	for host, want := range map[string]string{"github.com": "dotcom-token", "octo.ghe.com": "tenant-token"} {
		_, password, err := service.ResolveGitCredential(context.Background(), "workspace-a", "github", host, "org", "repo")
		if err != nil || password != want {
			t.Fatalf("ResolveGitCredential(%s) = %q, %v; want %q", host, password, err, want)
		}
	}
	_, _, err := service.ResolveGitCredential(context.Background(), "workspace-a", "github", "ghe.example.com", "org", "repo")
	var notConnected *HostNotConnectedError
	if !errors.As(err, &notConnected) {
		t.Fatalf("unconnected host error = %v, want HostNotConnectedError", err)
	}
}
