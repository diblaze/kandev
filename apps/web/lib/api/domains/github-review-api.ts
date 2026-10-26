import { githubWorkspaceQuery } from "@/lib/github/host";
import { fetchJson } from "../client";
import type { GitHubPRRef } from "./github-pr-api";

export async function requestPRReviewers(
  ref: GitHubPRRef,
  reviewers: string[],
  workspaceId: string,
) {
  const query = githubWorkspaceQuery(workspaceId, ref.host);
  return fetchJson<{ requested: boolean }>(
    `/api/v1/github/prs/${ref.owner}/${ref.repo}/${ref.number}/requested-reviewers?${query}`,
    {
      init: {
        method: "POST",
        body: JSON.stringify({ reviewers }),
      },
    },
  );
}
