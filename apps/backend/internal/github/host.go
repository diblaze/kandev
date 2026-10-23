package github

import (
	"fmt"
	"net/url"
	"strings"
)

// normalizeGitHubHost returns the bare lowercase hostname of a GitHub host.
// Empty means github.com. Any form that could become a gh flag or alter a URL
// (scheme, path, port, user info, whitespace, leading dash) is rejected.
func normalizeGitHubHost(raw string) (string, error) {
	host := strings.ToLower(strings.TrimSpace(raw))
	if host == "" || host == "www."+defaultGitHubHost {
		return defaultGitHubHost, nil
	}
	if strings.HasPrefix(host, "-") || strings.ContainsAny(host, "/:@ \t\r\n") {
		return "", fmt.Errorf("invalid GitHub host %q", raw)
	}
	parsed, err := url.Parse("https://" + host)
	if err != nil || parsed.Hostname() != host {
		return "", fmt.Errorf("invalid GitHub host %q", raw)
	}
	return host, nil
}

const gheCloudSuffix = ".ghe.com"

// githubEndpoints are the base URLs of one GitHub host.
type githubEndpoints struct {
	Web              string
	REST             string
	GraphQL          string
	AppInstallPrefix string
}

// endpointsForHost maps a normalized host (empty means github.com) to its endpoints: github.com, a
// GHE.com tenant (<sub>.ghe.com, API on api.<sub>.ghe.com), or GitHub
// Enterprise Server (API under /api/v3 and /api/graphql on the host itself).
func endpointsForHost(host string) githubEndpoints {
	host = githubHostOrDefault(host)
	web := "https://" + host
	if host == defaultGitHubHost || strings.HasSuffix(host, gheCloudSuffix) {
		api := "https://api." + host
		return githubEndpoints{Web: web, REST: api, GraphQL: api + "/graphql", AppInstallPrefix: "/apps/"}
	}
	return githubEndpoints{Web: web, REST: web + "/api/v3", GraphQL: web + "/api/graphql", AppInstallPrefix: "/github-apps/"}
}

// WebHostFromURL returns the GitHub host of an https pull-request, issue, or
// repository URL. A URL with another scheme, user info, a port, or an invalid
// host is an error, so it cannot be routed to the github.com connection.
func WebHostFromURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" {
		return "", fmt.Errorf("invalid GitHub URL %q", raw)
	}
	return normalizeGitHubHost(parsed.Hostname())
}

// HostFromProviderHost returns the GitHub host of a repository provider_host,
// stored as the origin https://<host> or, in older rows, a bare host. An empty
// or malformed value is an unknown identity and an error.
func HostFromProviderHost(providerHost string) (string, error) {
	raw := strings.TrimSpace(providerHost)
	if raw == "" {
		return "", fmt.Errorf("repository has no GitHub host")
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "https" || strings.Trim(parsed.Path, "/") != "" {
			return "", fmt.Errorf("invalid GitHub provider host %q", providerHost)
		}
		raw = parsed.Host
	}
	return normalizeGitHubHost(raw)
}

// RepositoryGitHubHost returns the GitHub host of a GitHub repository row's
// provider_host. An empty value is a row saved before provider hosts existed,
// which the repositories migration treats as github.com; a malformed value is
// an error.
func RepositoryGitHubHost(providerHost string) (string, error) {
	if strings.TrimSpace(providerHost) == "" {
		return defaultGitHubHost, nil
	}
	return HostFromProviderHost(providerHost)
}

// githubHostOrDefault maps an empty stored or configured host to github.com.
func githubHostOrDefault(host string) string {
	if host == "" {
		return defaultGitHubHost
	}
	return host
}

// applyWatchHost replaces a stored watch host with a validated update, when
// the update names one.
func applyWatchHost(stored *string, update *string) error {
	if update == nil {
		return nil
	}
	host, err := normalizeGitHubHost(*update)
	if err != nil {
		return err
	}
	*stored = host
	return nil
}
