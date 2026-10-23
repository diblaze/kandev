package github

import (
	"context"
	"errors"
	"testing"
	"time"
)

type countingAuthSecrets struct {
	values  map[string]string
	reveals map[string]int
}

func (s *countingAuthSecrets) Reveal(_ context.Context, id string) (string, error) {
	s.reveals[id]++
	value, ok := s.values[id]
	if !ok {
		return "", errors.New("secret not found")
	}
	return value, nil
}

func newHostPATResolver() (*CredentialResolver, *countingAuthSecrets) {
	connections := &fakeConnectionReader{
		workspaces: map[string]*WorkspaceConnection{
			"ws-1": {WorkspaceID: "ws-1", Source: ConnectionSourcePAT, GitHubHost: defaultGitHubHost,
				Login: "octocat", Status: ConnectionStatusActive, CredentialGeneration: 1},
		},
		hostWorkspaces: map[string]*WorkspaceConnection{
			"ws-1|" + enterpriseTestHost: {WorkspaceID: "ws-1", Source: ConnectionSourcePAT, GitHubHost: enterpriseTestHost,
				Login: "hubot", Status: ConnectionStatusActive, CredentialGeneration: 7},
		},
	}
	secrets := &countingAuthSecrets{reveals: map[string]int{}, values: map[string]string{
		WorkspacePATSecretKey("ws-1", defaultGitHubHost):  "ghp_public",
		WorkspacePATSecretKey("ws-1", enterpriseTestHost): "ghp_enterprise",
	}}
	return NewCredentialResolver(connections, secrets), secrets
}

func TestResolverHostsRouteToConnectionOfRequestedHost(t *testing.T) {
	resolver, _ := newHostPATResolver()

	resolved, err := resolver.Resolve(context.Background(), ResolveCredentialRequest{
		WorkspaceID: "ws-1", Host: enterpriseTestHost, Purpose: CredentialPurposeAutomation,
	})

	if err != nil {
		t.Fatalf("resolve enterprise host: %v", err)
	}
	client, ok := resolved.Client.(*TokenClient)
	if !ok || client.endpoints.REST != "https://api."+enterpriseTestHost || resolved.credential != "ghp_enterprise" {
		t.Fatalf("enterprise client = %#v, credential %q", resolved.Client, resolved.credential)
	}
	if resolved.Principal.Login != "hubot" || resolved.Principal.GitHubHost != enterpriseTestHost {
		t.Fatalf("enterprise principal = %+v", resolved.Principal)
	}
}

func TestResolverHostsReportUnconnectedHostWithoutTokenSource(t *testing.T) {
	resolver, secrets := newHostPATResolver()
	resolver.ghToken = func(context.Context, string, string) (string, error) {
		t.Fatal("unconnected host reached the gh token source")
		return "", nil
	}

	_, err := resolver.Resolve(context.Background(), ResolveCredentialRequest{
		WorkspaceID: "ws-1", Host: "ghe.example.com", Purpose: CredentialPurposeAutomation,
	})

	var notConnected *HostNotConnectedError
	if !errors.As(err, &notConnected) || notConnected.Host != "ghe.example.com" || !errors.Is(err, ErrGitHubNotConfigured) {
		t.Fatalf("unconnected host error = %v", err)
	}
	if len(secrets.reveals) != 0 {
		t.Fatalf("unconnected host revealed secrets %v", secrets.reveals)
	}
}

func TestResolverHostsRejectInvalidHost(t *testing.T) {
	resolver, secrets := newHostPATResolver()

	_, err := resolver.Resolve(context.Background(), ResolveCredentialRequest{
		WorkspaceID: "ws-1", Host: "octo.ghe.com/evil", Purpose: CredentialPurposeAutomation,
	})

	if err == nil || len(secrets.reveals) != 0 {
		t.Fatalf("invalid host: err %v, reveals %v", err, secrets.reveals)
	}
}

func TestResolverHostsCacheEachHostSeparately(t *testing.T) {
	resolver, secrets := newHostPATResolver()
	resolve := func(host string) *ResolvedCredential {
		t.Helper()
		resolved, err := resolver.Resolve(context.Background(), ResolveCredentialRequest{
			WorkspaceID: "ws-1", Host: host, Purpose: CredentialPurposeAutomation,
		})
		if err != nil {
			t.Fatalf("resolve %s: %v", host, err)
		}
		return resolved
	}

	resolve(defaultGitHubHost)
	resolve(enterpriseTestHost)
	public := resolve(defaultGitHubHost)

	if public.credential != "ghp_public" || secrets.reveals[WorkspacePATSecretKey("ws-1", defaultGitHubHost)] != 1 {
		t.Fatalf("github.com credential %q after enterprise resolution, reveals %v", public.credential, secrets.reveals)
	}
}

func TestInstallationCredentialProviderTargetsConnectionHost(t *testing.T) {
	installationID := int64(42)
	provider := NewCachedInstallationCredentialProvider(NewInstallationTokenCache(fixedInstallationMinter{
		token: InstallationToken{
			Token: "installation-token", ExpiresAt: time.Now().Add(time.Hour),
			Principal: TokenPrincipal{Kind: TokenCredentialInstallation},
		},
	}))

	resolved, err := provider.ResolveInstallation(context.Background(), &WorkspaceConnection{
		Source: ConnectionSourceGitHubAppInstallation, GitHubHost: "ghe.example.com",
		InstallationID: &installationID, InstallationAccountLogin: "acme",
	}, ResolveCredentialRequest{WorkspaceID: "workspace-1", Purpose: CredentialPurposeAutomation})

	if err != nil {
		t.Fatalf("ResolveInstallation: %v", err)
	}
	client, ok := resolved.Client.(*TokenClient)
	if !ok || client.endpoints.REST != "https://ghe.example.com/api/v3" {
		t.Fatalf("installation client = %#v", resolved.Client)
	}
}
