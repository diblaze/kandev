import { DEFAULT_GITHUB_HOST, githubWebOrigin } from "@/lib/github/host";
import { DEFAULT_GITHUB_HOSTS, listedGitHubHost } from "@/lib/github/parse-url";

export type ParsedGitHubRepoUrl = {
  /** GitHub host; absent means github.com. */
  host?: string;
  owner: string;
  repo: string;
  branch?: string;
  path?: string;
};

const SSH_URL_RE = /^[A-Za-z0-9_.-]+@([^:/\s]+):([^/\s]+)\/([^/\s]+?)(?:\.git)?\/?$/;

function withHost(host: string, parts: Omit<ParsedGitHubRepoUrl, "host">): ParsedGitHubRepoUrl {
  return host === DEFAULT_GITHUB_HOST ? parts : { host, ...parts };
}

// parseGitHubRepoUrl extracts owner/repo (and branch + directory for
// /tree/... and /blob/... links) from a pasted GitHub URL. Returns null when
// the input isn't a recognizable GitHub repository link.
//
// Branch names containing "/" cannot be told apart from the leading path
// segments without asking the GitHub API, so single-segment branch names are
// assumed. A /blob/ link to a file resolves to the file's directory.
export function parseGitHubRepoUrl(
  input: string,
  hosts: readonly string[] = DEFAULT_GITHUB_HOSTS,
): ParsedGitHubRepoUrl | null {
  const raw = input.trim();
  if (!raw) return null;

  const ssh = raw.match(SSH_URL_RE);
  if (ssh) {
    const host = listedGitHubHost(ssh[1], hosts);
    return host ? withHost(host, { owner: ssh[2], repo: ssh[3] }) : null;
  }

  let url: URL;
  try {
    url = new URL(raw.includes("://") ? raw : `https://${raw}`);
  } catch {
    return null;
  }
  const host = url.port ? null : listedGitHubHost(url.hostname, hosts);
  if (!host) return null;

  let segments: string[];
  try {
    segments = url.pathname.split("/").filter(Boolean).map(decodeURIComponent);
  } catch {
    // Malformed percent escapes (e.g. a trailing "%") must read as "not a
    // recognized link", not crash the dialog on every keystroke.
    return null;
  }
  if (segments.length < 2) return null;
  const [owner, rawRepo, ...rest] = segments;
  const repo = rawRepo.replace(/\.git$/, "");
  if (!owner || !repo) return null;
  return withHost(host, { owner, repo, ...parseBranchAndPath(rest) });
}

// buildGitHubRepoUrl renders a stored config back into a canonical GitHub
// link (the inverse of parseGitHubRepoUrl) so the settings form can show one
// URL instead of separate owner/repo/path fields.
export function buildGitHubRepoUrl(parts: ParsedGitHubRepoUrl): string {
  const base = `${githubWebOrigin(parts.host)}/${parts.owner}/${parts.repo}`;
  if (!parts.branch) return base;
  const path = parts.path ? `/${parts.path.split("/").map(encodeURIComponent).join("/")}` : "";
  return `${base}/tree/${encodeURIComponent(parts.branch)}${path}`;
}

function parseBranchAndPath(segments: string[]): Pick<ParsedGitHubRepoUrl, "branch" | "path"> {
  const [marker, branch, ...rest] = segments;
  if ((marker !== "tree" && marker !== "blob") || !branch) return {};
  const pathSegments = marker === "blob" ? rest.slice(0, -1) : rest;
  if (pathSegments.length === 0) return { branch };
  return { branch, path: pathSegments.join("/") };
}
