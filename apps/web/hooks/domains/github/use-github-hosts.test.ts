import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { StateProvider } from "@/components/state-provider";
import type { GitHubHostStatus, GitHubStatus } from "@/lib/types/github";
import { useGitHubHosts } from "./use-github-hosts";

const GITHUB_COM = "github.com";

const fetchGitHubStatusMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api/domains/github-api", () => ({
  fetchGitHubStatus: fetchGitHubStatusMock,
}));

const WORKSPACE = "ws-1";

afterEach(() => {
  fetchGitHubStatusMock.mockReset();
});

function hostStatus(github_host: string): GitHubHostStatus {
  return { github_host, source: "pat", login: "octocat", status: "active" };
}

function wrapperFor(hosts: GitHubHostStatus[] | undefined) {
  const status = {
    workspace_id: WORKSPACE,
    authenticated: true,
    username: "octocat",
    auth_method: "pat",
    token_configured: true,
    required_scopes: [],
    hosts,
  } as GitHubStatus;
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(StateProvider, {
      initialState: {
        githubStatus: {
          byWorkspaceId: { [WORKSPACE]: { status, loaded: true, loading: false } },
          pendingPRDiscoveryHealthByWorkspaceId: {},
        },
      },
      children,
    });
  };
}

describe("useGitHubHosts", () => {
  it("lists github.com when the workspace has no host connection", () => {
    const { result } = renderHook(() => useGitHubHosts(WORKSPACE), { wrapper: wrapperFor([]) });
    expect(result.current).toEqual([GITHUB_COM]);
  });

  it("lists github.com once when only github.com is connected", () => {
    const { result } = renderHook(() => useGitHubHosts(WORKSPACE), {
      wrapper: wrapperFor([hostStatus(GITHUB_COM)]),
    });
    expect(result.current).toEqual([GITHUB_COM]);
  });

  it("lists every connected host after github.com without a second status request", () => {
    const { result } = renderHook(() => useGitHubHosts(WORKSPACE), {
      wrapper: wrapperFor([
        hostStatus("ghe.example.com"),
        hostStatus(GITHUB_COM),
        hostStatus("octo.ghe.com"),
      ]),
    });
    expect(result.current).toEqual([GITHUB_COM, "ghe.example.com", "octo.ghe.com"]);
    expect(fetchGitHubStatusMock).not.toHaveBeenCalled();
  });
});
