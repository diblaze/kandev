import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { IconGitPullRequest } from "@tabler/icons-react";
import { buildParams, useGitHubSearch } from "./use-github-search";
import type { PresetOption } from "./search-bar";

const presets: PresetOption[] = [
  { value: "open", label: "Open", filter: "is:open", group: "inbox", icon: IconGitPullRequest },
  {
    value: "closed",
    label: "Closed",
    filter: "is:closed",
    group: "inbox",
    icon: IconGitPullRequest,
  },
];

describe("buildParams", () => {
  it("uses preset filter when custom query is empty", () => {
    expect(buildParams(presets, "open", "", "")).toEqual({ filter: "is:open" });
  });

  it("trims custom query and routes to query branch", () => {
    expect(buildParams(presets, "open", "  foo bar  ", "")).toEqual({ query: "foo bar" });
  });

  it("appends repo qualifier to custom query", () => {
    expect(buildParams(presets, "open", "foo", "acme/widget")).toEqual({
      query: "foo repo:acme/widget",
    });
  });

  it("appends repo qualifier to preset filter", () => {
    expect(buildParams(presets, "closed", "", "acme/widget")).toEqual({
      filter: "is:closed repo:acme/widget",
    });
  });

  it("falls back to empty filter when preset is unknown", () => {
    expect(buildParams(presets, "missing", "", "")).toEqual({ filter: "" });
  });

  it("returns repo-only filter when preset is unknown but repo given", () => {
    expect(buildParams(presets, "missing", "", "a/b")).toEqual({ filter: "repo:a/b" });
  });

  it("whitespace-only custom query treated as empty (uses preset)", () => {
    expect(buildParams(presets, "open", "   ", "")).toEqual({ filter: "is:open" });
  });
});

const searchMocks = vi.hoisted(() => ({
  searchUserPRs: vi.fn(),
  searchUserIssues: vi.fn(),
}));
vi.mock("@/lib/api/domains/github-api", () => searchMocks);

describe("useGitHubSearch hosts", () => {
  it("searches the selected GitHub host", async () => {
    searchMocks.searchUserPRs.mockResolvedValue({ prs: [], total_count: 0 });
    renderHook(() =>
      useGitHubSearch({
        kind: "pr",
        presets,
        preset: "open",
        customQuery: "",
        workspaceId: "ws-1",
        host: "octo.ghe.com",
      }),
    );

    await waitFor(() =>
      expect(searchMocks.searchUserPRs).toHaveBeenCalledWith(
        expect.objectContaining({ workspaceId: "ws-1", host: "octo.ghe.com" }),
      ),
    );
  });
});
