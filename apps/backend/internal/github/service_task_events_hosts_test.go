package github

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/events/bus"
)

func TestHandleWorkspaceDeletedRemovesEveryHostSecret(t *testing.T) {
	svc, secrets := newWorkspaceConnectionService(t, "octocat")
	secrets.values[WorkspacePATSecretKey("ws-1", enterpriseTestHost)] = "enterprise-pat"
	secrets.values[UserAccessTokenSecretKey("ws-1", "user-1", enterpriseTestHost)] = "enterprise-access"
	secrets.values[WorkspacePATSecretKey("ws-10", enterpriseTestHost)] = "other"

	event := bus.NewEvent(events.WorkspaceDeleted, "task-service", map[string]interface{}{"id": "ws-1"})
	if err := svc.handleWorkspaceDeleted(context.Background(), event); err != nil {
		t.Fatalf("handleWorkspaceDeleted: %v", err)
	}

	for _, id := range []string{
		WorkspacePATSecretKey("ws-1", enterpriseTestHost),
		UserAccessTokenSecretKey("ws-1", "user-1", enterpriseTestHost),
	} {
		if _, ok := secrets.values[id]; ok {
			t.Fatalf("enterprise secret %q survived workspace deletion", id)
		}
	}
	if secrets.values[WorkspacePATSecretKey("ws-10", enterpriseTestHost)] != "other" {
		t.Fatalf("another workspace's secret changed: %v", secrets.values)
	}
}
