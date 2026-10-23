package github

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestWatchHostErrorsRecordUnconnectedHostAndKeepConfiguration(t *testing.T) {
	fixture := newHostRoutingFixture(t)
	ctx := context.Background()
	store := fixture.service.store
	review := &ReviewWatch{WorkspaceID: testWorkspaceID, WorkflowID: "wf", WorkflowStepID: "step",
		AgentProfileID: "agent", ExecutorProfileID: "exec", Enabled: true, GitHubHost: unconnectedTestHost}
	issue := &IssueWatch{WorkspaceID: testWorkspaceID, WorkflowID: "wf", WorkflowStepID: "step",
		AgentProfileID: "agent", ExecutorProfileID: "exec", Enabled: true, GitHubHost: unconnectedTestHost}
	if err := store.CreateReviewWatch(ctx, review); err != nil {
		t.Fatalf("create review watch: %v", err)
	}
	if err := store.CreateIssueWatch(ctx, issue); err != nil {
		t.Fatalf("create issue watch: %v", err)
	}

	_, reviewErr := fixture.service.CheckReviewWatch(ctx, review)
	_, issueErr := fixture.service.CheckIssueWatch(ctx, issue)

	var notConnected *HostNotConnectedError
	if !errors.As(reviewErr, &notConnected) || !errors.As(issueErr, &notConnected) {
		t.Fatalf("errors = %v, %v; want host not connected", reviewErr, issueErr)
	}
	storedReview, _ := store.GetReviewWatch(ctx, review.ID)
	storedIssue, _ := store.GetIssueWatch(ctx, issue.ID)
	for name, got := range map[string]struct {
		enabled   bool
		lastError string
		host      string
	}{
		"review": {storedReview.Enabled, storedReview.LastError, storedReview.GitHubHost},
		"issue":  {storedIssue.Enabled, storedIssue.LastError, storedIssue.GitHubHost},
	} {
		if !got.enabled || got.host != unconnectedTestHost || !strings.Contains(got.lastError, unconnectedTestHost) {
			t.Fatalf("%s watch = %+v, want enabled with the host error recorded", name, got)
		}
	}
}

func TestWatchHostErrorsClearRecordedErrorAfterReconnect(t *testing.T) {
	fixture := newHostRoutingFixture(t)
	ctx := context.Background()
	store := fixture.service.store
	review := &ReviewWatch{WorkspaceID: testWorkspaceID, WorkflowID: "wf", WorkflowStepID: "step",
		AgentProfileID: "agent", ExecutorProfileID: "exec", Enabled: true, GitHubHost: enterpriseTestHost}
	if err := store.CreateReviewWatch(ctx, review); err != nil {
		t.Fatalf("create review watch: %v", err)
	}
	if err := store.SetReviewWatchLastError(ctx, review.ID, "GitHub host \"octo.ghe.com\" is not connected"); err != nil {
		t.Fatalf("record error: %v", err)
	}
	review, _ = store.GetReviewWatch(ctx, review.ID)

	if _, err := fixture.service.CheckReviewWatch(ctx, review); err != nil {
		t.Fatalf("CheckReviewWatch: %v", err)
	}

	stored, _ := store.GetReviewWatch(ctx, review.ID)
	if stored.LastError != "" || stored.LastErrorAt != nil {
		t.Fatalf("last error after reconnect = %q at %v", stored.LastError, stored.LastErrorAt)
	}
}
