package github

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/kandev/kandev/internal/db/dialect"
	"github.com/kandev/kandev/internal/testutil"
)

// legacyPostgresConnectionTriggers are the trigger functions from before
// personal connections carried a GitHub host.
func legacyPostgresConnectionTriggers() string {
	return strings.NewReplacer(
		"\n\t\t\tAND workspace.github_host = NEW.github_host", "",
		"\n\t\t\t\tAND personal.github_host = OLD.github_host", "",
		"\n\t\t\tAND personal.github_host = OLD.github_host", "",
	).Replace(postgresGitHubAuthTriggers())
}

func newPostgresLegacyConnectionStore(t *testing.T) *Store {
	t.Helper()
	database := testutil.OpenIsolatedPostgres(t, testutil.PostgresDSNFromEnv(t))
	if _, err := database.Exec(`
		CREATE TABLE workspaces (id TEXT PRIMARY KEY);
		CREATE TABLE tasks (id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, archived_at TIMESTAMPTZ);
		INSERT INTO workspaces (id) VALUES ('ws-1')`); err != nil {
		t.Fatalf("create prerequisite tables: %v", err)
	}
	store, err := NewStore(database, database)
	if err != nil {
		t.Fatalf("initialize GitHub store: %v", err)
	}
	seedStoreAppRegistration(t, store)
	legacyTables := dialect.MustRenderSchema(database.DriverName(), withoutSQLiteGitHubAuthTriggers(legacyConnectionSchemaSQL))
	for _, statement := range []string{
		`DROP TABLE github_user_connection_versions, github_user_connections, github_workspace_connections CASCADE`,
		legacyTables,
		legacyPostgresConnectionTriggers(),
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("install legacy PostgreSQL connection schema: %v", err)
		}
	}
	seedLegacyConnectionRows(t, database)
	return store
}

func postgresConnectionSchemaSnapshot(t *testing.T, database *sqlx.DB) string {
	t.Helper()
	var snapshot strings.Builder
	rows, err := database.Queryx(`
		SELECT table_name, column_name, data_type, COALESCE(column_default, ''), is_nullable
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name IN ($1, $2, $3)
		ORDER BY table_name, ordinal_position`,
		connectionSchemaObjects[0], connectionSchemaObjects[1], connectionSchemaObjects[2])
	if err != nil {
		t.Fatalf("read PostgreSQL connection columns: %v", err)
	}
	for rows.Next() {
		var table, column, kind, fallback, nullable string
		if err := rows.Scan(&table, &column, &kind, &fallback, &nullable); err != nil {
			t.Fatalf("scan PostgreSQL connection column: %v", err)
		}
		fmt.Fprintf(&snapshot, "column %s.%s %s %s %s\n", table, column, kind, fallback, nullable)
	}
	_ = rows.Close()
	for _, table := range connectionSchemaObjects {
		_, definition, err := (&Store{db: database}).postgresPrimaryKey(table)
		if err != nil {
			t.Fatalf("read %s primary key: %v", table, err)
		}
		fmt.Fprintf(&snapshot, "key %s %s\n", table, definition)
		tableRows, err := database.Queryx(`SELECT * FROM ` + table + ` ORDER BY workspace_id`)
		if err != nil {
			t.Fatalf("read %s rows: %v", table, err)
		}
		for tableRows.Next() {
			row := map[string]any{}
			if err := tableRows.MapScan(row); err != nil {
				t.Fatalf("scan %s row: %v", table, err)
			}
			fmt.Fprintf(&snapshot, "row %s %v\n", table, row)
		}
		_ = tableRows.Close()
	}
	return snapshot.String()
}

func TestStoreConnectionHostMigrationPostgresKeysExistingRowsAsGitHubCom(t *testing.T) {
	store := newPostgresLegacyConnectionStore(t)

	if _, err := NewStore(store.db, store.db); err != nil {
		t.Fatalf("upgrade legacy PostgreSQL store: %v", err)
	}

	for table, want := range map[string]string{
		"github_workspace_connections":    "PRIMARY KEY (workspace_id, github_host)",
		"github_user_connections":         "PRIMARY KEY (workspace_id, user_id, github_host)",
		"github_user_connection_versions": "PRIMARY KEY (workspace_id, user_id, github_host)",
	} {
		if _, definition, err := store.postgresPrimaryKey(table); err != nil || definition != want {
			t.Fatalf("%s primary key = %q, err %v; want %q", table, definition, err, want)
		}
	}
	personal, err := store.GetUserConnection(t.Context(), "ws-1", "user-1", defaultGitHubHost)
	if err != nil || personal == nil || personal.Login != "octocat" || personal.CredentialGeneration != 4 ||
		!personal.CreatedAt.Equal(legacyConnectionFixtureTime) {
		t.Fatalf("migrated personal connection = %+v, err %v", personal, err)
	}
	generation, err := store.GetUserConnectionGeneration(t.Context(), "ws-1", "user-1", defaultGitHubHost)
	if err != nil || generation != 5 {
		t.Fatalf("migrated personal generation = %d, err %v", generation, err)
	}
	workspace, err := store.GetWorkspaceConnection(t.Context(), "ws-1", defaultGitHubHost)
	if err != nil || workspace == nil || workspace.CredentialGeneration != 3 ||
		!workspace.CreatedAt.Equal(legacyConnectionFixtureTime) {
		t.Fatalf("migrated workspace connection = %+v, err %v", workspace, err)
	}
	if err := store.DeleteWorkspaceConnection(t.Context(), "ws-1", defaultGitHubHost); err == nil {
		t.Fatal("deleting a workspace connection that still has personal connections succeeded")
	}

	migrated := postgresConnectionSchemaSnapshot(t, store.db)
	if _, err := NewStore(store.db, store.db); err != nil {
		t.Fatalf("replay migrated PostgreSQL store: %v", err)
	}
	if replayed := postgresConnectionSchemaSnapshot(t, store.db); replayed != migrated {
		t.Fatalf("replay changed PostgreSQL connection schema:\nbefore:\n%s\nafter:\n%s", migrated, replayed)
	}
}

