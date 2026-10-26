export const DEFAULT_GITHUB_HOST = "github.com";

/**
 * Normalizes a GitHub host to a lowercase bare hostname, mirroring the backend
 * rule: empty means github.com, www.github.com is github.com, and a value with a
 * scheme, path, port, user info, whitespace, or a leading hyphen is null.
 */
export function normalizeGitHubHost(raw: string): string | null {
  const host = raw.trim().toLowerCase();
  if (host === "" || host === `www.${DEFAULT_GITHUB_HOST}`) return DEFAULT_GITHUB_HOST;
  if (host.startsWith("-") || /[/:@\s]/.test(host)) return null;
  try {
    return new URL(`https://${host}`).hostname === host ? host : null;
  } catch {
    return null;
  }
}

const GHE_CLOUD_SUFFIX = ".ghe.com";

/** SSH user of a host's clone URLs: the tenant name on GHE.com, `git` elsewhere. */
export function githubSSHUser(host: string): string {
  return host.endsWith(GHE_CLOUD_SUFFIX) ? host.slice(0, -GHE_CLOUD_SUFFIX.length) : "git";
}

export function githubWebOrigin(host: string | undefined | null): string {
  return `https://${host || DEFAULT_GITHUB_HOST}`;
}

/** Host of an https GitHub URL; null for another scheme, user info, a port, or a bad host. */
export function hostFromUrl(raw: string): string | null {
  let url: URL;
  try {
    url = new URL(raw.trim());
  } catch {
    return null;
  }
  if (url.protocol !== "https:" || url.username || url.password || url.port) return null;
  return normalizeGitHubHost(url.hostname);
}

/**
 * Host of a repository provider_host (`https://<host>` or a bare host). An empty
 * value is a row saved before provider hosts existed and means github.com.
 */
export function hostFromProviderHost(providerHost: string | undefined | null): string | null {
  const raw = (providerHost ?? "").trim();
  if (!raw) return DEFAULT_GITHUB_HOST;
  if (!raw.includes("://")) return normalizeGitHubHost(raw);
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    return null;
  }
  if (url.protocol !== "https:" || url.port || url.pathname.replace(/\//g, "")) return null;
  return normalizeGitHubHost(url.hostname);
}

/**
 * Sets the `host` request parameter for any host other than github.com. A
 * malformed host is sent as given so the backend rejects it instead of the
 * request reaching the github.com connection.
 */
export function setGitHubHostParam(params: URLSearchParams, host: string | undefined | null) {
  if (host && normalizeGitHubHost(host) !== DEFAULT_GITHUB_HOST) params.set("host", host);
  return params;
}

export function githubWorkspaceQuery(workspaceId: string, host: string | undefined | null) {
  return setGitHubHostParam(new URLSearchParams({ workspace_id: workspaceId }), host);
}
