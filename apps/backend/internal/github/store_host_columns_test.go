package github

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/kandev/kandev/internal/db/dialect"
	"github.com/kandev/kandev/internal/testutil"
)

var hostColumnTables = []string{
	"github_pr_watches", "github_task_prs", "github_review_watches", "github_issue_watches",
}

type hostColumnRows struct {
	watch       *PRWatch
	taskPR      *TaskPR
	reviewWatch *ReviewWatch
	issueWatch  *IssueWatch
}

func createHostColumnRows(t *testing.T, store *Store, host string) hostColumnRows {
	t.Helper()
	ctx := context.Background()
	if _, err := store.db.Exec(store.db.Rebind(`INSERT INTO tasks (id, workspace_id) VALUES (?, 'ws-1')`), "task-"+host); err != nil {
		t.Fatalf("create task row: %v", err)
	}
	rows := hostColumnRows{
		watch: &PRWatch{WorkspaceID: "ws-1", TaskID: "task-" + host, RepositoryID: "repo-1",
			Owner: "octo", Repo: "app", Branch: "feat/" + host, GitHubHost: host},
		taskPR: &TaskPR{WorkspaceID: "ws-1", TaskID: "task-" + host, RepositoryID: "repo-1",
			Owner: "octo", Repo: "app", PRNumber: 7, PRURL: "https://example/pull/7", PRTitle: "t",
			HeadBranch: "feat", BaseBranch: "main", State: "open", CreatedAt: time.Now().UTC(), GitHubHost: host},
		reviewWatch: &ReviewWatch{WorkspaceID: "ws-1", WorkflowID: "wf", WorkflowStepID: "step",
			AgentProfileID: "agent", ExecutorProfileID: "exec", Enabled: true, GitHubHost: host},
		issueWatch: &IssueWatch{WorkspaceID: "ws-1", WorkflowID: "wf", WorkflowStepID: "step",
			AgentProfileID: "agent", ExecutorProfileID: "exec", Enabled: true, GitHubHost: host},
	}
	if err := store.CreatePRWatch(ctx, rows.watch); err != nil {
		t.Fatalf("create PR watch: %v", err)
	}
	if err := store.CreateTaskPR(ctx, rows.taskPR); err != nil {
		t.Fatalf("create task PR: %v", err)
	}
	if err := store.CreateReviewWatch(ctx, rows.reviewWatch); err != nil {
		t.Fatalf("create review watch: %v", err)
	}
	if err := store.CreateIssueWatch(ctx, rows.issueWatch); err != nil {
		t.Fatalf("create issue watch: %v", err)
	}
	return rows
}

func assertHostColumnRows(t *testing.T, store *Store, rows hostColumnRows, want string) {
	t.Helper()
	ctx := context.Background()
	watch, err := store.GetPRWatch(ctx, rows.watch.ID)
	if err != nil || watch == nil || watch.GitHubHost != want {
		t.Fatalf("PR watch host = %+v, %v; want %q", watch, err, want)
	}
	taskPR, err := store.GetTaskPRByRepository(ctx, rows.taskPR.TaskID, rows.taskPR.RepositoryID)
	if err != nil || taskPR == nil || taskPR.GitHubHost != want {
		t.Fatalf("task PR host = %+v, %v; want %q", taskPR, err, want)
	}
	reviewWatch, err := store.GetReviewWatch(ctx, rows.reviewWatch.ID)
	if err != nil || reviewWatch == nil || reviewWatch.GitHubHost != want {
		t.Fatalf("review watch host = %+v, %v; want %q", reviewWatch, err, want)
	}
	issueWatch, err := store.GetIssueWatch(ctx, rows.issueWatch.ID)
	if err != nil || issueWatch == nil || issueWatch.GitHubHost != want {
		t.Fatalf("issue watch host = %+v, %v; want %q", issueWatch, err, want)
	}
}

func TestStoreHostColumnsRoundTripEnterpriseHost(t *testing.T) {
	store := newTestStore(t)

	rows := createHostColumnRows(t, store, enterpriseTestHost)

	assertHostColumnRows(t, store, rows, enterpriseTestHost)
}

func TestStoreHostColumnsDefaultEmptyHostToGitHubDotCom(t *testing.T) {
	store := newTestStore(t)

	rows := createHostColumnRows(t, store, "")

	assertHostColumnRows(t, store, rows, defaultGitHubHost)
}

func TestStoreHostColumnsWatchUpdatesKeepTheirHost(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	rows := createHostColumnRows(t, store, enterpriseTestHost)

	rows.reviewWatch.Prompt = "changed"
	rows.issueWatch.Prompt = "changed"
	if err := store.UpdateReviewWatch(ctx, rows.reviewWatch); err != nil {
		t.Fatalf("update review watch: %v", err)
	}
	if err := store.UpdateIssueWatch(ctx, rows.issueWatch); err != nil {
		t.Fatalf("update issue watch: %v", err)
	}

	assertHostColumnRows(t, store, rows, enterpriseTestHost)
}

func TestStoreHostColumnsReplayAddsDefaultToLegacyRows(t *testing.T) {
	store := newTestStore(t)
	rows := createHostColumnRows(t, store, "")
	for _, table := range hostColumnTables {
		if _, err := store.db.Exec(`ALTER TABLE ` + table + ` DROP COLUMN github_host`); err != nil {
			t.Fatalf("drop %s.github_host: %v", table, err)
		}
	}

	replayed := replayHostColumnStore(t, store.db)
	replayHostColumnStore(t, store.db)

	assertHostColumnRows(t, replayed, rows, defaultGitHubHost)
	assertHostColumnDefaults(t, store.db)
}

