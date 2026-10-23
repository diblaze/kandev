package github

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestCleanupHostsWorkspaceSweepResolvesEachWatchHost(t *testing.T) {
	fixture := newHostRoutingFixture(t)
	ctx := context.Background()
	store := fixture.service.store
	for i, host := range []string{defaultGitHubHost, enterpriseTestHost, unconnectedTestHost} {
		watch := &ReviewWatch{WorkspaceID: testWorkspaceID, WorkflowID: "wf", WorkflowStepID: "step",
			AgentProfileID: "agent", ExecutorProfileID: "exec", Enabled: true, GitHubHost: host}
		if err := store.CreateReviewWatch(ctx, watch); err != nil {
			t.Fatalf("create review watch: %v", err)
		}
		if err := store.CreateReviewPRTask(ctx, &ReviewPRTask{ReviewWatchID: watch.ID, RepoOwner: "octo",
			RepoName: "app", PRNumber: i + 1, PRURL: "https://" + host + "/octo/app/pull/1", TaskID: "task"}); err != nil {
			t.Fatalf("create review PR task: %v", err)
		}
	}

	_, err := fixture.service.CleanupReviewTasksForWorkspace(ctx, testWorkspaceID)

	var notConnected *HostNotConnectedError
	if !errors.As(err, &notConnected) || notConnected.Host != unconnectedTestHost {
		t.Fatalf("error = %v, want the unconnected host reported", err)
	}
	loaded := fixture.connections.loadedHosts()
	for _, host := range []string{defaultGitHubHost, enterpriseTestHost, unconnectedTestHost} {
		if !slices.Contains(loaded, host) {
			t.Fatalf("loaded hosts = %v, missing %s", loaded, host)
		}
	}
}
