package github

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func newTwoHostBatchedService(t *testing.T) (*Service, *Store, map[string]*graphQLMockClient) {
	t.Helper()
	_, svc, _, store := setupPollerTest(t)
	clients := map[string]*graphQLMockClient{}
	providerClients := map[string]Client{}
	connections := map[string]*WorkspaceConnection{}
	for _, host := range []string{defaultGitHubHost, enterpriseTestHost} {
		client := &graphQLMockClient{MockClient: NewMockClient()}
		client.SetRepositoryDetails(GitHubRepository{FullName: "o/r", Owner: "o", Name: "r"})
		clients[host] = client
		providerClients[host] = client
		connections[host] = &WorkspaceConnection{WorkspaceID: testWorkspaceID, Source: ConnectionSourcePAT,
			GitHubHost: host, Login: "login", Status: ConnectionStatusActive, CredentialGeneration: 1}
	}
	svc.resolver = NewCredentialResolver(&hostRecordingConnections{connections: connections}, nil)
	svc.resolver.SetAutomationProvider(&hostClientProvider{clients: providerClients})
	return svc, store, clients
}

func seedHostSearchingWatches(t *testing.T, store *Store, host string, n int) []*PRWatch {
	t.Helper()
	watches := make([]*PRWatch, 0, n)
	for i := range n {
		taskID := fmt.Sprintf("task-%s-%d", host, i)
		seedTask(t, store, taskID, false)
		watch := withTestWorkspace(&PRWatch{SessionID: "session-" + taskID, TaskID: taskID, GitHubHost: host,
			Owner: "o", Repo: "r", Branch: fmt.Sprintf("feature/%d", i)})
		if err := store.CreatePRWatch(context.Background(), watch); err != nil {
			t.Fatalf("create PR watch: %v", err)
		}
		watches = append(watches, watch)
	}
	return watches
}

func TestPRWatchBatchedHostsRunOneBatchPerHost(t *testing.T) {
	svc, store, clients := newTwoHostBatchedService(t)
	public := seedHostSearchingWatches(t, store, defaultGitHubHost, 2)
	enterprise := seedHostSearchingWatches(t, store, enterpriseTestHost, 3)
	clients[defaultGitHubHost].branchResponses = []string{branchAliasResponse(t, len(public), nil)}
	clients[enterpriseTestHost].branchResponses = []string{branchAliasResponse(t, len(enterprise), nil)}
	interleaved := []*PRWatch{public[0], enterprise[0], public[1], enterprise[1], enterprise[2]}

	results, err := svc.SyncWorkspaceWatchesBatched(context.Background(), testWorkspaceID, interleaved)

	if err != nil {
		t.Fatalf("batched sync: %v", err)
	}
	if len(results) != len(interleaved) {
		t.Fatalf("results = %d, want %d", len(results), len(interleaved))
	}
	for host, want := range map[string]int{defaultGitHubHost: len(public), enterpriseTestHost: len(enterprise)} {
		client := clients[host]
		if len(client.branchQueries) != 1 {
			t.Fatalf("%s branch queries = %d, want 1", host, len(client.branchQueries))
		}
		if got := strings.Count(client.branchQueries[0], "pullRequests("); got != want {
			t.Fatalf("%s batch aliases = %d, want %d", host, got, want)
		}
		if client.statusCalls != 0 || len(client.prQueries) != 0 {
			t.Fatalf("%s extra calls: status %d, PR queries %d", host, client.statusCalls, len(client.prQueries))
		}
	}
}

func TestPRWatchBatchedHostsSkipUnconnectedHostAndSyncTheRest(t *testing.T) {
	svc, store, clients := newTwoHostBatchedService(t)
	public := seedHostSearchingWatches(t, store, defaultGitHubHost, 1)
	unconnected := seedHostSearchingWatches(t, store, unconnectedTestHost, 1)
	clients[defaultGitHubHost].branchResponses = []string{branchAliasResponse(t, 1, nil)}

	results, err := svc.SyncWorkspaceWatchesBatched(context.Background(), testWorkspaceID, append(public, unconnected...))

	if err != nil {
		t.Fatalf("batched sync: %v", err)
	}
	if len(results) != 1 || results[0].Watch.ID != public[0].ID {
		t.Fatalf("results = %+v, want only the github.com watch", results)
	}
	if len(clients[enterpriseTestHost].branchQueries) != 0 {
		t.Fatal("unconnected host watches reached another host's client")
	}
}
