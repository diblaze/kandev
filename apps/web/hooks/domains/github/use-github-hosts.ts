"use client";

import { useCallback, useMemo } from "react";
import { useOptionalAppStore } from "@/components/state-provider";
import { DEFAULT_GITHUB_HOST } from "@/lib/github/host";
import type { AppState } from "@/lib/state/store";
import type { GitHubHostStatus } from "@/lib/types/github";

const NO_HOSTS: GitHubHostStatus[] = [];

/** github.com first, then every other host that has a connection. */
function githubHostsFromConnections(hosts: readonly GitHubHostStatus[] | undefined) {
  const enterprise = (hosts ?? [])
    .map((entry) => entry.github_host)
    .filter((host) => host && host !== DEFAULT_GITHUB_HOST);
  return [DEFAULT_GITHUB_HOST, ...new Set(enterprise)];
}

/**
 * GitHub hosts whose URLs the workspace recognizes: github.com first, then every
 * other connected host. Reads the workspace status already loaded by
 * `useGitHubStatus`, so it never issues a request, and outside a store it
 * recognizes github.com only.
 */
export function useGitHubHosts(workspaceId?: string | null): string[] {
  const selectHosts = useCallback(
    (state: AppState) => {
      const id = workspaceId ?? state.workspaces.activeId;
      return (id && state.githubStatus.byWorkspaceId[id]?.status?.hosts) || NO_HOSTS;
    },
    [workspaceId],
  );
  const hosts = useOptionalAppStore(selectHosts, NO_HOSTS);
  const hostsKey = githubHostsFromConnections(hosts).join(",");
  return useMemo(() => hostsKey.split(","), [hostsKey]);
}
