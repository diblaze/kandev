package github

import (
	"context"
	"testing"
	"time"
)

func TestPRFeedbackHostsPersistOnlyToTaskPRsOfTheFetchedHost(t *testing.T) {
	_, svc, mockClient, store := setupPollerTest(t)
	ctx := context.Background()
	seedTask(t, store, "task-public", false)
	seedTask(t, store, "task-enterprise", false)
	seedFeedbackTaskPR(t, store, "task-public", 2408, nil)
	seedFeedbackTaskPR(t, store, "task-enterprise", 2408, func(tp *TaskPR) { tp.GitHubHost = enterpriseTestHost })
	addMergedFeedbackPR(mockClient, 2408, time.Now().UTC().Add(-10*time.Minute))

	if _, err := svc.GetPRFeedbackForWorkspace(
		ctx, testWorkspaceID, "", defaultGitHubHost, feedbackSyncOwner, feedbackSyncRepo, 2408,
	); err != nil {
		t.Fatalf("GetPRFeedbackForWorkspace: %v", err)
	}

	public, err := store.GetTaskPRByRepoAndNumber(ctx, "task-public", "repo-1", 2408)
	if err != nil || public == nil || public.State != prStateMerged {
		t.Fatalf("github.com task PR = %+v, %v; want merged", public, err)
	}
	enterprise, err := store.GetTaskPRByRepoAndNumber(ctx, "task-enterprise", "repo-1", 2408)
	if err != nil || enterprise == nil || enterprise.State != prStateOpen {
		t.Fatalf("%s task PR = %+v, %v; want untouched open", enterpriseTestHost, enterprise, err)
	}
}
