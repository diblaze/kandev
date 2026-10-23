package github

import (
	"testing"
	"time"
)

func TestUnwatchedTaskPRHostsWatchOnOneHostDoesNotCoverTheOther(t *testing.T) {
	watches := []*PRWatch{{GitHubHost: enterpriseTestHost, Owner: "o", Repo: "r", PRNumber: 7}}
	rows := []*TaskPR{
		{TaskID: "task-public", Owner: "o", Repo: "r", PRNumber: 7, State: prStateOpen},
		{TaskID: "task-enterprise", GitHubHost: enterpriseTestHost, Owner: "o", Repo: "r", PRNumber: 7, State: prStateOpen},
	}

	pending := unwatchedTaskPRs(rows, watches, time.Now(), true)

	if len(pending) != 1 || pending[0].TaskID != "task-public" {
		t.Fatalf("pending = %+v, want only the github.com row", pending)
	}
}
