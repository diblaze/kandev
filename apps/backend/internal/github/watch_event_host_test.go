package github

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/events/bus"
)

func TestWatchEventsCarryWatchHost(t *testing.T) {
	ctx := context.Background()
	eventBus := bus.NewMemoryEventBus(testLogger(t))
	t.Cleanup(eventBus.Close)
	hosts := map[string]string{}
	for _, subject := range []string{events.GitHubNewReviewPR, events.GitHubNewIssue} {
		subscription, err := eventBus.Subscribe(subject, func(_ context.Context, event *bus.Event) error {
			switch payload := event.Data.(type) {
			case *NewReviewPREvent:
				hosts[subject] = payload.GitHubHost
			case *NewIssueEvent:
				hosts[subject] = payload.GitHubHost
			}
			return nil
		})
		if err != nil {
			t.Fatalf("subscribe %s: %v", subject, err)
		}
		t.Cleanup(func() { _ = subscription.Unsubscribe() })
	}
	service := NewService(NewMockClient(), AuthMethodPAT, nil, nil, eventBus, testLogger(t))
	t.Cleanup(service.Stop)

	service.publishNewReviewPREvent(ctx, &ReviewWatch{ID: "review-1", GitHubHost: "octo.ghe.com"}, &PR{Number: 1})
	service.publishNewIssueEvent(ctx, &IssueWatch{ID: "issue-1", GitHubHost: "ghe.example.com"}, &Issue{Number: 2})

	if hosts[events.GitHubNewReviewPR] != "octo.ghe.com" || hosts[events.GitHubNewIssue] != "ghe.example.com" {
		t.Fatalf("event hosts = %v, want the watch hosts", hosts)
	}
}
