"use client";

import { useCallback, useState, type ReactNode } from "react";
import { IconCheck, IconRefresh, IconTrash, IconX } from "@tabler/icons-react";
import { Badge } from "@kandev/ui/badge";
import { Button } from "@kandev/ui/button";
import { controlSizingClassName } from "@kandev/ui/control-sizing";
import { Spinner } from "@kandev/ui/spinner";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import { useToast } from "@/components/toast-provider";
import { useGitHubHostStatus } from "@/hooks/domains/github/use-github-status";
import type { TaskGitCredentialsState } from "@/hooks/domains/github/use-task-git-credentials";
import { disconnectGitHubWorkspace } from "@/lib/api/domains/github-api";
import { DEFAULT_GITHUB_HOST } from "@/lib/github/host";
import type {
  GitHubAppRegistrationCatalogItem,
  GitHubConnectionSource,
  GitHubStatus,
} from "@/lib/types/github";
import { GitHubAccessHelp } from "./github-access-help";
import { GitHubConnectionDialog } from "./github-connection-dialog";
import { GitHubPermissionsDialog } from "./github-permissions-dialog";
import { GitHubPRDiscoveryHealthWarning } from "./github-pr-discovery-health";
import { GitHubRateLimitDisplay } from "./github-rate-limit";
import { GitHubTaskAccessSummary } from "./github-task-credentials-section";

// Keyed by the wire enum, which is never translated; only the label is copy.
// Catalog keys rather than `t()` calls because this is module scope.
const githubSourceLabelKeys: Record<GitHubConnectionSource, string> = {
  pat: "github:personalAccessToken",
  gh_cli: "github:githubCli",
  github_app_installation: "github:githubApp",
  legacy_shared: "github:legacySharedConnection",
};

function useHostDisconnect(workspaceId: string, host: string, onChanged: () => void) {
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

// `t` is threaded in rather than imported at module level so the label follows a
// locale switch and stays a pure function of its arguments.
function connectionLabel(t: TFunction, status: GitHubStatus): string {
  const connection = status.automation;
  if (!connection) return "";
  // The first three are GitHub account logins: data, never translated.
  return (
    connection.actor?.login ??
    connection.login ??
    connection.installation_account_login ??
    t("github:githubApp")
  );
}

function automationActor(status: GitHubStatus): string | null {
  if (!status.authenticated) return null;
  return status.automation?.actor?.login ?? null;
}

function StatusLine({
  status,
  onRateLimitOpen,
}: {
  status: GitHubStatus;
  onRateLimitOpen: () => void;
}) {
  const { t } = useTranslation();
  const connection = status.automation;
  if (!connection) {
    return (
      <div className="flex items-center gap-2 text-sm">
        <IconX className="h-4 w-4 shrink-0 text-destructive" />
        <span>{t("github:noAutomationConnection")}</span>
      </div>
    );
  }
  const actor = automationActor(status);
  const active = connection.status === "active" && actor !== null;
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2 text-sm">
      {active ? (
        <IconCheck className="h-4 w-4 shrink-0 text-green-500" />
      ) : (
        <IconX className="h-4 w-4 shrink-0 text-destructive" />
      )}
      <span className="min-w-0 break-words font-medium">
        {actor ??
          (connection.status === "active"
            ? t("github:authenticationUnavailable")
            : connectionLabel(t, status))}
      </span>
      <Badge variant={active ? "secondary" : "destructive"}>
        {t(githubSourceLabelKeys[connection.source])}
      </Badge>
      {connection.status !== "active" && <Badge variant="outline">{connection.status}</Badge>}
      <GitHubRateLimitDisplay
        info={status.rate_limit}
        discoveryHealth={status.pr_discovery_health}
        onOpen={onRateLimitOpen}
      />
    </div>
  );
}

function AutomationActorExplanation({
  status,
  appAutomation,
}: {
  status: GitHubStatus;
  appAutomation: boolean;
}) {
  const { t } = useTranslation();
  const actor = status.automation?.actor?.login;
  if (!status.authenticated || !actor) return null;
  const humanIdentity = status.effective_personal_actor?.kind === "human";
  return (
    <div className="flex items-start gap-1 text-xs text-muted-foreground">
      <GitHubAccessHelp
        label={t("github:explainWorkspaceGithubIdentity")}
        title={t("github:workspaceGithubIdentity")}
        description={t("github:kandevUsesThisWorkspaceConnectionFor")}
      />
      <div className="min-w-0 space-y-1 pt-3 sm:pt-1">
        <p>
          {appAutomation
            ? t("github:kandevManagedOperationsUseTheGithub", { actor })
            : t("github:kandevManagedOperationsActAs", { actor })}
        </p>
        {!appAutomation && humanIdentity && <p>{t("github:thisAccountAlsoPowersMyGithub")}</p>}
      </div>
    </div>
  );
}

function AppRegistrationDetails({ app }: { app?: GitHubAppRegistrationCatalogItem }) {
  const { t } = useTranslation();
  if (!app) return null;
  return (
    <>
      <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-muted-foreground">
        <span className="break-words">{app.display_name}</span>
        <Badge variant="outline" className="capitalize">
          {app.visibility}
        </Badge>
        <Badge variant="outline" className="capitalize">
          {t("github:webhook")} {app.webhook_status}
        </Badge>
        <span>{app.source === "managed" ? t("github:createdByKandev") : t("github:imported")}</span>
      </div>
      {app.shared && (
        <p className="text-xs text-amber-600 dark:text-amber-400">
          {t("github:thisAppRegistrationIsShared", { count: app.workspace_binding_count })}
        </p>
      )}
    </>
  );
}

type SummaryProps = {
  host: string;
  status: GitHubStatus;
  registrations: GitHubAppRegistrationCatalogItem[];
  taskAccess: Omit<TaskGitCredentialsState, "save">;
  onRateLimitOpen: () => void;
};

