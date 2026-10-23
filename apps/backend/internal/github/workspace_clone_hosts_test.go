package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func seedSourceHostPATConnections(t *testing.T, store *Store) {
	t.Helper()
	for _, host := range []string{enterpriseTestHost, defaultGitHubHost} {
		require.NoError(t, store.UpsertWorkspaceConnection(context.Background(), &WorkspaceConnection{
			WorkspaceID: "source", Source: ConnectionSourcePAT, GitHubHost: host, Login: "account-" + host,
			Status: ConnectionStatusActive, CredentialGeneration: 5,
		}))
	}
}

func TestWorkspaceCloneCopiesEveryGitHubHost(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	seedConnectionWorkspaces(t, store, "source", "target")
	seedSourceHostPATConnections(t, store)
	tx, err := store.db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })

	patHosts, err := store.CopyWorkspaceConfigurationTx(ctx, tx, "source", "target")

	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.Equal(t, []string{defaultGitHubHost, enterpriseTestHost}, patHosts)
	copied, err := store.ListWorkspaceConnections(ctx, "target")
	require.NoError(t, err)
	require.Len(t, copied, 2)
	require.Equal(t, "account-"+enterpriseTestHost, copied[1].Login)
}

func TestCopyWorkspaceConnectionToWorkspaceCopiesEveryHost(t *testing.T) {
	service, secrets := newWorkspaceConnectionService(t, "octocat")
	seedConnectionWorkspaces(t, service.store, "source", "target")
	seedSourceHostPATConnections(t, service.store)
	for _, host := range []string{defaultGitHubHost, enterpriseTestHost} {
		secrets.values[WorkspacePATSecretKey("source", host)] = "token-" + host
	}

	require.NoError(t, service.CopyWorkspaceConnectionToWorkspace(context.Background(), "source", "target"))

	for _, host := range []string{defaultGitHubHost, enterpriseTestHost} {
		connection, err := service.store.GetWorkspaceConnection(context.Background(), "target", host)
		require.NoError(t, err)
		require.NotNil(t, connection, host)
		require.Equal(t, "token-"+host, secrets.values[WorkspacePATSecretKey("target", host)])
	}
}
