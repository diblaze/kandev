package github

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// hostRecordingTokenFactory validates every token as login and records the
// host each validation was sent to.
func hostRecordingTokenFactory(service *Service, login string) *[]string {
	var hosts []string
	service.tokenClientFactory = func(host, _ string) Client {
		hosts = append(hosts, host)
		client := NewMockClient()
		client.SetUser(login)
		return client
	}
	return &hosts
}

func connectWorkspacePAT(t *testing.T, service *Service, host, token string) *WorkspaceConnection {
	t.Helper()
	connection, err := service.SetWorkspaceConnection(context.Background(), "ws-1", SetWorkspaceConnectionRequest{
		Source: ConnectionSourcePAT, Host: host, Token: token,
	})
	if err != nil {
		t.Fatalf("connect %s PAT: %v", host, err)
	}
	return connection
}

func TestWorkspaceConnectionHostsPATValidatesAgainstRequestedHost(t *testing.T) {
	service, secrets := newWorkspaceConnectionService(t, "octocat")
	hosts := hostRecordingTokenFactory(service, "octocat")
	connectWorkspacePAT(t, service, "", "ghp_public")

	enterprise := connectWorkspacePAT(t, service, "Octo.GHE.com", "ghp_enterprise")

	if got := *hosts; len(got) != 2 || got[0] != defaultGitHubHost || got[1] != enterpriseTestHost {
		t.Fatalf("validation hosts = %v", got)
	}
	if enterprise.GitHubHost != enterpriseTestHost {
		t.Fatalf("enterprise connection host = %q", enterprise.GitHubHost)
	}
	public, err := service.store.GetWorkspaceConnection(context.Background(), "ws-1", defaultGitHubHost)
	if err != nil || public == nil || public.CredentialGeneration != 1 {
		t.Fatalf("github.com connection after enterprise save = %+v, err %v", public, err)
	}
	if secrets.values[WorkspacePATSecretKey("ws-1", defaultGitHubHost)] != "ghp_public" ||
		secrets.values[WorkspacePATSecretKey("ws-1", enterpriseTestHost)] != "ghp_enterprise" {
		t.Fatalf("stored PATs = %v", secrets.values)
	}
}

func TestWorkspaceConnectionHostsReplaceKeepsOtherHostGeneration(t *testing.T) {
	service, _ := newWorkspaceConnectionService(t, "octocat")
	connectWorkspacePAT(t, service, defaultGitHubHost, "ghp_public")
	connectWorkspacePAT(t, service, enterpriseTestHost, "ghp_enterprise")

	replaced := connectWorkspacePAT(t, service, enterpriseTestHost, "ghp_enterprise_2")

	public, err := service.store.GetWorkspaceConnection(context.Background(), "ws-1", defaultGitHubHost)
	if err != nil || public == nil || public.CredentialGeneration != 1 || replaced.CredentialGeneration != 2 {
		t.Fatalf("after replace: github.com %+v (err %v), enterprise %+v", public, err, replaced)
	}
}

func TestWorkspaceConnectionHostsDeleteKeepsOtherHost(t *testing.T) {
	service, secrets := newWorkspaceConnectionService(t, "octocat")
	connectWorkspacePAT(t, service, defaultGitHubHost, "ghp_public")
	connectWorkspacePAT(t, service, enterpriseTestHost, "ghp_enterprise")

	if err := service.DeleteWorkspaceConnection(context.Background(), "ws-1", enterpriseTestHost); err != nil {
		t.Fatalf("delete enterprise connection: %v", err)
	}

	hosts, err := service.WorkspaceGitHubHosts(context.Background(), "ws-1")
	if err != nil || len(hosts) != 1 || hosts[0] != defaultGitHubHost {
		t.Fatalf("hosts after delete = %v, err %v", hosts, err)
	}
	if _, ok := secrets.values[WorkspacePATSecretKey("ws-1", enterpriseTestHost)]; ok {
		t.Fatal("enterprise PAT survived its connection")
	}
	if secrets.values[WorkspacePATSecretKey("ws-1", defaultGitHubHost)] != "ghp_public" {
		t.Fatalf("github.com PAT after enterprise delete = %v", secrets.values)
	}
}

func TestWorkspaceConnectionHostsRejectInvalidHostBeforeTokenResolution(t *testing.T) {
	service, _ := newWorkspaceConnectionService(t, "octocat")
	hosts := hostRecordingTokenFactory(service, "octocat")
	ghCalls := 0
	service.resolver.ghToken = func(context.Context, string, string) (string, error) {
		ghCalls++
		return "gho_token", nil
	}

	for _, request := range []SetWorkspaceConnectionRequest{
		{Source: ConnectionSourceGHCLI, Host: "--hostname=evil.example", Login: "octocat"},
		{Source: ConnectionSourcePAT, Host: "https://octo.ghe.com", Token: "ghp_token"},
	} {
		if _, err := service.SetWorkspaceConnection(context.Background(), "ws-1", request); err == nil {
			t.Fatalf("SetWorkspaceConnection(%+v) accepted an invalid host", request)
		}
	}
	if ghCalls != 0 || len(*hosts) != 0 {
		t.Fatalf("invalid hosts reached token resolution: gh %d, validations %v", ghCalls, *hosts)
	}
}