function AutomationStatusSummary({
  host,
  status,
  registrations,
  taskAccess,
  onRateLimitOpen,
}: SummaryProps) {
  const { t } = useTranslation();
  const appAutomation = status.automation?.source === "github_app_installation";
  const registrationId = status.app_registration?.id ?? status.automation?.app_registration_id;
  const app = registrations.find((item) => item.id === registrationId);
  // github.com shows its personal identity in its own section; other hosts have none.
  const personalLogin = host === DEFAULT_GITHUB_HOST ? undefined : status.personal?.login;
  return (
    <div className="min-w-0 space-y-1">
      <StatusLine status={status} onRateLimitOpen={onRateLimitOpen} />
      <GitHubPRDiscoveryHealthWarning health={status.pr_discovery_health} />
      <AutomationActorExplanation status={status} appAutomation={appAutomation} />
      {appAutomation && <AppRegistrationDetails app={app} />}
      {personalLogin && (
        <p className="text-xs text-muted-foreground">
          {t("github:personalLoginOnHost", { login: personalLogin })}
        </p>
      )}
      {status.automation?.last_error && (
        <p className="text-xs text-destructive">{status.automation.last_error}</p>
      )}
      {/* Task Git access is one workspace-wide setting, so only the first card shows it. */}
      {host === DEFAULT_GITHUB_HOST && <GitHubTaskAccessSummary {...taskAccess} />}
    </div>
  );
}

function GitHubRefreshButton({
  refreshing,
  onRefresh,
}: {
  refreshing: boolean;
  onRefresh: () => void;
}) {
  const { t } = useTranslation();
  return (
    <Button
      variant="outline"
      size="icon"
      onClick={onRefresh}
      disabled={refreshing}
      aria-busy={refreshing}
      className={controlSizingClassName("icon", "cursor-pointer")}
      aria-label={t("github:refreshGithubConnection")}
    >
      <IconRefresh className={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
    </Button>
  );
}

/** Outer frame shared by every host card, also while a host status is unavailable. */
function GitHubHostCardFrame({ host, children }: { host: string; children: ReactNode }) {
  return (
    <div
      data-testid="github-host-card"
      data-host={host}
      className="space-y-1 border-t pt-3 first:border-t-0 first:pt-0"
    >
      <span className="break-all text-sm font-semibold">{host}</span>
      <div
        className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
        data-testid="github-workspace-automation"
      >
        {children}
      </div>
    </div>
  );
}

type GitHubHostCardProps = {
  workspaceId: string;
  host: string;
  /** Status of `host`, as returned by the host-scoped status request. */
  status: GitHubStatus;
  registrations: GitHubAppRegistrationCatalogItem[];
  taskAccess: TaskGitCredentialsState;
  refreshing: boolean;
  /** Refetches this host's status. */
  onRefresh: () => void;
  /** Runs after the connection was changed or removed. */
  onChanged: () => void;
};

/** Connection, identity, API limits, and actions of one GitHub host. */
export function GitHubHostCard({
  workspaceId,
  host,
  status,
  registrations,
  taskAccess,
  refreshing,
  onRefresh,
  onChanged,
}: GitHubHostCardProps) {
  const { t } = useTranslation();
  const { busy, disconnect } = useHostDisconnect(workspaceId, host, onChanged);
  return (
    <GitHubHostCardFrame host={host}>
      <AutomationStatusSummary
        host={host}
        status={status}
        registrations={registrations}
        taskAccess={taskAccess}
        onRateLimitOpen={onRefresh}
      />
      <div className="flex flex-wrap gap-2">
        <GitHubPermissionsDialog status={status} />
        <GitHubConnectionDialog
          status={status}
          workspaceId={workspaceId}
          host={host}
          onSaved={onChanged}
          taskAccess={taskAccess}
        />
        <GitHubRefreshButton refreshing={refreshing} onRefresh={onRefresh} />
        {status.automation && (
          <Button
            variant="outline"
            onClick={() => void disconnect()}
            disabled={busy}
            className={controlSizingClassName("standard", "cursor-pointer text-destructive")}
          >
            <IconTrash className="mr-2 h-4 w-4" />
            {t("github:disconnect")}
          </Button>
        )}
      </div>
    </GitHubHostCardFrame>
  );
}

type GitHubEnterpriseHostCardProps = Omit<
  GitHubHostCardProps,
  "status" | "refreshing" | "onRefresh"
>;

/** A host other than github.com, with the status fetched for that host. */
export function GitHubEnterpriseHostCard(props: GitHubEnterpriseHostCardProps) {
  const { t } = useTranslation();
  const { workspaceId, host, onChanged } = props;
  const { status, loading, refresh } = useGitHubHostStatus(workspaceId, host);
  const changed = useCallback(() => {
    onChanged();
    refresh();
  }, [onChanged, refresh]);
  if (!status) {
    return (
      <GitHubHostCardFrame host={host}>
        {loading ? (
          <GitHubStatusLoading />
        ) : (
          <p className="text-xs text-destructive">
            {t("github:githubHostStatusUnavailable", { host })}
          </p>
        )}
        <GitHubRefreshButton refreshing={loading} onRefresh={refresh} />
      </GitHubHostCardFrame>
    );
  }
  return (
    <GitHubHostCard
      {...props}
      status={status}
      refreshing={loading}
      onRefresh={refresh}
      onChanged={changed}
    />
  );
}

export function GitHubStatusLoading() {
  const { t } = useTranslation();
  return (
    <div className="flex min-h-11 items-center gap-2 text-sm text-muted-foreground">
      <Spinner className="h-4 w-4" />
      {t("github:checkingGithubConnection")}
    </div>
  );
}
