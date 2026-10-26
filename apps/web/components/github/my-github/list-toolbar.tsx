"use client";

import { useTranslation } from "react-i18next";
import type { ReactNode } from "react";
import { IntegrationListToolbar } from "@/components/integrations/integration-list-toolbar";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@kandev/ui/select";
import { RepoFilterCombobox } from "./repo-filter-combobox";

type ListToolbarProps = {
  title: string;
  titleControl?: ReactNode;
  count: number;
  loading: boolean;
  lastFetchedAt: Date | null;
  customQuery: string;
  committedQuery: string;
  onCustomQueryChange: (value: string) => void;
  onCommitCustomQuery: () => void;
  repoFilter: string;
  onRepoFilterChange: (value: string) => void;
  repoOptions: string[];
  onRefresh: () => void;
  /** GitHub hosts to search; the selector shows only when there are two or more. */
  githubHosts?: readonly string[];
  githubHost?: string;
  onGitHubHostChange?: (host: string) => void;
};

function GitHubHostSelect({
  hosts,
  value,
  onChange,
}: {
  hosts: readonly string[];
  value: string;
  onChange: (host: string) => void;
}) {
  const { t } = useTranslation();
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger
        className="w-full text-xs md:w-[180px]"
        aria-label={t("github:githubHost")}
        data-testid="github-host-select"
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {hosts.map((host) => (
          <SelectItem key={host} value={host}>
            {host}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function ListToolbar({
  title,
  titleControl,
  count,
  loading,
  lastFetchedAt,
  customQuery,
  committedQuery,
  onCustomQueryChange,
  onCommitCustomQuery,
  repoFilter,
  onRepoFilterChange,
  repoOptions,
  onRefresh,
  githubHosts = [],
  githubHost = "",
  onGitHubHostChange,
}: ListToolbarProps) {
  const { t } = useTranslation();
  const repoFilterControl = (
    <RepoFilterCombobox
      repoFilter={repoFilter}
      onRepoFilterChange={onRepoFilterChange}
      repoOptions={repoOptions}
      ariaLabel={t("github:filterGithubResultsByRepository")}
      triggerClassName="w-full border border-input bg-background px-2 text-xs/relaxed hover:bg-secondary/50 md:w-[220px]"
      className="md:min-w-[360px]"
      testId="github-repo-filter-trigger"
      dropdownTestId="github-repo-filter-dropdown"
    />
  );
  return (
    <IntegrationListToolbar
      title={title}
      titleControl={titleControl}
      count={count}
      loading={loading}
      lastFetchedAt={lastFetchedAt}
      customQuery={customQuery}
      committedQuery={committedQuery}
      onCustomQueryChange={onCustomQueryChange}
      onCommitCustomQuery={onCommitCustomQuery}
      onRefresh={onRefresh}
      filter={
        githubHosts.length >= 2 && onGitHubHostChange ? (
          <div className="flex w-full flex-col gap-2 md:w-auto md:flex-row">
            <GitHubHostSelect
              hosts={githubHosts}
              value={githubHost}
              onChange={onGitHubHostChange}
            />
            {repoFilterControl}
          </div>
        ) : (
          repoFilterControl
        )
      }
      queryPlaceholder={t("github:customQueryPressEnterEG")}
      titleTestId="github-list-toolbar-title"
    />
  );
}
