package backendapp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/kandev/kandev/internal/db"
	"github.com/kandev/kandev/internal/github"
)

func TestE2EResetDeletesEveryGitHubHostPAT(t *testing.T) {
	raw, err := db.OpenSQLite(filepath.Join(t.TempDir(), "e2e-reset-hosts.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	database := sqlx.NewDb(raw, "sqlite3")
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`
		CREATE TABLE github_workspace_connections (workspace_id TEXT, github_host TEXT);
		CREATE TABLE github_user_connections (workspace_id TEXT, user_id TEXT);
		CREATE TABLE github_auth_flows (state_hash TEXT PRIMARY KEY, workspace_id TEXT);
		CREATE TABLE secrets (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	for _, id := range []string{
		github.WorkspacePATSecretKey("ws-1", "github.com"),
		github.WorkspacePATSecretKey("ws-1", "octo.ghe.com"),
		github.WorkspacePATSecretKey("ws-10", "octo.ghe.com"),
	} {
		if _, err := database.Exec(`INSERT INTO secrets (id) VALUES (?)`, id); err != nil {
			t.Fatalf("seed secret %s: %v", id, err)
		}
	}

	if err := deleteGitHubAuthForReset(context.Background(), database.DB, "ws-1"); err != nil {
		t.Fatalf("delete GitHub auth: %v", err)
	}

	var remaining []string
	if err := database.Select(&remaining, `SELECT id FROM secrets ORDER BY id`); err != nil {
		t.Fatalf("list secrets: %v", err)
	}
	if len(remaining) != 1 || remaining[0] != github.WorkspacePATSecretKey("ws-10", "octo.ghe.com") {
		t.Fatalf("remaining secrets = %v", remaining)
	}
}
