import { describe, expect, it } from "vitest";
import { findDuplicateRemoteRepo } from "./task-create-dialog-helpers";
import type { TaskRemoteRepoRow } from "@/components/task-create-dialog-types";

function remoteRow(key: string, url: string, branch = ""): TaskRemoteRepoRow {
  return { key, url, branch, source: "paste" };
}

describe("findDuplicateRemoteRepo GitHub hosts", () => {
  const hosts = ["github.com", "octo.ghe.com"];

  it("does not flag the same owner and repository on two hosts", () => {
    expect(
      findDuplicateRemoteRepo(
        [
          remoteRow("r0", "https://github.com/org/api"),
          remoteRow("r1", "https://octo.ghe.com/org/api"),
        ],
        hosts,
      ),
    ).toBeNull();
  });

  it("flags two pull requests of one enterprise repository", () => {
    expect(
      findDuplicateRemoteRepo(
        [
          remoteRow("r0", "https://octo.ghe.com/org/api/pull/1"),
          remoteRow("r1", "https://octo.ghe.com/org/api/pull/2"),
        ],
        hosts,
      ),
    ).toBe("org/api");
  });
});
