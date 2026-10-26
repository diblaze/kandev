import type {
  GitHubPRStatus,
  MergeMethod,
  PRFeedback,
  RepoMergeAccess,
  RepoMergeMethods,
} from "@/lib/types/github";
import { githubWorkspaceQuery, setGitHubHostParam } from "@/lib/github/host";
import { fetchJson, type ApiRequestOptions } from "../client";

export type PRStatusRef = { owner: string; repo: string; number: number };
/** A pull request on a GitHub host; an absent host means github.com. */
export type GitHubPRRef = PRStatusRef & { host?: string };
/** Request options naming the GitHub host of the item; an absent host means github.com. */
export type GitHubHostOptions = ApiRequestOptions & { host?: string };
export type PRReviewEvent = "APPROVE" | "COMMENT" | "REQUEST_CHANGES";

export async function getPRFeedback(
  workspaceId: string,
  owner: string,
  repo: string,
  number: number,
  options?: GitHubHostOptions,
) {
  return fetchJson<PRFeedback>(
    `/api/v1/github/prs/${owner}/${repo}/${number}?${githubWorkspaceQuery(workspaceId, options?.host)}`,
    options,
  );
}

export async function getPRStatus(
  workspaceId: string,
  owner: string,
  repo: string,
  number: number,
  options?: GitHubHostOptions,
) {
  return fetchJson<GitHubPRStatus>(
    `/api/v1/github/prs/${owner}/${repo}/${number}/status?${githubWorkspaceQuery(workspaceId, options?.host)}`,
    options,
  );
}

export async function getPRStatusesBatch(
  workspaceId: string,
  refs: PRStatusRef[],
  options?: GitHubHostOptions,
) {
  const host = setGitHubHostParam(new URLSearchParams(), options?.host).get("host");
  return fetchJson<{ statuses: Record<string, GitHubPRStatus> }>(`/api/v1/github/prs/statuses`, {
    ...options,
    init: {
      method: "POST",
      body: JSON.stringify({ workspace_id: workspaceId, ...(host ? { host } : {}), refs }),
      ...(options?.init ?? {}),
    },
  });
}

export async function submitPRReview(
  workspaceId: string,
  ref: GitHubPRRef,
  event: PRReviewEvent,
  body?: string,
) {
  return fetchJson<{ submitted: boolean }>(
    `/api/v1/github/prs/${ref.owner}/${ref.repo}/${ref.number}/reviews?${githubWorkspaceQuery(workspaceId, ref.host)}`,
    {
      init: {
        method: "POST",
        body: JSON.stringify({ event, body: body ?? "" }),
      },
    },
  );
}

export async function mergePR(workspaceId: string, ref: GitHubPRRef, mergeMethod?: MergeMethod) {
  return fetchJson<{ status: "merged" | "queued" }>(
    `/api/v1/github/prs/${ref.owner}/${ref.repo}/${ref.number}/merge?${githubWorkspaceQuery(workspaceId, ref.host)}`,
    {
      init: {
        method: "PUT",
        body: JSON.stringify({ merge_method: mergeMethod ?? "" }),
      },
    },
  );
}

export async function getRepoMergeMethods(
  workspaceId: string,
  owner: string,
  repo: string,
  options?: GitHubHostOptions,
) {
  return fetchJson<RepoMergeMethods>(
    `/api/v1/github/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/merge-methods?${githubWorkspaceQuery(workspaceId, options?.host)}`,
    options,
  );
}

export async function getRepoMergeAccess(
  workspaceId: string,
  owner: string,
  repo: string,
  options?: GitHubHostOptions,
) {
  return fetchJson<RepoMergeAccess>(
    `/api/v1/github/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/merge-access?${githubWorkspaceQuery(workspaceId, options?.host)}`,
    { ...options, cache: "no-store" },
  );
}
