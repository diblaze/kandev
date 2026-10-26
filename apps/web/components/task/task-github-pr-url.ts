import { DEFAULT_GITHUB_HOST, githubWebOrigin, hostFromProviderHost } from "@/lib/github/host";
import {
  DEFAULT_GITHUB_HOSTS,
  GITHUB_URL_HOST_PATTERN,
  listedGitHubHost,
} from "@/lib/github/parse-url";
import type { Repository } from "@/lib/types/http";

export type TaskPullRequestLinkTarget = {
  id: string;
  title: string;
  repositoryId?: string;
  repositories?: Array<{ repository_id: string }>;
};

export type GitHubPRRepoRef = {
  /** GitHub host; absent means github.com. */
  host?: string;
  owner: string;
  repo: string;
  repositoryId: string;
};

export type ParsedPullRequestURL = {
  /** GitHub host; absent means github.com. */
  host?: string;
  owner: string;
  repo: string;
  number: number;
  url: string;
};

export function githubReposForTask(
  task: TaskPullRequestLinkTarget,
  repositories: Repository[],
): GitHubPRRepoRef[] {
  const repoIds = new Set((task.repositories ?? []).map((repo) => repo.repository_id));
  if (task.repositoryId) repoIds.add(task.repositoryId);

  return repositories
    .filter((repo) => repoIds.has(repo.id) && repo.provider === "github")
    .filter((repo) => hostFromProviderHost(repo.provider_host) !== null)
    .map((repo) =>
      withHost(hostFromProviderHost(repo.provider_host), {
        owner: repo.provider_owner,
        repo: repo.provider_name,
        repositoryId: repo.id,
      }),
    )
    .filter((repo) => repo.owner && repo.repo);
}

const PULL_REQUEST_URL_RE = new RegExp(
  String.raw`${GITHUB_URL_HOST_PATTERN}\/([^/\s]+)\/([^/\s]+)\/pull\/(\d+)(?:[/?#].*)?$`,
);

function withHost<T extends object>(host: string | null, value: T): T & { host?: string } {
  return host && host !== DEFAULT_GITHUB_HOST ? { host, ...value } : value;
}

function pullRequestURL(host: string | undefined, owner: string, repo: string, number: string) {
  return `${githubWebOrigin(host)}/${owner}/${repo}/pull/${number}`;
}

export function parseGitHubPullRequestURL(
  input: string,
  hosts: readonly string[] = DEFAULT_GITHUB_HOSTS,
): ParsedPullRequestURL | null {
  const match = input.trim().match(PULL_REQUEST_URL_RE);
  if (!match) return null;
  const [, rawHost, owner, repo, number] = match;
  const host = listedGitHubHost(rawHost, hosts);
  if (!host) return null;
  const parsedNumber = Number(number);
  if (!Number.isSafeInteger(parsedNumber) || parsedNumber <= 0) return null;
  return withHost(host, {
    owner,
    repo,
    number: parsedNumber,
    url: pullRequestURL(host, owner, repo, String(parsedNumber)),
  });
}

function positivePullRequestNumber(input: string): string | null {
  const number = input.replace(/^#/, "");
  if (!/^\d+$/.test(number)) return null;
  const parsedNumber = Number(number);
  if (!Number.isSafeInteger(parsedNumber) || parsedNumber <= 0) return null;
  return String(parsedNumber);
}

export function pullRequestPayload(
  input: string,
  githubRepos: GitHubPRRepoRef[],
  hosts: readonly string[] = DEFAULT_GITHUB_HOSTS,
) {
  const trimmed = input.trim();
  const inferredNumber = positivePullRequestNumber(trimmed);
  const inferredRepo = inferredNumber && githubRepos.length === 1 ? githubRepos[0] : null;
  if (inferredRepo && inferredNumber) {
    return {
      pr_url: pullRequestURL(
        inferredRepo.host,
        inferredRepo.owner,
        inferredRepo.repo,
        inferredNumber,
      ),
      repository_id: inferredRepo.repositoryId,
    };
  }

  const parsed = parseGitHubPullRequestURL(trimmed, hosts);
  const matchingRepo = parsed
    ? githubRepos.find(
        (repo) =>
          (repo.host ?? DEFAULT_GITHUB_HOST) === (parsed.host ?? DEFAULT_GITHUB_HOST) &&
          repo.owner.toLowerCase() === parsed.owner.toLowerCase() &&
          repo.repo.toLowerCase() === parsed.repo.toLowerCase(),
      )
    : null;

  return {
    pr_url: parsed?.url ?? trimmed,
    repository_id: matchingRepo?.repositoryId,
  };
}
