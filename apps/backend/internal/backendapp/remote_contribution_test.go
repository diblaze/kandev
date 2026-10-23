package backendapp

import (
	"context"
	"testing"
)

func TestRemoteContributionCoordinatorLeavesOrdinaryRepositoriesUnmatched(t *testing.T) {
	coordinator := newRemoteContributionCoordinator(nil, nil)

	for _, rawURL := range []string{
		"https://github.com/acme/pull/repository.git",
		"https://github.com/acme/widget.git",
		"https://gitlab.example.com/group/project.git",
	} {
		resolution, matched, err := coordinator.Resolve(context.Background(), "workspace", "user", rawURL)
		if err != nil {
			t.Fatalf("Resolve(%q) error = %v", rawURL, err)
		}
		if matched || resolution != nil {
			t.Fatalf("Resolve(%q) = (%#v, %v), want an ordinary repository", rawURL, resolution, matched)
		}
	}
}

func TestRemoteContributionCoordinatorRoutesPullRequestURLsOnAnyHostToGitHub(t *testing.T) {
	coordinator := newRemoteContributionCoordinator(nil, nil)
	for rawURL, wantErr := range map[string]string{
		"https://octo.ghe.com/org/repo/pull/7":                        "GitHub integration is not configured",
		"https://github.com/acme/widget/pull/7":                       "GitHub integration is not configured",
		"https://gitlab.example.com/group/project/-/merge_requests/7": "GitLab integration is not configured",
	} {
		_, matched, err := coordinator.Resolve(context.Background(), "workspace", "user", rawURL)
		if !matched || err == nil || err.Error() != wantErr {
			t.Fatalf("Resolve(%q) = matched %v, error %v; want %q", rawURL, matched, err, wantErr)
		}
	}
}
