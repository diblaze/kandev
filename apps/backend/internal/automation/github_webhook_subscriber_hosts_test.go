package automation

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/events/bus"
	"github.com/kandev/kandev/internal/github"
)

func publishHostPush(t *testing.T, svc *Service, host string) {
	t.Helper()
	if err := svc.eventBus.Publish(context.Background(), events.GitHubPushReceived, bus.NewEvent(
		events.GitHubPushReceived, "test", &github.GitHubPushEventPayload{
			WorkspaceIDs: []string{"ws-1"}, Host: host, Owner: "acme", Name: "repo", Branch: "main", SHA: "abc1234",
		},
	)); err != nil {
		t.Fatal(err)
	}
}

func TestGitHubWebhookSubscriberHostsMatchTheDeliveryHost(t *testing.T) {
	cases := []struct {
		name, filterHost, deliveryHost, dedupKey string
	}{
		{"enterprise filter, enterprise delivery", enterpriseTriggerHost, enterpriseTriggerHost,
			"push:" + enterpriseTriggerHost + "/acme/repo@main@abc1234"},
		{"enterprise filter, github.com delivery", enterpriseTriggerHost, "github.com", ""},
		{"no host filter, enterprise delivery", "", enterpriseTriggerHost, ""},
		{"no host filter, github.com delivery", "", "github.com", "push:acme/repo@main@abc1234"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t)
			newTestSubscriber(t, svc)
			fired := subscribeAutomationTriggered(t, svc.eventBus)
			a := createTestAutomation(t, svc, "ws-1")
			addTestTrigger(t, svc, a.ID, TriggerTypeGitHubPush, GitHubPushTriggerConfig{
				Repos: []github.RepoFilter{{Host: tc.filterHost, Owner: "acme", Name: "repo"}},
			})

			publishHostPush(t, svc, tc.deliveryHost)

			select {
			case evt := <-fired:
				if tc.dedupKey == "" || evt.DedupKey != tc.dedupKey {
					t.Fatalf("fired with dedup key %q, want %q", evt.DedupKey, tc.dedupKey)
				}
			default:
				if tc.dedupKey != "" {
					t.Fatal("expected the push trigger to fire")
				}
			}
		})
	}
}
