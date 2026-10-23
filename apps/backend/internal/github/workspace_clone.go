package github

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kandev/kandev/internal/db/dialect"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

// CopyWorkspaceConfigurationTx copies only workspace-owned settings and
// automation connections of every GitHub host. It returns the hosts whose PAT
// the caller copies through the secret owner before committing this same
// transaction.
func (s *Store) CopyWorkspaceConfigurationTx(ctx context.Context, tx *sqlx.Tx, sourceID, targetID string) ([]string, error) {
	query := workspaceConnectionSelect + ` WHERE workspace_id = ? ORDER BY github_host`
	if dialect.IsPostgres(tx.DriverName()) {
		query += ` FOR SHARE`
	}
	var connections []WorkspaceConnection
	if err := tx.SelectContext(ctx, &connections, tx.Rebind(query), sourceID); err != nil {
		return nil, err
	}
	patHosts := []string{}
	for _, connection := range connections {
		switch connection.Source {
		case ConnectionSourcePAT:
			patHosts = append(patHosts, connection.GitHubHost)
		case ConnectionSourceGitHubAppInstallation, ConnectionSourceLegacyShared:
		case ConnectionSourceGHCLI:
			if err := requireGHCLIOperator(ctx); err != nil {
				return nil, err
			}
		default:
			return nil, repoerrors.ErrWorkspaceCloneConfiguration
		}
	}
	if err := s.copyCloneSettings(ctx, tx, sourceID, targetID); err != nil {
		return nil, err
	}
	if len(connections) == 0 {
		return patHosts, nil
	}
	now := time.Now().UTC()
	_, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO github_workspace_connections (workspace_id,source,github_host,login,installation_id,installation_account_login,installation_account_type,app_registration_id,status,credential_generation,last_error,created_at,updated_at)
 SELECT ?,source,github_host,login,installation_id,installation_account_login,installation_account_type,app_registration_id,status,1,NULL,?,? FROM github_workspace_connections WHERE workspace_id = ?`), targetID, now, now, sourceID)
	return patHosts, err
}

func (s *Store) copyCloneSettings(ctx context.Context, tx *sqlx.Tx, sourceID, targetID string) error {
	now := time.Now().UTC()
	_, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO github_workspace_settings (workspace_id,task_git_credentials_mode,repo_scope_mode,repo_scope_orgs,repo_scope_repos,saved_presets,default_query_presets,created_at,updated_at)
 SELECT ?,task_git_credentials_mode,repo_scope_mode,repo_scope_orgs,repo_scope_repos,saved_presets,default_query_presets,?,? FROM github_workspace_settings WHERE workspace_id = ?`), targetID, now, now, sourceID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, tx.Rebind(`INSERT INTO github_action_presets (workspace_id,pr_presets,issue_presets,updated_at) SELECT ?,pr_presets,issue_presets,? FROM github_action_presets WHERE workspace_id = ?`), targetID, now, sourceID)
	return err
}

// CopyWorkspaceConfigurationTx participates in an already authorized clone.
func (s *Service) CopyWorkspaceConfigurationTx(ctx context.Context, tx *sqlx.Tx, sourceID, targetID string) ([]string, error) {
	if s == nil || s.store == nil {
		return nil, errWorkspaceDefaultsUnavailable
	}
	return s.store.CopyWorkspaceConfigurationTx(ctx, tx, sourceID, targetID)
}
