package github

import (
	"context"
	"testing"
)

func newTwoHostStatusService(t *testing.T) *Service {
	t.Helper()
	service, _ := newWorkspaceConnectionService(t, "octocat")
	connectWorkspacePAT(t, service, defaultGitHubHost, "ghp_public")
	hostRecordingTokenFactory(service, "hubot")
	connectWorkspacePAT(t, service, enterpriseTestHost, "ghp_enterprise")
	return service
}

func TestWorkspaceAuthStatusListsEveryHost(t *testing.T) {
	service := newTwoHostStatusService(t)

	status, err := service.GetWorkspaceAuthStatus(context.Background(), "ws-1", DefaultUserID, "")

	if err != nil {
		t.Fatalf("GetWorkspaceAuthStatus: %v", err)
	}
	if status.Automation == nil || status.Automation.GitHubHost != defaultGitHubHost || status.Username != "octocat" {
		t.Fatalf("default projection = %+v", status.Automation)
	}
	if len(status.Hosts) != 2 ||
		status.Hosts[0].GitHubHost != defaultGitHubHost || status.Hosts[0].Login != "octocat" ||
		status.Hosts[1].GitHubHost != enterpriseTestHost || status.Hosts[1].Login != "hubot" ||
		status.Hosts[1].Source != ConnectionSourcePAT || status.Hosts[1].Status != ConnectionStatusActive {
		t.Fatalf("hosts = %+v", status.Hosts)
	}
}

func TestWorkspaceAuthStatusProjectsRequestedHost(t *testing.T) {
	service := newTwoHostStatusService(t)

	status, err := service.GetWorkspaceAuthStatus(context.Background(), "ws-1", DefaultUserID, enterpriseTestHost)

	if err != nil {
		t.Fatalf("GetWorkspaceAuthStatus: %v", err)
	}
	if status.Automation == nil || status.Automation.GitHubHost != enterpriseTestHost ||
		!status.Authenticated || status.Username != "hubot" || len(status.Hosts) != 2 {
		t.Fatalf("enterprise projection = %+v (automation %+v)", status, status.Automation)
	}
}

func TestWorkspaceAuthStatusRejectsInvalidHost(t *testing.T) {
	service := newTwoHostStatusService(t)

	if _, err := service.GetWorkspaceAuthStatus(context.Background(), "ws-1", DefaultUserID, "octo.ghe.com:8443"); err == nil {
		t.Fatal("status accepted a host with a port")
	}
}
