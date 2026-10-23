package github

import (
	"context"
	"errors"
	"testing"
)

const enterprisePRURL = "https://" + enterpriseTestHost + "/octo/app/pull/3"

func newPRURLHostsFixture(t *testing.T) hostRoutingFixture {
	t.Helper()
	fixture := newHostRoutingFixture(t)
	enterprise := fixture.provider.clients[enterpriseTestHost].(*MockClient)
	enterprise.AddPR(&PR{
		Number: 3, Title: "Enterprise PR", State: "open", HeadSHA: "abc", HeadBranch: "feat/x",
		BaseBranch: "main", RepoOwner: "octo", RepoName: "app", HTMLURL: enterprisePRURL,
	})
	return fixture
}

func assertOnlyHostLoaded(t *testing.T, fixture hostRoutingFixture, want string) {
	t.Helper()
	for _, host := range fixture.connections.loadedHosts() {
		if host != want {
			t.Fatalf("resolved the %s connection, want only %s", host, want)
		}
	}
}

func TestPRURLHostsAssociateExistingPROnEnterpriseHost(t *testing.T) {
	fixture := newPRURLHostsFixture(t)

	tp, err := fixture.service.AssociateExistingPRByURLForWorkspace(
		context.Background(), testWorkspaceID, "", "task-1", "repo-1", enterprisePRURL)
	if err != nil {
		t.Fatalf("AssociateExistingPRByURLForWorkspace: %v", err)
	}
	if tp.GitHubHost != enterpriseTestHost || tp.PRURL != enterprisePRURL {
		t.Fatalf("task PR host = %q, URL = %q, want %s", tp.GitHubHost, tp.PRURL, enterpriseTestHost)
	}
	assertOnlyHostLoaded(t, fixture, enterpriseTestHost)
}

func TestPRURLHostsAssociateExistingPROnUnconnectedHostFails(t *testing.T) {
	fixture := newPRURLHostsFixture(t)

	_, err := fixture.service.AssociateExistingPRByURLForWorkspace(
		context.Background(), testWorkspaceID, "", "task-1", "repo-1",
		"https://"+unconnectedTestHost+"/octo/app/pull/3")

	var notConnected *HostNotConnectedError
	if !errors.As(err, &notConnected) || notConnected.Host != unconnectedTestHost {
		t.Fatalf("error = %v, want HostNotConnectedError for %s", err, unconnectedTestHost)
	}
}

func TestPRURLHostsAssociatePRByURLWatchesOnEnterpriseHost(t *testing.T) {
	fixture := newPRURLHostsFixture(t)
	ctx := context.Background()

	if err := fixture.service.AssociatePRByURLForWorkspace(
		ctx, testWorkspaceID, "", "session-1", "task-1", "repo-1", enterprisePRURL, "",
	); err != nil {
		t.Fatalf("AssociatePRByURLForWorkspace: %v", err)
	}
	watch, err := fixture.service.store.GetPRWatchBySessionAndRepo(ctx, "session-1", "repo-1")
	if err != nil || watch == nil || watch.GitHubHost != enterpriseTestHost {
		t.Fatalf("watch = %+v, %v, want host %s", watch, err, enterpriseTestHost)
	}
	tp, err := fixture.service.store.GetTaskPR(ctx, "task-1")
	if err != nil || tp == nil || tp.GitHubHost != enterpriseTestHost {
		t.Fatalf("task PR = %+v, %v, want host %s", tp, err, enterpriseTestHost)
	}
	assertOnlyHostLoaded(t, fixture, enterpriseTestHost)
}
