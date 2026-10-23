package github

import (
	"context"
	"testing"
)

func TestWatchHostsUpdateValidatesAndStoresHost(t *testing.T) {
	fixture := newHostRoutingFixture(t)
	ctx := context.Background()
	review := &ReviewWatch{WorkspaceID: testWorkspaceID, WorkflowID: "wf", WorkflowStepID: "step",
		AgentProfileID: "agent", ExecutorProfileID: "exec", Enabled: true}
	issue := &IssueWatch{WorkspaceID: testWorkspaceID, WorkflowID: "wf", WorkflowStepID: "step",
		AgentProfileID: "agent", ExecutorProfileID: "exec", Enabled: true}
	if err := fixture.service.store.CreateReviewWatch(ctx, review); err != nil {
		t.Fatalf("create review watch: %v", err)
	}
	if err := fixture.service.store.CreateIssueWatch(ctx, issue); err != nil {
		t.Fatalf("create issue watch: %v", err)
	}
	invalid, upper := "octo.ghe.com:8443", "OCTO.ghe.com"

	if err := fixture.service.UpdateReviewWatch(ctx, review.ID, &UpdateReviewWatchRequest{GitHubHost: &invalid}); err == nil {
		t.Fatal("review watch accepted a host with a port")
	}
	if err := fixture.service.UpdateIssueWatch(ctx, issue.ID, &UpdateIssueWatchRequest{GitHubHost: &invalid}); err == nil {
		t.Fatal("issue watch accepted a host with a port")
	}
	if err := fixture.service.UpdateReviewWatch(ctx, review.ID, &UpdateReviewWatchRequest{GitHubHost: &upper}); err != nil {
		t.Fatalf("update review watch host: %v", err)
	}
	if err := fixture.service.UpdateIssueWatch(ctx, issue.ID, &UpdateIssueWatchRequest{GitHubHost: &upper}); err != nil {
		t.Fatalf("update issue watch host: %v", err)
	}

	storedReview, _ := fixture.service.store.GetReviewWatch(ctx, review.ID)
	storedIssue, _ := fixture.service.store.GetIssueWatch(ctx, issue.ID)
	if storedReview.GitHubHost != enterpriseTestHost || storedIssue.GitHubHost != enterpriseTestHost {
		t.Fatalf("hosts = %q, %q", storedReview.GitHubHost, storedIssue.GitHubHost)
	}
}

func TestWatchHostsCreateRejectsInvalidHost(t *testing.T) {
	fixture := newHostRoutingFixture(t)

	_, err := fixture.service.CreateIssueWatch(context.Background(), &CreateIssueWatchRequest{
		WorkspaceID: testWorkspaceID, GitHubHost: "https://octo.ghe.com", WorkflowID: "wf", WorkflowStepID: "step",
		AgentProfileID: "agent", ExecutorProfileID: "exec",
	})

	if err == nil {
		t.Fatal("issue watch accepted a host with a scheme")
	}
}
