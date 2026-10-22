package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const invalidRefDetail = "Git reported an invalid local reference. Inspect repository references before retrying."

// @covers AC-WORKSPACES-WORKTREE-BASE-REFRESH-001.14
func TestInvalidLocalRefDiagnosticClassification(t *testing.T) {
	tests := []struct {
		name   string
		output string
		valid  bool
	}{
		{"fetch bad object", "fatal: bad object refs/heads/broken-secret", true},
		{"show-ref bad ref", "fatal: git show-ref: bad ref refs/heads/broken-secret (0000000000000000000000000000000000000000)\n", true},
		{"bad ref", "fatal: bad ref refs/remotes/origin/broken-secret", true},
		{"multiline", "From https://token:remote-secret@example.invalid/repo\nfatal: bad object refs/heads/broken-secret\nerror: remote did not send all necessary objects\n", true},
		{"whitespace", "  fatal: bad object refs/heads/broken-secret\r\n", true},
		{"generic object suffix", "error: remote did not send all necessary objects", false},
		{"missing blob", "fatal: missing blob object '0123456789abcdef'", false},
		{"arbitrary object", "fatal: bad object 0123456789abcdef", false},
		{"URL fragment", "fatal: unexpected URL https://example.invalid/fatal: bad object refs/heads/broken-secret", false},
		{"progress text", "progress: fatal: bad object refs/heads/broken-secret", false},
		{"remote error", "remote: fatal: bad object refs/heads/broken-secret", false},
		{"branch text", "fatal: unexpected branch refs/heads/bad-ref-refs/heads/broken-secret", false},
		{"missing ref name", "fatal: bad object refs/", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyRefreshDiagnostic(tt.output, errors.New("exit status 1"), nil)
			if tt.valid {
				require.Equal(t, "invalid_local_ref", got.code)
				require.Equal(t, invalidRefDetail, got.detail)
			} else {
				require.Equal(t, "unknown", got.code)
			}
			require.NotContains(t, got.detail, "secret")
		})
	}
	for _, ctxErr := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(ctxErr.Error(), func(t *testing.T) {
			output := "fatal: bad object refs/heads/broken-secret"
			want := classifyRefreshDiagnostic("", ctxErr, ctxErr)
			require.Equal(t, want, classifyRefreshDiagnostic(output, nil, ctxErr))
			require.Equal(t, want, classifyRefreshDiagnostic(output, ctxErr, nil))
		})
	}
}

// @covers AC-WORKSPACES-WORKTREE-BASE-REFRESH-001.14
// @covers AC-WORKSPACES-WORKTREE-BASE-REFRESH-001.3
// @covers AC-WORKSPACES-WORKTREE-BASE-REFRESH-001.6
func TestInvalidLocalRefRefreshDiagnostic(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, required := range []bool{true, false} {
		t.Run(fmt.Sprintf("required=%t", required), func(t *testing.T) {
			repo := newInvalidRefFixture(t)
			log, logs := newObservedWorktreeLogger(t)
			mgr, err := NewManager(newTestConfig(t), newMockStore(), log)
			require.NoError(t, err)
			ctx := t.Context()
			_, _, err = mgr.pullBaseBranchWithPolicy(ctx, repo, "main", "", required, nil)
			require.NoError(t, err, "healthy fixture must refresh before corruption")
			require.Empty(t, logs.FilterMessage("git refresh failed").All())

			broken := filepath.Join(repo, ".git", "refs", "heads", "broken-secret")
			require.NoError(t, os.WriteFile(broken, nil, 0o600))
			var events []SyncProgressEvent
			ref, _, err := mgr.pullBaseBranchWithPolicy(ctx, repo, "main", "", required, captureSyncProgress(&events))
			if required {
				require.ErrorIs(t, err, ErrGitCommandFailed)
				require.Empty(t, ref)
				require.Equal(t, SyncProgressFailed, events[len(events)-1].Status)
			} else {
				require.NoError(t, err)
				require.Equal(t, "main", ref)
				require.Equal(t, SyncProgressCompleted, events[len(events)-1].Status)
			}
			entries := logs.FilterMessage("git refresh failed").All()
			require.Len(t, entries, 1)
			fields := entries[0].ContextMap()
			require.Equal(t, "invalid_local_ref", fields["diagnostic_code"])
			require.Equal(t, invalidRefDetail, fields["detail"])
			require.Equal(t, "fetch", fields["operation"])
			require.Equal(t, "git_command_failed", fields["reason"])
			require.Equal(t, "main", fields["branch"])
			require.Equal(t, repo, fields["repository_path"])
			require.EqualValues(t, 1, fields["exit_code"])
			require.NotContains(t, fmt.Sprint(logs.All(), events, err), "broken-secret")
			contents, readErr := os.ReadFile(broken)
			require.NoError(t, readErr, "refresh must not remove the broken ref")
			require.Empty(t, contents, "refresh must not repair the broken ref")
			require.NoError(t, os.Remove(broken))
			_, _, err = mgr.pullBaseBranchWithPolicy(ctx, repo, "main", "", required, nil)
			require.NoError(t, err, "removing only the fixture ref restores refresh")
		})
	}
}

func newInvalidRefFixture(t *testing.T) string {
	t.Helper()
	origin := t.TempDir()
	runGit(t, origin, "init", "-b", "main")
	runGit(t, origin, "-c", "user.name=Test", "-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "fixture")
	repo := t.TempDir()
	runGit(t, repo, "clone", "--no-local", origin, ".")
	runGit(t, repo, "config", "fetch.recurseSubmodules", "false")
	require.Equal(t, "main", strings.TrimSpace(runGit(t, repo, "branch", "--show-current")))
	return repo
}
