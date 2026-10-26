import { describe, expect, it } from "vitest";
import {
  githubSSHUser,
  githubWebOrigin,
  hostFromProviderHost,
  hostFromUrl,
  normalizeGitHubHost,
} from "./host";

const GITHUB_COM = "github.com";
const ENTERPRISE_HOST = "octo.ghe.com";
const SERVER_HOST = "ghe.example.com";

describe("normalizeGitHubHost", () => {
  it.each([
    ["", GITHUB_COM],
    [GITHUB_COM, GITHUB_COM],
    ["www.github.com", GITHUB_COM],
    [ENTERPRISE_HOST, ENTERPRISE_HOST],
    [SERVER_HOST, SERVER_HOST],
    ["  GHE.Example.COM ", SERVER_HOST],
  ])("normalizes %j to %s", (raw, want) => {
    expect(normalizeGitHubHost(raw)).toBe(want);
  });

  it.each(["https://x", "x/y", "x:443", "a@b", "-x", "a b", "a\tb"])("rejects %j", (raw) => {
    expect(normalizeGitHubHost(raw)).toBeNull();
  });
});

describe("githubWebOrigin", () => {
  it("builds the web origin of a host and defaults to github.com", () => {
    expect(githubWebOrigin(ENTERPRISE_HOST)).toBe("https://octo.ghe.com");
    expect(githubWebOrigin("")).toBe("https://github.com");
    expect(githubWebOrigin(undefined)).toBe("https://github.com");
  });
});

describe("hostFromUrl", () => {
  it.each([
    ["https://github.com/o/r/pull/1", GITHUB_COM],
    ["https://octo.ghe.com/o/r/issues/2", ENTERPRISE_HOST],
    ["https://GHE.Example.com/o/r/pull/3", SERVER_HOST],
  ])("reads the host of %s", (raw, want) => {
    expect(hostFromUrl(raw)).toBe(want);
  });

  it.each([
    "http://ghe.example.com/o/r",
    "https://user@ghe.example.com/o/r",
    "https://ghe.example.com:8443/o/r",
    "not a url",
    "",
  ])("rejects %j", (raw) => {
    expect(hostFromUrl(raw)).toBeNull();
  });
});

describe("hostFromProviderHost", () => {
  it.each([
    ["https://github.com", GITHUB_COM],
    ["https://octo.ghe.com", ENTERPRISE_HOST],
    [SERVER_HOST, SERVER_HOST],
    ["OCTO.ghe.com", ENTERPRISE_HOST],
    ["", GITHUB_COM],
    [undefined, GITHUB_COM],
  ])("reads the host of %j", (raw, want) => {
    expect(hostFromProviderHost(raw)).toBe(want);
  });

  it.each(["https://x/y", "http://ghe.example.com", "https://ghe.example.com:8443"])(
    "rejects %j",
    (raw) => {
      expect(hostFromProviderHost(raw)).toBeNull();
    },
  );
});

describe("githubSSHUser", () => {
  it("uses the tenant name on a GHE.com host and git elsewhere", () => {
    expect(githubSSHUser(ENTERPRISE_HOST)).toBe("octo");
    expect(githubSSHUser(GITHUB_COM)).toBe("git");
    expect(githubSSHUser(SERVER_HOST)).toBe("git");
  });
});