func replayHostColumnStore(t *testing.T, database *sqlx.DB) *Store {
	t.Helper()
	store, err := NewStore(database, database)
	if err != nil {
		t.Fatalf("replay GitHub store schema: %v", err)
	}
	return store
}

func assertHostColumnDefaults(t *testing.T, database *sqlx.DB) {
	t.Helper()
	for _, table := range hostColumnTables {
		var fallback string
		query := `SELECT COALESCE(dflt_value, '') FROM pragma_table_info('` + table + `') WHERE name = 'github_host'`
		if dialect.IsPostgres(database.DriverName()) {
			query = `SELECT COALESCE(column_default, '') FROM information_schema.columns
				WHERE table_schema = current_schema() AND table_name = '` + table + `' AND column_name = 'github_host'`
		}
		if err := database.Get(&fallback, query); err != nil {
			t.Fatalf("read %s.github_host default: %v", table, err)
		}
		if fallback != "'github.com'" && fallback != "'github.com'::text" {
			t.Fatalf("%s.github_host default = %q", table, fallback)
		}
	}
}

func TestStoreHostColumnsPostgresReplay(t *testing.T) {
	database := testutil.OpenIsolatedPostgres(t, testutil.PostgresDSNFromEnv(t))
	if _, err := database.Exec(`
		CREATE TABLE workspaces (id TEXT PRIMARY KEY);
		CREATE TABLE tasks (id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, archived_at TIMESTAMPTZ)`); err != nil {
		t.Fatalf("create prerequisite tables: %v", err)
	}
	store := replayHostColumnStore(t, database)
	legacy := createHostColumnRows(t, store, "")
	for _, table := range hostColumnTables {
		if _, err := database.Exec(`ALTER TABLE ` + table + ` DROP COLUMN github_host`); err != nil {
			t.Fatalf("drop %s.github_host: %v", table, err)
		}
	}
	// pgx caches statement result shapes per connection; the legacy schema
	// above is a new shape, as it would be after a process restart.
	database.SetMaxIdleConns(0)
	database.SetMaxIdleConns(1)

	replayed := replayHostColumnStore(t, database)
	replayHostColumnStore(t, database)

	assertHostColumnRows(t, replayed, legacy, defaultGitHubHost)
	assertHostColumnDefaults(t, database)
	fresh := createHostColumnRows(t, replayed, enterpriseTestHost+"x")
	assertHostColumnRows(t, replayed, fresh, enterpriseTestHost+"x")
}

func TestStoreHostColumnsLegacyRebuildsPreserveHost(t *testing.T) {
	database := openLegacyGitHubDB(t)
	now := time.Now().UTC()
	if _, err := database.Exec(`
		INSERT INTO workspaces (id) VALUES ('ws-1');
		INSERT INTO tasks (id, workspace_id) VALUES ('task-1', 'ws-1');
		ALTER TABLE github_pr_watches ADD COLUMN github_host TEXT NOT NULL DEFAULT 'github.com';
		INSERT INTO github_pr_watches (id, session_id, task_id, owner, repo, pr_number, branch, github_host, created_at, updated_at)
		VALUES ('watch-1', 'session-1', 'task-1', 'octo', 'app', 0, 'feat', 'octo.ghe.com', ?, ?)`, now, now); err != nil {
		t.Fatalf("seed legacy PR watch: %v", err)
	}
	store := replayHostColumnStore(t, database)
	var tableSQL string
	if err := database.Get(&tableSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'github_task_prs'`); err != nil {
		t.Fatalf("read task PR schema: %v", err)
	}
	legacySQL := strings.Replace(tableSQL, "UNIQUE(task_id, repository_id, pr_number)", "UNIQUE(task_id, pr_number)", 1)
	if _, err := database.Exec(`DROP TABLE github_task_prs`); err != nil {
		t.Fatalf("drop task PR table: %v", err)
	}
	if _, err := database.Exec(legacySQL); err != nil {
		t.Fatalf("create legacy task PR table: %v", err)
	}
	taskPR := &TaskPR{WorkspaceID: "ws-1", TaskID: "task-1", Owner: "octo", Repo: "app", PRNumber: 7,
		PRURL: "https://octo.ghe.com/octo/app/pull/7", PRTitle: "t", HeadBranch: "feat", BaseBranch: "main",
		State: "open", CreatedAt: now, GitHubHost: enterpriseTestHost}
	if err := store.CreateTaskPR(context.Background(), taskPR); err != nil {
		t.Fatalf("seed legacy task PR: %v", err)
	}

	replayed := replayHostColumnStore(t, database)

	watch, err := replayed.GetPRWatch(context.Background(), "watch-1")
	if err != nil || watch == nil || watch.GitHubHost != enterpriseTestHost {
		t.Fatalf("rebuilt PR watch = %+v, %v", watch, err)
	}
	rebuilt, err := replayed.GetTaskPRByID(context.Background(), taskPR.ID)
	if err != nil || rebuilt == nil || rebuilt.GitHubHost != enterpriseTestHost {
		t.Fatalf("rebuilt task PR = %+v, %v", rebuilt, err)
	}
}
