package github

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/events/bus"
)

func TestGitHubWebhookAppliesOnlyToRegistrationHostConnections(t *testing.T) {
	installationID := int64(42)
	connection := func(workspaceID, host string) *WorkspaceConnection {
		return &WorkspaceConnection{
			WorkspaceID: workspaceID, GitHubHost: host, AppRegistrationID: "registration-enterprise",
			Source: ConnectionSourceGitHubAppInstallation, InstallationID: &installationID,
			Status: ConnectionStatusActive,
		}
	}
	store := &githubWebhookMemoryStore{workspaces: map[int64][]*WorkspaceConnection{
		installationID: {connection("ws-enterprise", enterpriseTestHost), connection("ws-public", defaultGitHubHost)},
	}}
	eventBus := newTestMemoryBus(t)
	var received *GitHubPushEventPayload
	if _, err := eventBus.Subscribe(events.GitHubPushReceived, func(_ context.Context, e *bus.Event) error {
		received = e.Data.(*GitHubPushEventPayload)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	service := NewAppRegistrationWebhookService(
		"registration-enterprise", enterpriseTestHost, "enterprise-secret", store, nil, nil, eventBus,
	)
	payload := []byte(`{"ref":"refs/heads/main","after":"abc1234",
		"repository":{"name":"repo","owner":{"login":"acme"}},"installation":{"id":42}}`)

	if _, err := service.Handle(context.Background(),
		signedWebhookRequest("enterprise-secret", "delivery-enterprise-push", "push", payload)); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if received == nil || len(received.WorkspaceIDs) != 1 || received.WorkspaceIDs[0] != "ws-enterprise" {
		t.Fatalf("push workspaces = %+v", received)
	}
}
