package github

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

// legacyConnectionSchemaSQL is the connection schema before connections were
// keyed by GitHub host.
//
//nolint:dupword // each SQLite trigger body ends a CASE and then the trigger with END.
const legacyConnectionSchemaSQL = `
	CREATE TABLE IF NOT EXISTS github_workspace_connections (
		workspace_id TEXT PRIMARY KEY,
		source TEXT NOT NULL CHECK (source IN ('legacy_shared', 'pat', 'gh_cli', 'github_app_installation')),
		github_host TEXT NOT NULL CHECK (github_host <> ''),
		login TEXT,
		installation_id BIGINT,
		installation_account_login TEXT,
		installation_account_type TEXT CHECK (
			installation_account_type IS NULL OR installation_account_type IN ('User', 'Organization')
		),
		app_registration_id TEXT,
		status TEXT NOT NULL CHECK (status IN ('active', 'invalid', 'suspended', 'revoked')),
		credential_generation BIGINT NOT NULL DEFAULT 1 CHECK (credential_generation > 0),
		last_error TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		CHECK (
			(source = 'legacy_shared' AND login IS NULL AND installation_id IS NULL AND app_registration_id IS NULL) OR
			(source IN ('pat', 'gh_cli') AND login IS NOT NULL AND login <> '' AND installation_id IS NULL AND app_registration_id IS NULL) OR
			(source = 'github_app_installation' AND installation_id IS NOT NULL AND installation_id > 0 AND
			 installation_account_login IS NOT NULL AND installation_account_login <> '' AND
			 installation_account_type IS NOT NULL AND app_registration_id IS NOT NULL)
		),
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
		FOREIGN KEY (app_registration_id) REFERENCES github_app_registrations(id) ON DELETE RESTRICT
	);

	CREATE TABLE IF NOT EXISTS github_user_connections (
		workspace_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		app_registration_id TEXT NOT NULL,
		github_user_id BIGINT NOT NULL CHECK (github_user_id > 0),
		login TEXT NOT NULL CHECK (login <> ''),
		status TEXT NOT NULL CHECK (status IN ('active', 'invalid', 'revoked')),
		access_expires_at TIMESTAMP NOT NULL,
		refresh_expires_at TIMESTAMP,
		credential_generation BIGINT NOT NULL DEFAULT 1 CHECK (credential_generation > 0),
		last_error TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		PRIMARY KEY (workspace_id, user_id),
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
		FOREIGN KEY (app_registration_id) REFERENCES github_app_registrations(id) ON DELETE RESTRICT
	);

	CREATE TRIGGER IF NOT EXISTS github_user_connections_registration_insert
	BEFORE INSERT ON github_user_connections
	BEGIN
		SELECT CASE WHEN NOT EXISTS (
			SELECT 1 FROM github_workspace_connections workspace
			WHERE workspace.workspace_id = NEW.workspace_id
				AND workspace.source = 'github_app_installation'
				AND workspace.app_registration_id = NEW.app_registration_id
		) THEN RAISE(ABORT, 'personal GitHub App registration must match workspace') END;
	END;

	CREATE TRIGGER IF NOT EXISTS github_user_connections_registration_update
	BEFORE UPDATE OF workspace_id, app_registration_id ON github_user_connections
	BEGIN
		SELECT CASE WHEN NOT EXISTS (
			SELECT 1 FROM github_workspace_connections workspace
			WHERE workspace.workspace_id = NEW.workspace_id
				AND workspace.source = 'github_app_installation'
				AND workspace.app_registration_id = NEW.app_registration_id
		) THEN RAISE(ABORT, 'personal GitHub App registration must match workspace') END;
	END;

	CREATE TRIGGER IF NOT EXISTS github_workspace_connections_registration_update
	BEFORE UPDATE OF source, app_registration_id ON github_workspace_connections
	BEGIN
		SELECT CASE WHEN EXISTS (
			SELECT 1 FROM github_user_connections personal
			WHERE personal.workspace_id = OLD.workspace_id
				AND (
					NEW.source <> 'github_app_installation' OR
					NEW.app_registration_id IS NULL OR
					personal.app_registration_id <> NEW.app_registration_id
				)
		) THEN RAISE(ABORT, 'workspace GitHub App registration must match personal connections') END;
	END;

	CREATE TRIGGER IF NOT EXISTS github_workspace_connections_registration_delete
	BEFORE DELETE ON github_workspace_connections
	BEGIN
		SELECT CASE WHEN EXISTS (
			SELECT 1 FROM github_user_connections personal
			WHERE personal.workspace_id = OLD.workspace_id
		) THEN RAISE(ABORT, 'workspace GitHub App connection still has personal connections') END;
	END;

	CREATE TABLE IF NOT EXISTS github_user_connection_versions (
		workspace_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		credential_generation BIGINT NOT NULL DEFAULT 0 CHECK (credential_generation >= 0),
		updated_at TIMESTAMP NOT NULL,
		PRIMARY KEY (workspace_id, user_id),
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
	);
`

