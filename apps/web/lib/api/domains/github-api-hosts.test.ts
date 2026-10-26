import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/config", () => ({
  getBackendConfig: () => ({ apiBaseUrl: "http://api.test" }),
}));

import {
  disconnectGitHubPersonal,
  disconnectGitHubWorkspace,
  fetchAccessibleRepos,
  fetchGitHubStatus,
  fetchIssueInfo,
  fetchPRInfo,
  fetchRepoBranches,
  getPRFeedback,
  getPRStatus,
  getPRStatusesBatch,
  getRepoMergeAccess,
  getRepoMergeMethods,
  linkTaskIssue,
  listUserOrgs,
  mergePR,
  searchOrgRepos,
  searchUserIssues,
  searchUserPRs,
  startGitHubPersonalConnect,
  submitPRReview,
} from "./github-api";
import { requestPRReviewers } from "./github-review-api";

const WS = "ws-1";
const ENTERPRISE = "octo.ghe.com";
const fetchSpy = vi.fn<(...args: Parameters<typeof fetch>) => Promise<Response>>();

beforeEach(() => {
  fetchSpy.mockReset();
  fetchSpy.mockImplementation(async () => new Response("{}", { status: 200 }));
  vi.stubGlobal("fetch", fetchSpy);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function urls() {
  return fetchSpy.mock.calls.map((call) => String(call[0]));
}

function bodyAt(index: number) {
  return JSON.parse(String(fetchSpy.mock.calls[index]?.[1]?.body));
}

describe("GitHub host on API requests", () => {
  it("names an enterprise host on status and connection lifecycle requests", async () => {
    await fetchGitHubStatus(WS, undefined, false, ENTERPRISE);
    await disconnectGitHubWorkspace(WS, ENTERPRISE);
    await startGitHubPersonalConnect(WS, ENTERPRISE);
    await disconnectGitHubPersonal(WS, ENTERPRISE);

    expect(urls()).toEqual([
      "http://api.test/api/v1/github/status?workspace_id=ws-1&host=octo.ghe.com",
      "http://api.test/api/v1/github/workspace-connection?workspace_id=ws-1&host=octo.ghe.com",
      "http://api.test/api/v1/github/personal-connection/start?host=octo.ghe.com",
      "http://api.test/api/v1/github/personal-connection?workspace_id=ws-1&host=octo.ghe.com",
    ]);
  });

  it("names an enterprise host on repository and pull-request requests", async () => {
    const host = { host: ENTERPRISE };
    await fetchPRInfo(WS, "org", "repo", 7, host);
    await fetchIssueInfo(WS, "org", "repo", 8, host);
    await fetchRepoBranches(WS, "org", "repo", host);
    await getPRFeedback(WS, "org", "repo", 7, host);
    await getPRStatus(WS, "org", "repo", 7, host);
    await getRepoMergeMethods(WS, "org", "repo", host);
    await getRepoMergeAccess(WS, "org", "repo", host);
    await submitPRReview(WS, { owner: "org", repo: "repo", number: 7, ...host }, "APPROVE");
    await mergePR(WS, { owner: "org", repo: "repo", number: 7, ...host }, "squash");
    await requestPRReviewers({ owner: "org", repo: "repo", number: 7, ...host }, ["bob"], WS);

    const suffix = "workspace_id=ws-1&host=octo.ghe.com";
    expect(urls()).toEqual([
      `http://api.test/api/v1/github/prs/org/repo/7/info?${suffix}`,
      `http://api.test/api/v1/github/issues/org/repo/8/info?${suffix}`,
      `http://api.test/api/v1/github/repos/org/repo/branches?${suffix}`,
      `http://api.test/api/v1/github/prs/org/repo/7?${suffix}`,
      `http://api.test/api/v1/github/prs/org/repo/7/status?${suffix}`,
      `http://api.test/api/v1/github/repos/org/repo/merge-methods?${suffix}`,
      `http://api.test/api/v1/github/repos/org/repo/merge-access?${suffix}`,
      `http://api.test/api/v1/github/prs/org/repo/7/reviews?${suffix}`,
      `http://api.test/api/v1/github/prs/org/repo/7/merge?${suffix}`,
      `http://api.test/api/v1/github/prs/org/repo/7/requested-reviewers?${suffix}`,
    ]);
  });

  it("names an enterprise host on batch statuses and issue links in the body", async () => {
    await getPRStatusesBatch(WS, [{ owner: "org", repo: "repo", number: 7 }], {
      host: ENTERPRISE,
    });
    await linkTaskIssue("task-1", { issue: "#8", owner: "org", repo: "repo", host: ENTERPRISE });

    expect(bodyAt(0)).toEqual({
      workspace_id: WS,
      host: ENTERPRISE,
      refs: [{ owner: "org", repo: "repo", number: 7 }],
    });
    expect(bodyAt(1)).toMatchObject({ host: ENTERPRISE });
  });

  it("names an enterprise host on listing and search requests and tags repositories", async () => {
    fetchSpy.mockImplementationOnce(
      async () =>
        new Response(
          JSON.stringify({
            repos: [
              {
                owner: "org",
                name: "repo",
                full_name: "org/repo",
                default_branch: "main",
                private: true,
              },
            ],
          }),
          { status: 200 },
        ),
    );
    const repos = await fetchAccessibleRepos({ workspaceId: WS, host: ENTERPRISE });
    await listUserOrgs(WS, { host: ENTERPRISE });
    await searchOrgRepos(WS, "org", "re", { host: ENTERPRISE });
    await searchUserPRs({ workspaceId: WS, filter: "is:open", host: ENTERPRISE });
    await searchUserIssues({ workspaceId: WS, filter: "is:open", host: ENTERPRISE });

    expect(repos[0]).toMatchObject({ provider: "github", host: ENTERPRISE });
    for (const url of urls()) expect(new URL(url).searchParams.get("host")).toBe(ENTERPRISE);
  });

  it("omits the host for github.com so existing requests are unchanged", async () => {
    await fetchPRInfo(WS, "org", "repo", 7, { host: "github.com" });
    await fetchGitHubStatus(WS, undefined, false, "github.com");
    await disconnectGitHubWorkspace(WS);
    const repos = await fetchAccessibleRepos({ workspaceId: WS }).catch(() => []);

    expect(urls().slice(0, 3)).toEqual([
      "http://api.test/api/v1/github/prs/org/repo/7/info?workspace_id=ws-1",
      "http://api.test/api/v1/github/status?workspace_id=ws-1",
      "http://api.test/api/v1/github/workspace-connection?workspace_id=ws-1",
    ]);
    expect(repos).toEqual([]);
  });

  it("forwards a malformed host instead of falling back to github.com", async () => {
    await fetchPRInfo(WS, "org", "repo", 7, { host: "evil.test/x" });
    expect(new URL(urls()[0]).searchParams.get("host")).toBe("evil.test/x");
  });
});
