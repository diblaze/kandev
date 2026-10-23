package github

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/kandev/kandev/internal/db/dialect"
)

// legacyWorkspaceConnectionKey marks a connection table keyed by workspace
// alone, before one workspace could hold a connection per GitHub host.
const legacyWorkspaceConnectionKey = "workspace_id TEXT PRIMARY KEY"

const dropConnectionTriggersSQL = `
	DROP TRIGGER IF EXISTS github_user_connections_registration_insert;
	DROP TRIGGER IF EXISTS github_user_connections_registration_update;
	DROP TRIGGER IF EXISTS github_workspace_connections_registration_update;
	DROP TRIGGER IF EXISTS github_workspace_connections_registration_delete;`

type connectionTableRebuild struct {
	table      string
	create     string
	columns    string
	selectList string
}

var connectionHostKeyRebuilds = []connectionTableRebuild{
	{
		table:  "github_workspace_connections",
		create: workspaceConnectionsTableSQL,
		columns: `workspace_id, source, github_host, login, installation_id, installation_account_login,
			installation_account_type, app_registration_id, status, credential_generation, last_error,
			created_at, updated_at`,
	},
	{
		table:  "github_user_connections",
		create: userConnectionsTableSQL,
		columns: `workspace_id, user_id, github_host, app_registration_id, github_user_id, login, status,
			access_expires_at, refresh_expires_at, credential_generation, last_error, created_at, updated_at`,
		selectList: `workspace_id, user_id, 'github.com', app_registration_id, github_user_id, login, status,
			access_expires_at, refresh_expires_at, credential_generation, last_error, created_at, updated_at`,
	},
	{
		table:      "github_user_connection_versions",
		create:     userConnectionVersionsTableSQL,
		columns:    `workspace_id, user_id, github_host, credential_generation, updated_at`,
		selectList: `workspace_id, user_id, 'github.com', credential_generation, updated_at`,
	},
}

// migrateGitHubConnectionsToHostKey re-keys workspace and personal
// connections by GitHub host. Existing rows become github.com connections.
// The whole change runs in one transaction, so a failure leaves the legacy
// schema and rows untouched, and a migrated database is left as is.
func (s *Store) migrateGitHubConnectionsToHostKey() error {
	if dialect.IsPostgres(s.db.DriverName()) {
		return s.migratePostgresConnectionsToHostKey()
	}
	legacy, err := s.tableSQLContains("github_workspace_connections", legacyWorkspaceConnectionKey)
	if err != nil || !legacy {
		return err
	}
	steps := []string{dropConnectionTriggersSQL}
	for _, rebuild := range connectionHostKeyRebuilds {
		steps = append(steps, rebuild.sqliteSteps()...)
	}
	steps = append(steps, connectionTriggersSQL)
	return s.runConnectionHostMigration(steps)
}

func (r connectionTableRebuild) sqliteSteps() []string {
	selectColumns := r.selectList
	if selectColumns == "" {
		selectColumns = r.columns
	}
	replacement := r.table + "_new"
	return []string{
		strings.Replace(r.create, "CREATE TABLE IF NOT EXISTS "+r.table+" (", "CREATE TABLE "+replacement+" (", 1),
		"INSERT INTO " + replacement + " (" + r.columns + ") SELECT " + selectColumns + " FROM " + r.table,
		"DROP TABLE " + r.table,
		"ALTER TABLE " + replacement + " RENAME TO " + r.table,
	}
}

var postgresConnectionHostKeys = []struct {
	table string
	key   string
}{
	{"github_workspace_connections", "workspace_id, github_host"},
	{"github_user_connections", "workspace_id, user_id, github_host"},
	{"github_user_connection_versions", "workspace_id, user_id, github_host"},
}

func (s *Store) migratePostgresConnectionsToHostKey() error {
	steps := []string{
		`ALTER TABLE github_user_connections ADD COLUMN IF NOT EXISTS github_host TEXT NOT NULL DEFAULT 'github.com'`,
		`ALTER TABLE github_user_connection_versions ADD COLUMN IF NOT EXISTS github_host TEXT NOT NULL DEFAULT 'github.com'`,
	}
	for _, target := range postgresConnectionHostKeys {
		name, definition, err := s.postgresPrimaryKey(target.table)
		if err != nil {
			return err
		}
		if definition == "PRIMARY KEY ("+target.key+")" {
			continue
		}
		steps = append(steps,
			`ALTER TABLE `+target.table+` DROP CONSTRAINT "`+name+`"`,
			`ALTER TABLE `+target.table+` ADD PRIMARY KEY (`+target.key+`)`,
		)
	}
	if len(steps) == 2 {
		return nil
	}
	return s.runConnectionHostMigration(steps)
}

func (s *Store) postgresPrimaryKey(table string) (string, string, error) {
	var name, definition string
	err := s.db.QueryRow(`
		SELECT conname, pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = to_regclass($1) AND contype = 'p'`, table).Scan(&name, &definition)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", fmt.Errorf("%s has no primary key", table)
	}
	if err != nil {
		return "", "", fmt.Errorf("read %s primary key: %w", table, err)
	}
	return name, definition, nil
}

func (s *Store) runConnectionHostMigration(steps []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for index, step := range steps {
		if _, err := tx.Exec(step); err != nil {
			return fmt.Errorf("key GitHub connections by host: %w", err)
		}
		if s.connectionHostMigrationFailpoint != nil {
			if err := s.connectionHostMigrationFailpoint(index); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
