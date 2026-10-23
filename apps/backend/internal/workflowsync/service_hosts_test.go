package workflowsync

import (
	"context"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/db/dialect"
	"github.com/kandev/kandev/internal/github"
	"github.com/kandev/kandev/internal/testutil"
)

const enterpriseSyncHost = "octo.ghe.com"

type hostRecordingGitHubClients struct {
	client github.Client
	mu     sync.Mutex
	hosts  []string
}

func (f *hostRecordingGitHubClients) record(host string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hosts = append(f.hosts, host)
}

func (f *hostRecordingGitHubClients) ListRepoDirectoryForWorkspace(
	ctx context.Context, _, host, owner, repo, path, ref string,
) ([]github.RepoContentEntry, error) {
	f.record(host)
	return f.client.ListRepoDirectory(ctx, owner, repo, path, ref)
}

func (f *hostRecordingGitHubClients) GetRepoFileContentForWorkspace(
	ctx context.Context, _, host, owner, repo, path, ref string,
) ([]byte, error) {
	f.record(host)
	return f.client.GetRepoFileContent(ctx, owner, repo, path, ref)
}

func TestWorkflowSyncHostsReadTheConfiguredHost(t *testing.T) {
	for _, tc := range []struct{ name, host, want string }{
		{"enterprise host", enterpriseSyncHost, enterpriseSyncHost},
		{"no host", "", "github.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clients := &hostRecordingGitHubClients{client: seededMockClient()}
			log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "console"})
			require.NoError(t, err)
			svc := NewService(setupTestStore(t), clients, nil, &fakeApplier{}, log)
			cfg, err := svc.SetConfigForWorkspace(context.Background(), "ws-1", &SetConfigRequest{
				GitHubHost: tc.host, RepoOwner: "acme", RepoName: "flows",
			})
			require.NoError(t, err)
			assert.Equal(t, tc.want, cfg.GitHubHost)

			_, err = svc.SyncWorkspace(context.Background(), "ws-1")

			require.NoError(t, err)
			require.NotEmpty(t, clients.hosts)
			for _, host := range clients.hosts {
				assert.Equal(t, tc.want, host)
			}
		})
	}
}

func TestWorkflowSyncHostsRejectInvalidHost(t *testing.T) {
	req := &SetConfigRequest{GitHubHost: "octo.ghe.com:8443", RepoOwner: "acme", RepoName: "flows"}

	assert.ErrorIs(t, req.Normalize(), ErrInvalidConfig)
}

func assertHostColumnReplay(t *testing.T, database *sqlx.DB) {
	t.Helper()
	ctx := context.Background()
	store, err := NewStore(database, database)
	require.NoError(t, err)
	_, err = store.UpsertConfigForWorkspace(ctx, "ws-1", testRequest())
	require.NoError(t, err)
	_, err = database.Exec(`ALTER TABLE workflow_sync_configs DROP COLUMN github_host`)
	require.NoError(t, err)
	if dialect.IsPostgres(database.DriverName()) {
		// pgx caches result shapes per connection; reconnect as a restart would.
		database.SetMaxIdleConns(0)
		database.SetMaxIdleConns(1)
	}

	replayed, err := NewStore(database, database)
	require.NoError(t, err)
	_, err = NewStore(database, database)
	require.NoError(t, err)

	cfg, err := replayed.GetConfigForWorkspace(ctx, "ws-1")
	require.NoError(t, err)
	assert.Equal(t, "github.com", cfg.GitHubHost)
}

func TestWorkflowSyncHostsLegacyConfigReplaysAsGitHubDotCom(t *testing.T) {
	assertHostColumnReplay(t, setupTestStore(t).db)
}

func TestWorkflowSyncHostsPostgresReplay(t *testing.T) {
	assertHostColumnReplay(t, testutil.OpenIsolatedPostgres(t, testutil.PostgresDSNFromEnv(t)))
}