func TestStoreConnectionHostMigrationPostgresRollsBackEveryStep(t *testing.T) {
	probe := newPostgresLegacyConnectionStore(t)
	steps := 0
	probe.connectionHostMigrationFailpoint = func(int) error { steps++; return nil }
	if err := probe.migrateGitHubConnectionsToHostKey(); err != nil || steps == 0 {
		t.Fatalf("count PostgreSQL migration steps: steps %d, err %v", steps, err)
	}

	for failAt := range steps {
		store := newPostgresLegacyConnectionStore(t)
		legacy := postgresConnectionSchemaSnapshot(t, store.db)
		store.connectionHostMigrationFailpoint = func(step int) error {
			if step == failAt {
				return fmt.Errorf("injected failure after step %d", step)
			}
			return nil
		}
		if err := store.migrateGitHubConnectionsToHostKey(); err == nil {
			t.Fatalf("step %d: PostgreSQL migration succeeded despite injected failure", failAt)
		}
		if after := postgresConnectionSchemaSnapshot(t, store.db); after != legacy {
			t.Fatalf("step %d: rollback changed PostgreSQL schema:\nbefore:\n%s\nafter:\n%s", failAt, legacy, after)
		}
	}
}

func TestStoreConnectionHostsPostgresCoexistPerHost(t *testing.T) {
	database := testutil.OpenIsolatedPostgres(t, testutil.PostgresDSNFromEnv(t))
	if _, err := database.Exec(`
		CREATE TABLE workspaces (id TEXT PRIMARY KEY);
		CREATE TABLE tasks (id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, archived_at TIMESTAMPTZ);
		INSERT INTO workspaces (id) VALUES ('ws-1')`); err != nil {
		t.Fatalf("create prerequisite tables: %v", err)
	}
	store, err := NewStore(database, database)
	if err != nil {
		t.Fatalf("initialize GitHub store: %v", err)
	}
	ctx := t.Context()
	seedHostAppConnections(t, store)
	for host, registrationID := range map[string]string{
		defaultGitHubHost: "registration-public", enterpriseTestHost: "registration-enterprise",
	} {
		if err := store.UpsertUserConnection(ctx, &UserConnection{
			WorkspaceID: "ws-1", UserID: "user-1", GitHubHost: host, AppRegistrationID: registrationID,
			GitHubUserID: 7, Login: "login-" + host, Status: ConnectionStatusActive,
			AccessExpiresAt: legacyConnectionFixtureTime, CredentialGeneration: 2,
		}); err != nil {
			t.Fatalf("upsert %s personal connection: %v", host, err)
		}
	}

	if err := store.DeleteUserConnectionsByWorkspace(ctx, "ws-1", defaultGitHubHost); err != nil {
		t.Fatalf("delete github.com personal connections: %v", err)
	}
	if err := store.DeleteWorkspaceConnection(ctx, "ws-1", defaultGitHubHost); err != nil {
		t.Fatalf("delete github.com connection: %v", err)
	}

	connections, err := store.ListWorkspaceConnections(ctx, "ws-1")
	if err != nil || len(connections) != 1 || connections[0].GitHubHost != enterpriseTestHost {
		t.Fatalf("connections = %+v, err %v", connections, err)
	}
	personal, err := store.ListUserConnectionsByWorkspace(ctx, "ws-1")
	if err != nil || len(personal) != 1 || personal[0].GitHubHost != enterpriseTestHost {
		t.Fatalf("personal connections = %+v, err %v", personal, err)
	}
	generation, err := store.GetUserConnectionGeneration(ctx, "ws-1", "user-1", defaultGitHubHost)
	if err != nil || generation != 3 {
		t.Fatalf("github.com personal generation = %d, err %v", generation, err)
	}
}

func TestWorkspaceClonePostgresCopiesEveryGitHubHost(t *testing.T) {
	database := testutil.OpenIsolatedPostgres(t, testutil.PostgresDSNFromEnv(t))
	if _, err := database.Exec(`
		CREATE TABLE workspaces (id TEXT PRIMARY KEY);
		CREATE TABLE tasks (id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, archived_at TIMESTAMPTZ);
		INSERT INTO workspaces (id) VALUES ('source'), ('target')`); err != nil {
		t.Fatalf("create prerequisite tables: %v", err)
	}
	store, err := NewStore(database, database)
	if err != nil {
		t.Fatalf("initialize GitHub store: %v", err)
	}
	seedSourceHostPATConnections(t, store)
	tx, err := database.BeginTxx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	patHosts, err := store.CopyWorkspaceConfigurationTx(t.Context(), tx, "source", "target")
	if err != nil || tx.Commit() != nil {
		t.Fatalf("copy workspace configuration: %v", err)
	}

	copied, err := store.ListWorkspaceConnections(t.Context(), "target")
	if len(patHosts) != 2 || err != nil || len(copied) != 2 {
		t.Fatalf("PAT hosts %v, copied %+v, err %v", patHosts, copied, err)
	}
}
