"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Label } from "@kandev/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@kandev/ui/select";
import { fetchGitHubCLIAccounts } from "@/lib/api/domains/github-api";
import type { GitHubCLIAccount } from "@/lib/types/github";
import { Trans, useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

function GitHubCLIAccountNotice({
  loadError,
  loading,
  hasAccounts,
}: {
  loadError: string | null;
  loading: boolean;
  hasAccounts: boolean;
}) {
  const { t } = useTranslation();
  if (loading || hasAccounts) return null;
  if (loadError) {
    return (
      <p role="alert" className="text-xs text-destructive">
        {t("github:couldNotLoadGithubCliAccounts", { error: loadError })}
      </p>
    );
  }
  return (
    <p className="text-xs text-muted-foreground">
      {/* The command is a value, not copy: writing it into the catalog lets the
          pseudo-locale transliterate it into something the user cannot type. */}
      <Trans i18nKey="github:signInWithGhAuthLogin" values={{ command: "gh auth login" }}>
        Sign in with <code>{"{{command}}"}</code>, then reopen this dialog.
      </Trans>
    </p>
  );
}

// Plain function, so `t` is threaded in rather than taken from the hook. The
// guard never inspects this shape — `mode: "jsx-only"` only sees JSX literals.
function accountPlaceholder(t: TFunction, loading: boolean, loadError: string | null) {
  if (loading) return t("github:loadingAccounts");
  if (loadError) return t("github:accountsUnavailable");
  return t("github:noGhAccountsFound");
}

/**
 * An account on a host that already has a connection is taken, except on the
 * host being changed, so choosing it cannot silently replace another host.
 */
function cliAccountTaken(
  account: GitHubCLIAccount,
  connectedHosts: readonly string[],
  editedHost: string | undefined,
): boolean {
  return account.host !== editedHost && connectedHosts.includes(account.host);
}

function preferredAccount(
  items: GitHubCLIAccount[],
  connectedHosts: readonly string[],
  editedHost: string | undefined,
) {
  const available = items.filter((item) => !cliAccountTaken(item, connectedHosts, editedHost));
  return (
    available.find((item) => editedHost && item.host === editedHost && item.selected) ??
    available.find((item) => editedHost && item.host === editedHost && item.active) ??
    available.find((item) => item.selected) ??
    available.find((item) => item.active) ??
    available[0]
  );
}

export function GitHubCLIForm({
  workspaceId,
  onAccountChange,
  editedHost,
  connectedHosts = [],
  disabled,
}: {
  workspaceId: string;
  onAccountChange: (account: GitHubCLIAccount | null) => void;
  /** Host whose connection is being changed; absent when connecting a new host. */
  editedHost?: string;
  /** Hosts that already have a connection. */
  connectedHosts?: readonly string[];
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const [accounts, setAccounts] = useState<GitHubCLIAccount[]>([]);
  const [selected, setSelected] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const onAccountChangeRef = useRef(onAccountChange);
  onAccountChangeRef.current = onAccountChange;
  const connectedHostsRef = useRef(connectedHosts);
  connectedHostsRef.current = connectedHosts;

  useEffect(() => {
    let current = true;
    setAccounts([]);
    setSelected("");
    setLoadError(null);
    setLoading(true);
    onAccountChangeRef.current(null);
    fetchGitHubCLIAccounts(workspaceId, { cache: "no-store" })
      .then((items) => {
        if (!current) return;
        setAccounts(items);
        const preferred = preferredAccount(items, connectedHostsRef.current, editedHost);
        setSelected(preferred ? `${preferred.host}\n${preferred.login}` : "");
      })
      .catch((error) => {
        if (!current) return;
        setAccounts([]);
        setLoadError(
          error instanceof Error ? error.message : t("github:unexpectedResponseFromTheServer"),
        );
      })
      .finally(() => current && setLoading(false));
    return () => {
      current = false;
    };
  }, [editedHost, workspaceId]);

  const account = useMemo(() => {
    const [host, login] = selected.split("\n");
    return accounts.find((item) => item.host === host && item.login === login);
  }, [accounts, selected]);

  useEffect(() => {
    onAccountChangeRef.current(account ?? null);
  }, [account]);

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="github-cli-account">{t("github:githubCliAccount")}</Label>
        <p className="text-xs text-muted-foreground">
          {t("github:chooseTheExactAccountKandevDoes")}
        </p>
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-stretch">
        <Select
          value={selected}
          onValueChange={setSelected}
          disabled={disabled || loading || !accounts.length}
        >
          <SelectTrigger id="github-cli-account" className="min-w-0 sm:flex-1">
            <SelectValue placeholder={accountPlaceholder(t, loading, loadError)} />
          </SelectTrigger>
          <SelectContent>
            {accounts.map((item) => (
              <SelectItem
                key={`${item.host}:${item.login}`}
                value={`${item.host}\n${item.login}`}
                disabled={cliAccountTaken(item, connectedHosts, editedHost)}
              >
                {item.login} ({item.host}){item.active ? t("github:cliAccountActiveSuffix") : ""}
                {cliAccountTaken(item, connectedHosts, editedHost)
                  ? t("github:cliAccountConnectedSuffix")
                  : ""}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <GitHubCLIAccountNotice
        loadError={loadError}
        loading={loading}
        hasAccounts={accounts.length > 0}
      />
    </div>
  );
}
