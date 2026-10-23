package orchestrator

import (
	"context"
	"testing"
)

type hostRecordingResolver struct {
	hosts []string
}

func (r *hostRecordingResolver) ResolveForReview(
	_ context.Context, _, _, host, _, _, _ string,
) (string, string, error) {
	r.hosts = append(r.hosts, host)
	return "repo1", "main", nil
}

func TestReviewRepositoryResolutionUsesWatchHost(t *testing.T) {
	resolver := &hostRecordingResolver{}
	reviewSvc, _ := setupReviewTaskTest(t)
	reviewSvc.SetGitHubService(&mockGitHubService{reserveReturn: true})
	reviewSvc.SetReviewTaskCreator(&countingReviewTaskCreator{taskID: "task-review"})
	reviewSvc.repositoryResolver = resolver
	reviewEvent := newReviewEvent()
	reviewEvent.GitHubHost = "octo.ghe.com"
	reviewSvc.createReviewTask(context.Background(), reviewEvent)

	issueSvc, _ := setupIssueTaskTest(t)
	issueSvc.SetGitHubService(&mockGitHubService{issueReserveReturn: true})
	issueSvc.SetIssueTaskCreator(&countingIssueTaskCreator{})
	issueSvc.repositoryResolver = resolver
	issueEvent := newIssueEvent()
	issueEvent.GitHubHost = "ghe.example.com"
	issueSvc.createIssueTask(context.Background(), issueEvent)

	if len(resolver.hosts) != 2 || resolver.hosts[0] != "octo.ghe.com" || resolver.hosts[1] != "ghe.example.com" {
		t.Fatalf("resolver hosts = %v, want the watch hosts", resolver.hosts)
	}
}
