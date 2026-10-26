import { DEFAULT_GITHUB_HOST, normalizeGitHubHost } from "./host";

export const DEFAULT_GITHUB_HOSTS: readonly string[] = [DEFAULT_GITHUB_HOST];

/** Optional scheme, then a dotted hostname captured as group 1. */
export const GITHUB_URL_HOST_PATTERN = String.raw`^(?:https?:\/\/)?((?:[A-Za-z0-9-]+\.)+[A-Za-z0-9-]+)`;
const SEGMENT = String.raw`([A-Za-z0-9_.-]+)`;

const REPO_URL_RE = new RegExp(
  String.raw`${GITHUB_URL_HOST_PATTERN}\/${SEGMENT}\/([A-Za-z0-9_.-]+?)(?:\.git)?\/?(?:[?#].*)?$`,
);
const ITEM_URL_RE = new RegExp(
  String.raw`${GITHUB_URL_HOST_PATTERN}\/${SEGMENT}\/${SEGMENT}\/(pull|issues)\/(\d+)(?:[/?#].*)?$`,
);

/** Normalized host when it is github.com or one of `hosts`, otherwise null. */
export function listedGitHubHost(raw: string, hosts: readonly string[]): string | null {
  const host = normalizeGitHubHost(raw);
  return host && (host === DEFAULT_GITHUB_HOST || hosts.includes(host)) ? host : null;
}

/**
 * Parses a GitHub repository URL of the form
 *   https://<host>/<owner>/<repo>[.git][/][?query][#fragment]
 * on github.com or one of `hosts` (scheme and `www.` are optional).
 *
 * Returns `{ host, owner, repo }` on a match, or `null` for empty input, an
 * unlisted host, or any URL that doesn't match the repo pattern. PR URLs
 * (`/pull/<n>`) intentionally do NOT match here.
 */
export function parseGitHubRepoUrl(
  url: string,
  hosts: readonly string[] = DEFAULT_GITHUB_HOSTS,
): { host: string; owner: string; repo: string } | null {
  const match = url.trim().match(REPO_URL_RE);
  if (!match) return null;
  const host = listedGitHubHost(match[1], hosts);
  if (!host) return null;
  return { host, owner: match[2], repo: match[3] };
}

/** A pull-request or issue URL of `kind` on github.com or one of `hosts`. */
export function parseGitHubItemUrl(url: string, kind: "pull" | "issues", hosts: readonly string[]) {
  const match = url.trim().match(ITEM_URL_RE);
  if (!match || match[4] !== kind) return null;
  const host = listedGitHubHost(match[1], hosts);
  if (!host) return null;
  return { host, owner: match[2], repo: match[3], number: parseInt(match[5], 10) };
}

/** Host of a GitHub pull-request or issue URL on a host that is neither github.com nor in `hosts`. */
export function unconnectedGitHubHost(raw: string, hosts: readonly string[]): string | null {
  const match = raw.trim().match(ITEM_URL_RE);
  const host = match && normalizeGitHubHost(match[1]);
  return host && host !== DEFAULT_GITHUB_HOST && !hosts.includes(host) ? host : null;
}
