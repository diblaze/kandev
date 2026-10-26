"use client";

import { useCallback, useState } from "react";
import { IconCheck, IconDots, IconTrash, IconX } from "@tabler/icons-react";
import { Badge } from "@kandev/ui/badge";
import { Button } from "@kandev/ui/button";
import { controlSizingClassName } from "@kandev/ui/control-sizing";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@kandev/ui/dropdown-menu";
import { useTranslation } from "react-i18next";
import { useToast } from "@/components/toast-provider";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import type { TaskGitCredentialsState } from "@/hooks/domains/github/use-task-git-credentials";
import { disconnectGitHubWorkspace } from "@/lib/api/domains/github-api";
import { DEFAULT_GITHUB_HOST } from "@/lib/github/host";
import type { GitHubConnectionSource, GitHubHostStatus, GitHubStatus } from "@/lib/types/github";
import { GitHubConnectionDialog } from "./github-connection-dialog";

// Keyed by the wire enum, which is never translated; only the label is copy.
// Catalog keys rather than `t()` calls because this is module scope.
export const githubSourceLabelKeys: Record<GitHubConnectionSource, string> = {
  pat: "github:personalAccessToken",
  gh_cli: "github:githubCli",
  github_app_installation: "github:githubApp",
  legacy_shared: "github:legacySharedConnection",
};

type HostCardProps = {
  workspaceId: string;
  status: GitHubStatus;
  entry: GitHubHostStatus;
  taskAccess: TaskGitCredentialsState;
  onChanged: () => void;
};

export function useHostDisconnect(workspaceId: string, host: string, onChanged: () => void) {
  const { t } = useTranslation();
  const { toast } = useToast();
  const [busy, setBusy] = useState(false);
  const disconnect = useCallback(async () => {
    setBusy(true);
    try {
      await disconnectGitHubWorkspace(workspaceId, host);
      const description =
        host === DEFAULT_GITHUB_HOST
          ? t("github:workspaceGithubConnectionRemoved")
          : t("github:githubHostConnectionRemoved", { host });
      toast({ description, variant: "success" });
      onChanged();
    } catch (error) {
      toast({
        description: error instanceof Error ? error.message : t("github:disconnectFailed"),
        variant: "error",
      });
    } finally {
      setBusy(false);
    }
  }, [host, onChanged, t, toast, workspaceId]);
  return { busy, disconnect };
}

function HostSummary({ entry }: { entry: GitHubHostStatus }) {
  const { t } = useTranslation();
  const active = entry.status === "active";
  const account = entry.login || entry.installation_account_login;
  return (
    <div className="min-w-0 space-y-1">
      <div className="flex min-w-0 flex-wrap items-center gap-2 text-sm">
        {active ? (
          <IconCheck className="h-4 w-4 shrink-0 text-success" />
        ) : (
          <IconX className="h-4 w-4 shrink-0 text-destructive" />
        )}
        {account && <span className="min-w-0 break-words font-medium">{account}</span>}
        <Badge variant={active ? "secondary" : "destructive"}>
          {t(githubSourceLabelKeys[entry.source])}
        </Badge>
        {!active && <Badge variant="outline">{entry.status}</Badge>}
      </div>
      {entry.personal_login && (
        <p className="text-xs text-muted-foreground">
          {t("github:personalLoginOnHost", { login: entry.personal_login })}
        </p>
      )}
      {entry.last_error && <p className="text-xs text-destructive">{entry.last_error}</p>}
    </div>
  );
}

function PhoneHostActions({
  host,
  busy,
  onChange,
  onDisconnect,
}: {
  host: string;
  busy: boolean;
  onChange: () => void;
  onDisconnect: () => void;
}) {
  const { t } = useTranslation();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-11 w-11 shrink-0 cursor-pointer"
          aria-label={t("github:actionsForGithubHost", { host })}
        >
          <IconDots className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem className="cursor-pointer" onSelect={onChange}>
          {t("github:changeConnection")}
        </DropdownMenuItem>
        <DropdownMenuItem
          className="cursor-pointer text-destructive"
          disabled={busy}
          onSelect={onDisconnect}
        >
          {t("github:disconnect")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** One connection of a GitHub host other than github.com. */
export function GitHubHostConnectionCard({
  workspaceId,
  status,
  entry,
  taskAccess,
  onChanged,
}: HostCardProps) {
  const { t } = useTranslation();
  const { isMobile } = useResponsiveBreakpoint();
  const [dialogOpen, setDialogOpen] = useState(false);
  const host = entry.github_host;
  const { busy, disconnect } = useHostDisconnect(workspaceId, host, onChanged);
  const dialog = (
    <GitHubConnectionDialog
      status={status}
      workspaceId={workspaceId}
      host={host}
      onSaved={onChanged}
      taskAccess={taskAccess}
      {...(isMobile ? { open: dialogOpen, onOpenChange: setDialogOpen } : {})}
    />
  );
  return (
    <div
      data-testid="github-host-card"
      data-host={host}
      className="flex flex-col gap-3 border-t pt-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div className="min-w-0 space-y-1">
        <div className="flex items-center justify-between gap-2">
          <span className="break-all text-sm font-semibold">{host}</span>
          {isMobile && (
            <PhoneHostActions
              host={host}
              busy={busy}
              onChange={() => setDialogOpen(true)}
              onDisconnect={() => void disconnect()}
            />
          )}
        </div>
        <HostSummary entry={entry} />
      </div>
      {isMobile ? (
        dialog
      ) : (
        <div className="flex flex-wrap gap-2">
          {dialog}
          <Button
            variant="outline"
            onClick={() => void disconnect()}
            disabled={busy}
            className={controlSizingClassName("standard", "cursor-pointer text-destructive")}
          >
            <IconTrash className="mr-2 h-4 w-4" />
            {t("github:disconnect")}
          </Button>
        </div>
      )}
    </div>
  );
}
