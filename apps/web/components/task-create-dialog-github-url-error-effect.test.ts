import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { useGitHubUrlErrorEffect } from "./task-create-dialog-effects";
import type { DialogFormState } from "./task-create-dialog-types";

type UrlErrorFake = {
  useRemote?: boolean;
  remoteRepos?: Array<{ key: string; url: string; branch: string; source: "paste" | "picker" }>;
  setGitHubUrlError?: ReturnType<typeof vi.fn>;
  githubHosts?: readonly string[];
};

function makeUrlErrorFs(overrides: UrlErrorFake = {}): DialogFormState {
  const remoteRepos = overrides.remoteRepos ?? [
    { key: "remote-0", url: "", branch: "", source: "paste" as const },
  ];
  return {
    useRemote: overrides.useRemote ?? true,
    setGitHubUrlError: overrides.setGitHubUrlError ?? vi.fn(),
    remoteRepos,
    githubHosts: overrides.githubHosts,
  } as unknown as DialogFormState;
}

describe("useGitHubUrlErrorEffect", () => {
  it("surfaces 'Invalid GitHub URL' for an unparseable first-row URL", () => {
    const setGitHubUrlError = vi.fn();
    const fs = makeUrlErrorFs({
      remoteRepos: [{ key: "remote-0", url: "not a url", branch: "", source: "paste" }],
      setGitHubUrlError,
    });
    renderHook(() => useGitHubUrlErrorEffect(fs, true));
    expect(setGitHubUrlError).toHaveBeenCalledWith(expect.stringContaining("Invalid GitHub URL"));
  });

  it("clears the error for a valid repo URL", () => {
    const setGitHubUrlError = vi.fn();
    const fs = makeUrlErrorFs({
      remoteRepos: [
        { key: "remote-0", url: "https://github.com/acme/site", branch: "", source: "paste" },
      ],
      setGitHubUrlError,
    });
    renderHook(() => useGitHubUrlErrorEffect(fs, true));
    expect(setGitHubUrlError).toHaveBeenLastCalledWith(null);
  });

  it("clears the error for an empty URL", () => {
    const setGitHubUrlError = vi.fn();
    const fs = makeUrlErrorFs({
      remoteRepos: [{ key: "remote-0", url: "", branch: "", source: "paste" }],
      setGitHubUrlError,
    });
    renderHook(() => useGitHubUrlErrorEffect(fs, true));
    expect(setGitHubUrlError).toHaveBeenCalledWith(null);
  });

  it("clears stale errors when useRemote is false", () => {
    const setGitHubUrlError = vi.fn();
    const fs = makeUrlErrorFs({
      useRemote: false,
      remoteRepos: [{ key: "remote-0", url: "not a url", branch: "", source: "paste" }],
      setGitHubUrlError,
    });
    renderHook(() => useGitHubUrlErrorEffect(fs, true));
    expect(setGitHubUrlError).toHaveBeenCalledWith(null);
  });
});

describe("useGitHubUrlErrorEffect GitHub hosts", () => {
  const enterpriseRows = [
    {
      key: "remote-0",
      url: "https://octo.ghe.com/org/api/pull/7",
      branch: "",
      source: "paste" as const,
    },
  ];

  it("accepts a URL on a connected enterprise host", () => {
    const setGitHubUrlError = vi.fn();
    const fs = makeUrlErrorFs({
      remoteRepos: enterpriseRows,
      setGitHubUrlError,
      githubHosts: ["github.com", "octo.ghe.com"],
    });
    renderHook(() => useGitHubUrlErrorEffect(fs, true));
    expect(setGitHubUrlError).toHaveBeenLastCalledWith(null);
  });

  it("flags a URL on a host that is not connected", () => {
    const setGitHubUrlError = vi.fn();
    const fs = makeUrlErrorFs({ remoteRepos: enterpriseRows, setGitHubUrlError });
    renderHook(() => useGitHubUrlErrorEffect(fs, true));
    expect(setGitHubUrlError).toHaveBeenLastCalledWith(
      expect.stringContaining("Invalid GitHub URL"),
    );
  });
});