var connectionSchemaObjects = []string{
	"github_workspace_connections", "github_user_connections", "github_user_connection_versions",
}

var legacyConnectionFixtureTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func installLegacyConnectionSchema(t *testing.T, store *Store) {
	t.Helper()
	statements := []string{
		`DROP TRIGGER IF EXISTS github_user_connections_registration_insert`,
		`DROP TRIGGER IF EXISTS github_user_connections_registration_update`,
		`DROP TRIGGER IF EXISTS github_workspace_connections_registration_update`,
		`DROP TRIGGER IF EXISTS github_workspace_connections_registration_delete`,
		`DROP TABLE github_user_connection_versions`,
		`DROP TABLE github_user_connections`,
		`DROP TABLE github_workspace_connections`,
		legacyConnectionSchemaSQL,
	}
	for _, statement := range statements {
		if _, err := store.db.Exec(statement); err != nil {
			t.Fatalf("install legacy connection schema: %v", err)
		}
	}
}

func seedLegacyConnectionRows(t *testing.T, database *sqlx.DB) {
	t.Helper()
	at := legacyConnectionFixtureTime
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO github_workspace_connections (
			workspace_id, source, github_host, installation_id, installation_account_login,
			installation_account_type, app_registration_id, status, credential_generation, created_at, updated_at
		) VALUES ('ws-1', 'github_app_installation', 'github.com', 42, 'acme', 'Organization',
			'registration-store-test', 'active', 3, ?, ?)`, []any{at, at.Add(time.Minute)}},
		{`INSERT INTO github_user_connections (
			workspace_id, user_id, app_registration_id, github_user_id, login, status, access_expires_at,
			credential_generation, created_at, updated_at
		) VALUES ('ws-1', 'user-1', 'registration-store-test', 7, 'octocat', 'active', ?, 4, ?, ?)`,
			[]any{at.Add(time.Hour), at, at.Add(2 * time.Minute)}},
		{`INSERT INTO github_user_connection_versions (workspace_id, user_id, credential_generation, updated_at)
			VALUES ('ws-1', 'user-1', 5, ?)`, []any{at.Add(3 * time.Minute)}},
	}
	for _, statement := range statements {
		if _, err := database.Exec(database.Rebind(statement.query), statement.args...); err != nil {
			t.Fatalf("seed legacy connection rows: %v", err)
		}
	}
}

func newLegacyConnectionStore(t *testing.T) *Store {
	t.Helper()
	store := newTestStore(t)
	seedConnectionWorkspaces(t, store, "ws-1")
	seedStoreAppRegistration(t, store)
	installLegacyConnectionSchema(t, store)
	seedLegacyConnectionRows(t, store.db)
	return store
}

func TestStoreConnectionHostMigrationKeysExistingRowsAsGitHubCom(t *testing.T) {
	store := newLegacyConnectionStore(t)

	if err := store.migrateGitHubConnectionsToHostKey(); err != nil {
		t.Fatalf("migrate connections to host key: %v", err)
	}

	at := legacyConnectionFixtureTime
	var workspace struct {
		Host       string    `db:"github_host"`
		Generation int64     `db:"credential_generation"`
		CreatedAt  time.Time `db:"created_at"`
		UpdatedAt  time.Time `db:"updated_at"`
	}
	if err := store.db.Get(&workspace, `SELECT github_host, credential_generation, created_at, updated_at
		FROM github_workspace_connections WHERE workspace_id = 'ws-1'`); err != nil {
		t.Fatalf("read migrated workspace connection: %v", err)
	}
	if workspace.Host != defaultGitHubHost || workspace.Generation != 3 ||
		!workspace.CreatedAt.Equal(at) || !workspace.UpdatedAt.Equal(at.Add(time.Minute)) {
		t.Fatalf("migrated workspace connection = %+v", workspace)
	}
	var personal struct {
		Host       string    `db:"github_host"`
		Login      string    `db:"login"`
		Generation int64     `db:"credential_generation"`
		CreatedAt  time.Time `db:"created_at"`
		UpdatedAt  time.Time `db:"updated_at"`
	}
	if err := store.db.Get(&personal, `SELECT github_host, login, credential_generation, created_at, updated_at
		FROM github_user_connections WHERE workspace_id = 'ws-1' AND user_id = 'user-1'`); err != nil {
		t.Fatalf("read migrated personal connection: %v", err)
	}
	if personal.Host != defaultGitHubHost || personal.Login != "octocat" || personal.Generation != 4 ||
		!personal.CreatedAt.Equal(at) || !personal.UpdatedAt.Equal(at.Add(2*time.Minute)) {
		t.Fatalf("migrated personal connection = %+v", personal)
	}
	var version struct {
		Host       string    `db:"github_host"`
		Generation int64     `db:"credential_generation"`
		UpdatedAt  time.Time `db:"updated_at"`
	}
	if err := store.db.Get(&version, `SELECT github_host, credential_generation, updated_at
		FROM github_user_connection_versions WHERE workspace_id = 'ws-1' AND user_id = 'user-1'`); err != nil {
		t.Fatalf("read migrated personal version: %v", err)
	}
	if version.Host != defaultGitHubHost || version.Generation != 5 || !version.UpdatedAt.Equal(at.Add(3*time.Minute)) {
		t.Fatalf("migrated personal version = %+v", version)
	}
}

func connectionSchemaSnapshot(t *testing.T, database *sqlx.DB) string {
	t.Helper()
	var snapshot strings.Builder
	rows, err := database.Queryx(`SELECT type, name, COALESCE(sql, '') FROM sqlite_master
		WHERE tbl_name IN (?, ?, ?) ORDER BY type, name`,
		connectionSchemaObjects[0], connectionSchemaObjects[1], connectionSchemaObjects[2])
	if err != nil {
		t.Fatalf("read connection schema: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var kind, name, definition string
		if err := rows.Scan(&kind, &name, &definition); err != nil {
			t.Fatalf("scan connection schema: %v", err)
		}
		fmt.Fprintf(&snapshot, "%s %s %s\n", kind, name, definition)
	}
	for _, table := range connectionSchemaObjects {
		tableRows, err := database.Queryx(`SELECT * FROM ` + table + ` ORDER BY workspace_id`)
		if err != nil {
			t.Fatalf("read %s rows: %v", table, err)
		}
		for tableRows.Next() {
			row := map[string]any{}
			if err := tableRows.MapScan(row); err != nil {
				t.Fatalf("scan %s row: %v", table, err)
			}
			fmt.Fprintf(&snapshot, "%s %v\n", table, row)
		}
		_ = tableRows.Close()
	}
	return snapshot.String()
}

func TestStoreConnectionHostMigrationReplayIsNoOp(t *testing.T) {
	store := newLegacyConnectionStore(t)
	if _, err := NewStore(store.db, store.db); err != nil {
		t.Fatalf("upgrade legacy store: %v", err)
	}
	migrated := connectionSchemaSnapshot(t, store.db)

	if _, err := NewStore(store.db, store.db); err != nil {
		t.Fatalf("replay migrated store: %v", err)
	}

	if replayed := connectionSchemaSnapshot(t, store.db); replayed != migrated {
		t.Fatalf("replay changed connection schema:\nbefore:\n%s\nafter:\n%s", migrated, replayed)
	}
}

func TestStoreConnectionHostMigrationTriggersMatchHost(t *testing.T) {
	store := newLegacyConnectionStore(t)
	if err := store.migrateGitHubConnectionsToHostKey(); err != nil {
		t.Fatalf("migrate connections to host key: %v", err)
	}

	_, err := store.db.Exec(`INSERT INTO github_user_connections (
		workspace_id, user_id, github_host, app_registration_id, github_user_id, login, status,
		access_expires_at, credential_generation, created_at, updated_at
	) VALUES ('ws-1', 'user-2', 'octo.ghe.com', 'registration-store-test', 8, 'hubot', 'active', ?, 1, ?, ?)`,
		legacyConnectionFixtureTime, legacyConnectionFixtureTime, legacyConnectionFixtureTime)
	if err == nil || !strings.Contains(err.Error(), "personal GitHub App registration must match workspace") {
		t.Fatalf("personal connection on a host without an App connection: err = %v", err)
	}
	if _, err := store.db.Exec(`DELETE FROM github_workspace_connections WHERE workspace_id = 'ws-1'`); err == nil {
		t.Fatal("deleting a workspace connection that still has personal connections succeeded")
	}
}

func TestStoreConnectionHostMigrationRollsBackEveryStep(t *testing.T) {
	probe := newLegacyConnectionStore(t)
	steps := 0
	probe.connectionHostMigrationFailpoint = func(int) error { steps++; return nil }
	if err := probe.migrateGitHubConnectionsToHostKey(); err != nil || steps == 0 {
		t.Fatalf("count migration steps: steps %d, err %v", steps, err)
	}

	for failAt := range steps {
		store := newLegacyConnectionStore(t)
		legacy := connectionSchemaSnapshot(t, store.db)
		injected := fmt.Errorf("injected failure after step %d", failAt)
		store.connectionHostMigrationFailpoint = func(step int) error {
			if step == failAt {
				return injected
			}
			return nil
		}
		if err := store.migrateGitHubConnectionsToHostKey(); err == nil {
			t.Fatalf("step %d: migration succeeded despite injected failure", failAt)
		}
		if after := connectionSchemaSnapshot(t, store.db); after != legacy {
			t.Fatalf("step %d: rollback changed connection schema:\nbefore:\n%s\nafter:\n%s", failAt, legacy, after)
		}
	}
}
