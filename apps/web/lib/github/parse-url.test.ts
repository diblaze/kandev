import { describe, expect, it } from "vitest";
import { listedGitHubHost, parseGitHubRepoUrl, unconnectedGitHubHost } from "./parse-url";

const GITHUB_COM = "github.com";

describe("parseGitHubRepoUrl", () => {
  it("parses a plain https URL", () => {
    expect(parseGitHubRepoUrl("https://github.com/acme/site")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("parses an http URL", () => {
    expect(parseGitHubRepoUrl("http://github.com/acme/site")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("parses a URL without a scheme", () => {
    expect(parseGitHubRepoUrl("github.com/acme/site")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("parses a URL with www.", () => {
    expect(parseGitHubRepoUrl("https://www.github.com/acme/site")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("strips a .git suffix", () => {
    expect(parseGitHubRepoUrl("https://github.com/acme/site.git")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("tolerates a trailing slash", () => {
    expect(parseGitHubRepoUrl("https://github.com/acme/site/")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("tolerates a .git suffix and trailing slash together", () => {
    expect(parseGitHubRepoUrl("https://github.com/acme/site.git/")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("tolerates a query string suffix (canonical share URL)", () => {
    // The GitHub UI pins active tabs via ?tab=… and similar; the picker
    // should accept these without stripping the tail by hand.
    expect(parseGitHubRepoUrl("https://github.com/acme/site?tab=readme")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("tolerates a fragment suffix", () => {
    expect(parseGitHubRepoUrl("https://github.com/acme/site#readme")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("tolerates a trailing slash followed by a fragment", () => {
    expect(parseGitHubRepoUrl("https://github.com/acme/site/#section")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });

  it("accepts hyphens, dots, and underscores in owner and repo", () => {
    expect(parseGitHubRepoUrl("https://github.com/my-org_1.x/repo-name_2.x")).toEqual({
      host: GITHUB_COM,
      owner: "my-org_1.x",
      repo: "repo-name_2.x",
    });
  });

  it("trims surrounding whitespace before matching", () => {
    expect(parseGitHubRepoUrl("   https://github.com/acme/site   ")).toEqual({
      host: GITHUB_COM,
      owner: "acme",
      repo: "site",
    });
  });
});

describe("parseGitHubRepoUrl rejections", () => {
  it("returns null for empty input", () => {
    expect(parseGitHubRepoUrl("")).toBeNull();
    expect(parseGitHubRepoUrl("   ")).toBeNull();
  });

  it("returns null for a non-GitHub host", () => {
    expect(parseGitHubRepoUrl("https://gitlab.com/acme/site")).toBeNull();
  });

  it("returns null for a malformed URL", () => {
    expect(parseGitHubRepoUrl("https://github.com/acme")).toBeNull();
    expect(parseGitHubRepoUrl("not a url at all")).toBeNull();
  });

  it("returns null when the URL is embedded in surrounding text", () => {
    // Regression: the regex was unanchored, so anything containing a valid
    // github.com/<owner>/<repo> substring (chat messages, log lines, etc.)
    // matched and got treated as a clean repo URL. Anchoring at ^ rules out
    // these false positives — input must START with the URL pattern.
    expect(parseGitHubRepoUrl("some prefix text github.com/owner/repo")).toBeNull();
    expect(parseGitHubRepoUrl("see https://github.com/owner/repo for details")).toBeNull();
    expect(parseGitHubRepoUrl("malicious.com/github.com/owner/repo")).toBeNull();
  });
});

const CONNECTED = [GITHUB_COM, "octo.ghe.com"];

describe("parseGitHubRepoUrl hosts", () => {
  it("parses a repository URL on a connected enterprise host", () => {
    expect(parseGitHubRepoUrl("https://octo.ghe.com/org/repo.git", CONNECTED)).toEqual({
      host: "octo.ghe.com",
      owner: "org",
      repo: "repo",
    });
  });

  it("returns null for a repository URL on a host that is not connected", () => {
    expect(parseGitHubRepoUrl("https://ghe.example.com/org/repo", CONNECTED)).toBeNull();
    expect(parseGitHubRepoUrl("https://octo.ghe.com/org/repo")).toBeNull();
  });
});

describe("listedGitHubHost", () => {
  it("accepts github.com without a connection list and listed hosts", () => {
    expect(listedGitHubHost(GITHUB_COM, [])).toBe(GITHUB_COM);
    expect(listedGitHubHost("www.github.com", [])).toBe(GITHUB_COM);
    expect(listedGitHubHost(CONNECTED[1], CONNECTED)).toBe(CONNECTED[1]);
    expect(listedGitHubHost("OCTO.ghe.com", CONNECTED)).toBe(CONNECTED[1]);
  });

  it("rejects hosts that are not listed", () => {
    expect(listedGitHubHost("ghe.example.com", CONNECTED)).toBeNull();
    expect(listedGitHubHost("gitlab.com", [GITHUB_COM])).toBeNull();
  });
});

describe("unconnectedGitHubHost", () => {
  it("names the host of a GitHub-shaped URL on a host that is not connected", () => {
    expect(unconnectedGitHubHost("https://ghe.example.com/org/repo/pull/7", CONNECTED)).toBe(
      "ghe.example.com",
    );
    expect(unconnectedGitHubHost("ghe.example.com/org/repo/issues/3#top", [])).toBe(
      "ghe.example.com",
    );
  });

  it("is null for connected hosts and for input that is not a GitHub URL", () => {
    expect(unconnectedGitHubHost("https://octo.ghe.com/org/repo/pull/7", CONNECTED)).toBeNull();
    expect(unconnectedGitHubHost("https://github.com/org/repo/issues/7", [])).toBeNull();
    expect(unconnectedGitHubHost("#7", CONNECTED)).toBeNull();
    expect(
      unconnectedGitHubHost("https://gitlab.example.com/org/repo/-/merge_requests/7", []),
    ).toBeNull();
    expect(unconnectedGitHubHost("https://ghe.example.com/org/repo", [])).toBeNull();
    expect(unconnectedGitHubHost("not a url", [])).toBeNull();
  });
});