func TestWorkspaceConnectionHostsFailedValidationKeepsPreviousRow(t *testing.T) {
	service, _ := newWorkspaceConnectionService(t, "octocat")
	previous := connectWorkspacePAT(t, service, enterpriseTestHost, "ghp_enterprise")
	service.tokenClientFactory = func(_, _ string) Client { return &NoopClient{} }

	_, err := service.SetWorkspaceConnection(context.Background(), "ws-1", SetWorkspaceConnectionRequest{
		Source: ConnectionSourcePAT, Host: enterpriseTestHost, Token: "ghp_revoked",
	})

	if !errors.Is(err, ErrInvalidToken) || !strings.Contains(err.Error(), enterpriseTestHost) {
		t.Fatalf("failed validation error = %v, want ErrInvalidToken naming %s", err, enterpriseTestHost)
	}
	current, getErr := service.store.GetWorkspaceConnection(context.Background(), "ws-1", enterpriseTestHost)
	if getErr != nil || current == nil || current.CredentialGeneration != previous.CredentialGeneration {
		t.Fatalf("connection after failed validation = %+v, err %v", current, getErr)
	}
}

func TestWorkspacePATSecretKeyKeepsGitHubComKey(t *testing.T) {
	if got := WorkspacePATSecretKey("ws-1", defaultGitHubHost); got != "github:workspace:ws-1:pat" {
		t.Fatalf("github.com PAT key = %q", got)
	}
	if got := WorkspacePATSecretKey("ws-1", enterpriseTestHost); got != "github:workspace:ws-1:pat:"+enterpriseTestHost {
		t.Fatalf("enterprise PAT key = %q", got)
	}
}

func TestWorkspaceConnectionFingerprintCoversEveryHost(t *testing.T) {
	service, _ := newWorkspaceConnectionService(t, "octocat")
	connectWorkspacePAT(t, service, defaultGitHubHost, "ghp_public")
	before, err := service.WorkspaceConnectionFingerprint(context.Background(), "ws-1")
	if err != nil || before == "" {
		t.Fatalf("fingerprint = %q, err %v", before, err)
	}

	connectWorkspacePAT(t, service, enterpriseTestHost, "ghp_enterprise")

	after, err := service.WorkspaceConnectionFingerprint(context.Background(), "ws-1")
	if err != nil || after == before {
		t.Fatalf("fingerprint unchanged by an enterprise connection: %q, err %v", after, err)
	}
}

func TestWorkspaceConnectionHostsNotifyObserverAfterCommit(t *testing.T) {
	service, _ := newWorkspaceConnectionService(t, "octocat")
	var notified []string
	service.SetHostConnectedObserver(func(_ context.Context, workspaceID, host string) {
		notified = append(notified, workspaceID+"|"+host)
	})
	connectWorkspacePAT(t, service, enterpriseTestHost, "ghp_enterprise")
	service.tokenClientFactory = func(_, _ string) Client { return &NoopClient{} }

	_, err := service.SetWorkspaceConnection(context.Background(), "ws-1", SetWorkspaceConnectionRequest{
		Source: ConnectionSourcePAT, Host: defaultGitHubHost, Token: "ghp_invalid",
	})

	if err == nil || len(notified) != 1 || notified[0] != "ws-1|"+enterpriseTestHost {
		t.Fatalf("notified = %v, err %v", notified, err)
	}
}

func TestAppInstallationCommitNotifiesHostObserver(t *testing.T) {
	service, _ := newWorkspaceConnectionService(t, "octocat")
	enterprise := newAppRegistration("registration-enterprise", 777, "Enterprise", time.Now().UTC())
	enterprise.GitHubHost = enterpriseTestHost
	if err := service.store.UpsertDeploymentAppRegistration(context.Background(), enterprise); err != nil {
		t.Fatalf("seed registration: %v", err)
	}
	var notified []string
	service.SetHostConnectedObserver(func(_ context.Context, workspaceID, host string) {
		notified = append(notified, workspaceID+"|"+host)
	})
	installationID := int64(42)

	err := (&serviceAppConnectionStore{service: service}).ReplaceWorkspaceConnection(context.Background(), &WorkspaceConnection{
		WorkspaceID: "ws-1", Source: ConnectionSourceGitHubAppInstallation, GitHubHost: enterpriseTestHost,
		InstallationID: &installationID, InstallationAccountLogin: "acme", InstallationAccountType: "Organization",
		AppRegistrationID: enterprise.ID, Status: ConnectionStatusActive, CredentialGeneration: 1,
	}, WorkspaceConnectionExpectation{})

	if err != nil || len(notified) != 1 || notified[0] != "ws-1|"+enterpriseTestHost {
		t.Fatalf("notified = %v, err %v", notified, err)
	}
}
