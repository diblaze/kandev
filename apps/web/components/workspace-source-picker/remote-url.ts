import { DEFAULT_GITHUB_HOSTS, listedGitHubHost } from "@/lib/github/parse-url";

/** Matches the task-create picker’s URL-shape check, including scheme-less URLs. */
export function looksLikeURL(value: string): boolean {
  if (!value) return false;
  const withScheme = /^[a-z][a-z\d+.-]*:\/\//i.test(value) ? value : `https://${value}`;
  try {
    const parsed = new URL(withScheme);
    return parsed.hostname.includes(".") && value.includes("/");
  } catch {
    return false;
  }
}

/**
 * Supported provider-host restriction shared by task creation and workspace
 * source attachment. SCP Git syntax is valid only for the same providers; a
 * connected GitHub host accepts any SCP user, as GHE.com tenants use their own.
 */
export function looksLikeSupportedRemoteURL(
  value: string,
  githubHosts: readonly string[] = DEFAULT_GITHUB_HOSTS,
): boolean {
  if (/^git@(github\.com|gitlab\.com|ssh\.dev\.azure\.com):\S+$/i.test(value)) return true;
  const scp = value.match(/^[A-Za-z0-9_.-]+@([^:/\s]+):\S+$/);
  if (scp) return listedGitHubHost(scp[1], githubHosts) !== null;
  if (!looksLikeURL(value)) return false;
  const candidate = /^[a-z][a-z\d+.-]*:\/\//i.test(value) ? value : `https://${value}`;
  try {
    const host = new URL(candidate).hostname.toLowerCase();
    return (
      host === "gitlab.com" ||
      host === "dev.azure.com" ||
      listedGitHubHost(host, githubHosts) !== null
    );
  } catch {
    return false;
  }
}
