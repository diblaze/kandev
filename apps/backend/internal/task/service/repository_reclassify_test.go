package service

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/task/models"
)

func TestReclassifyWorkspaceGitHubRepositories(t *testing.T) {
	svc, eventBus, repo := createTestService(t)
	ctx := context.Background()
	if err := repo.CreateWorkspace(ctx, &models.Workspace{ID: "ws-1", Name: "Workspace"}); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	rows := map[string]*models.Repository{
		"remote-match": {Name: "remote", SourceType: sourceTypeProvider, RemoteURL: "ssh://octo.ghe.com/org/remote"},
		"local-match": {Name: "local", SourceType: sourceTypeLocal,
			LocalPath: writeOriginRepo(t, "org@octo.ghe.com:org/local.git"), RemoteURL: "ssh://octo.ghe.com/org/local"},
		"other-host": {Name: "other", SourceType: sourceTypeProvider, RemoteURL: "https://ghe.example.com/org/other"},
		"classified": {Name: "classified", SourceType: sourceTypeProvider, Provider: "gitlab",
			ProviderHost: "https://octo.ghe.com", ProviderOwner: "org", ProviderName: "classified",
			RemoteURL: "https://octo.ghe.com/org/classified.git"},
	}
	for id, row := range rows {
		row.ID, row.WorkspaceID = id, "ws-1"
		if err := repo.CreateRepository(ctx, row); err != nil {
			t.Fatalf("CreateRepository %s: %v", id, err)
		}
	}

	count, err := svc.ReclassifyWorkspaceGitHubRepositories(ctx, "ws-1", "octo.ghe.com")
	if err != nil || count != 2 {
		t.Fatalf("Reclassify count=%d err=%v, want 2", count, err)
	}

	want := map[string][4]string{
		"remote-match": {"github", "https://octo.ghe.com", "remote", "https://octo.ghe.com/org/remote.git"},
		"local-match":  {"github", "https://octo.ghe.com", "local", "octo@octo.ghe.com:org/local.git"},
		"other-host":   {"", "", "", "https://ghe.example.com/org/other"},
		"classified":   {"gitlab", "https://octo.ghe.com", "classified", "https://octo.ghe.com/org/classified.git"},
	}
	for id, w := range want {
		stored, err := repo.GetRepository(ctx, id)
		if err != nil {
			t.Fatalf("GetRepository %s: %v", id, err)
		}
		got := [4]string{stored.Provider, stored.ProviderHost, stored.ProviderName, stored.RemoteURL}
		if got != w {
			t.Errorf("%s = %v, want %v", id, got, w)
		}
		if w[0] == "github" && stored.ProviderOwner != "org" {
			t.Errorf("%s owner = %q, want org", id, stored.ProviderOwner)
		}
	}

	updated := map[string]bool{}
	for _, event := range eventBus.GetPublishedEvents() {
		if event.Type != events.RepositoryUpdated {
			continue
		}
		if data, ok := event.Data.(map[string]interface{}); ok {
			updated[data["id"].(string)] = true
		}
	}
	if len(updated) != 2 || !updated["remote-match"] || !updated["local-match"] {
		t.Fatalf("RepositoryUpdated for %v, want remote-match and local-match", updated)
	}
}
