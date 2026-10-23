package configsync

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/db/dialect"
	"github.com/kandev/kandev/internal/github"
	"github.com/kandev/kandev/internal/testutil"
)

const enterpriseConfigHost = "octo.ghe.com"

type hostRecordingGitHub struct {
	*fakeGitHub
	hosts []string
}

func (f *hostRecordingGitHub) ListRepoDirectoryForWorkspace(
	ctx context.Context, workspaceID, host, owner, repo, path, ref string,
) ([]github.RepoContentEntry, error) {
	f.hosts = append(f.hosts, host)
	return f.fakeGitHub.ListRepoDirectoryForWorkspace(ctx, workspaceID, host, owner, repo, path, ref)
}

func (f *hostRecordingGitHub) GetRepoFileContentForWorkspace(
	ctx context.Context, workspaceID, host, owner, repo, path, ref string,
) ([]byte, error) {
	f.hosts = append(f.hosts, host)
	return f.fakeGitHub.GetRepoFileContentForWorkspace(ctx, workspaceID, host, owner, repo, path, ref)
}

func TestConfigSyncHostsWalkReadsTheConfiguredHost(t *testing.T) {
	for _, tc := range []struct{ name, host, want string }{
		{"enterprise host", enterpriseConfigHost, enterpriseConfigHost},
		{"no host", "", "github.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &SetConfigRequest{Provider: ProviderGitHub, GitHubHost: tc.host, RepoOwner: "acme", RepoName: "cfg"}
			require.NoError(t, req.Normalize())
			cfg, err := setupTestStore(t).UpsertConfigForWorkspace(context.Background(), "ws-1", req)
			require.NoError(t, err)
			fg := &hostRecordingGitHub{fakeGitHub: newFakeGitHub()}
			fg.dirs[""] = []github.RepoContentEntry{dirEntry("agents")}
			fg.dirs["agents"] = []github.RepoContentEntry{fileEntry("agents/a.yml")}
			fg.files["agents/a.yml"] = []byte("name: a\n")

			_, _ = newWalker(fg, nil, DefaultLimits).Walk(context.Background(), cfg)

			require.NotEmpty(t, fg.hosts)
			for _, host := range fg.hosts {
				assert.Equal(t, tc.want, host)
			}
		})
	}
}

func TestConfigSyncHostsRejectInvalidHost(t *testing.T) {
	req := &SetConfigRequest{Provider: ProviderGitHub, GitHubHost: "https://octo.ghe.com/x", RepoOwner: "acme", RepoName: "cfg"}

	assert.ErrorIs(t, req.Normalize(), ErrInvalidConfig)
}

func assertConfigHostReplay(t *testing.T, database *sqlx.DB) {
	t.Helper()
	ctx := context.Background()
	store, err := NewStore(database, database)
	require.NoError(t, err)
	_, err = store.UpsertConfigForWorkspace(ctx, "ws-1", testRequest())
	require.NoError(t, err)
	_, err = database.Exec(`ALTER TABLE office_config_sync_configs DROP COLUMN github_host`)
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

func TestConfigSyncHostsLegacyConfigReplaysAsGitHubDotCom(t *testing.T) {
	assertConfigHostReplay(t, setupTestStore(t).db)
}

func TestConfigSyncHostsPostgresReplay(t *testing.T) {
	assertConfigHostReplay(t, testutil.OpenIsolatedPostgres(t, testutil.PostgresDSNFromEnv(t)))
}
