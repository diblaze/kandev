package automation

import (
	"testing"

	"github.com/kandev/kandev/internal/github"
)

func TestMergedRepoHostsMatchTheTaskPRHost(t *testing.T) {
	enterprise := GitHubPRMergedTriggerConfig{Repos: []github.RepoFilter{{Host: enterpriseTriggerHost, Owner: "acme"}}}
	legacy := GitHubPRMergedTriggerConfig{Repos: []github.RepoFilter{{Owner: "acme", Name: "repo"}}}
	for _, tc := range []struct {
		name string
		host string
		cfg  GitHubPRMergedTriggerConfig
		want bool
	}{
		{"enterprise filter, enterprise PR", enterpriseTriggerHost, enterprise, true},
		{"enterprise filter, github.com PR", "github.com", enterprise, false},
		{"legacy filter, github.com PR", "", legacy, true},
		{"legacy filter, enterprise PR", enterpriseTriggerHost, legacy, false},
	} {
		if got := matchesMergedRepo(tc.host, "acme", "repo", tc.cfg); got != tc.want {
			t.Errorf("%s: matchesMergedRepo = %v, want %v", tc.name, got, tc.want)
		}
	}
}
