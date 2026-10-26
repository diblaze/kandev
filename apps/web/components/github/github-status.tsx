"use client";

import { useState } from "react";
import {
  IconBrandGithub,
  IconCheck,
  IconExternalLink,
  IconTrash,
  IconX,
} from "@tabler/icons-react";
import { Badge } from "@kandev/ui/badge";
import { Button } from "@kandev/ui/button";
import { SettingsSection } from "@/components/settings/settings-section";
import { useToast } from "@/components/toast-provider";
import { useGitHubStatus } from "@/hooks/domains/github/use-github-status";
import { useGitHubAppRegistrations } from "@/hooks/domains/github/use-github-app-registrations";
import { useTaskGitCredentials } from "@/hooks/domains/github/use-task-git-credentials";
import { disconnectGitHubPersonal, startGitHubPersonalConnect } from "@/lib/api/domains/github-api";
import { getGitHubPersonalIdentityState } from "@/lib/github-auth";
import { DEFAULT_GITHUB_HOST } from "@/lib/github/host";
import type { GitHubConnectionState, GitHubStatus } from "@/lib/types/github";
import { GitHubConnectionDialog } from "./github-connection-dialog";
import {
  GitHubEnterpriseHostCard,
  GitHubHostCard,
  GitHubStatusLoading,
} from "./github-host-connection-card";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

async function redirectFrom(t: TFunction, start: () => Promise<{ url?: string; URL?: string }>) {
  const response = await start();
  const url = response.url ?? response.URL;
  // Surfaced to the user through the caller's toast, so it is copy.
  if (!url) throw new Error(t("github:githubDidNotReturnARedirectUrl"));
  window.location.assign(url);
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback;
}

export function GitHubAutomationSettings({ workspaceId }: { workspaceId: string }) {
  const { status, loaded, loading, refresh } = useGitHubStatus(workspaceId);
  const { registrations } = useGitHubAppRegistrations(workspaceId);
  const taskAccess = useTaskGitCredentials(workspaceId);
  if (!loaded || !status) return <GitHubStatusLoading />;
  const enterpriseHosts = (status.hosts ?? []).filter(
    (entry) => entry.github_host !== DEFAULT_GITHUB_HOST,
  );
  const shared = { workspaceId, registrations, taskAccess, onChanged: refresh };
  return (
    <div className="space-y-3" data-testid="github-host-connections">
      <GitHubHostCard
        {...shared}
        host={DEFAULT_GITHUB_HOST}
        status={status}
        refreshing={loading}
        onRefresh={refresh}
      />
      {enterpriseHosts.map((entry) => (
        <GitHubEnterpriseHostCard key={entry.github_host} {...shared} host={entry.github_host} />
      ))}
      <div className="border-t pt-3">
        <GitHubConnectionDialog
          status={status}
          workspaceId={workspaceId}
          addHost
          onSaved={refresh}
          taskAccess={taskAccess}
        />
      </div>
    </div>
  );
}

type PersonalIdentityView = {
  active: boolean;
  actor: string;
  personalActive: boolean;
  status: GitHubConnectionState | null;
};

function personalIdentityView(status: GitHubStatus): PersonalIdentityView {
  const identity = getGitHubPersonalIdentityState(status);
  return {
    active: identity.active,
    actor: identity.actor,
    personalActive: identity.personalOAuthActive,
    status: status.personal?.status ?? null,
  };
}

function PersonalIdentityStatus({ view }: { view: PersonalIdentityView }) {
  const { t } = useTranslation();
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2 text-sm">
      {view.active ? (
        <IconCheck className="h-4 w-4 text-green-500" />
      ) : (
        <IconX className="h-4 w-4 text-destructive" />
      )}
      <span className="break-words font-medium">{view.actor}</span>
      {view.personalActive && <Badge variant="secondary">{t("github:personalOauth")}</Badge>}
      {view.status && view.status !== "active" && (
        <Badge variant="destructive">{view.status}</Badge>
      )}
    </div>
  );
}

function PersonalIdentityActions({
  status,
  busy,
  onConnect,
  onDisconnect,
}: {
  status: GitHubStatus;
  busy: boolean;
  onConnect: () => void;
  onDisconnect: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-2 sm:flex-row">
      {status.app_available && status.automation?.source === "github_app_installation" && (
        <Button disabled={busy} onClick={onConnect} className="cursor-pointer">
          <IconBrandGithub className="mr-2 h-4 w-4" />
          {status.personal ? t("github:reconnectIdentity") : t("github:connectIdentity")}
          <IconExternalLink className="ml-2 h-4 w-4" />
        </Button>
      )}
      {status.personal && (
        <Button
          variant="outline"
          onClick={onDisconnect}
          disabled={busy}
          className="cursor-pointer text-destructive"
        >
          <IconTrash className="mr-2 h-4 w-4" />
          {t("github:disconnect")}
        </Button>
      )}
    </div>
  );
}

export function GitHubPersonalSettings({ workspaceId }: { workspaceId: string }) {
  const { t } = useTranslation();
  const { status, loaded, refresh } = useGitHubStatus(workspaceId);
  const [busy, setBusy] = useState(false);
  const { toast } = useToast();
  if (!loaded || !status) return null;
  const view = personalIdentityView(status);
  const appAutomation = status.automation?.source === "github_app_installation";
  if (!status.automation) return null;
  if (!appAutomation) return null;
  const disconnect = async () => {
    setBusy(true);
    try {
      await disconnectGitHubPersonal(workspaceId);
      toast({ description: t("github:personalGithubIdentityDisconnected"), variant: "success" });
      refresh();
    } catch (error) {
      toast({
        description: error instanceof Error ? error.message : t("github:disconnectFailed"),
        variant: "error",
      });
    } finally {
      setBusy(false);
    }
  };
  const connect = () => {
    void redirectFrom(t, () => startGitHubPersonalConnect(workspaceId)).catch((error: unknown) =>
      toast({
        description: errorMessage(error, t("github:identityConnectionFailed")),
        variant: "error",
      }),
    );
  };
  return (
    <SettingsSection
      title={t("github:myGithubIdentity")}
      description={t("github:optionallyConnectYourGithubUserFor")}
    >
      <div className="space-y-4" data-testid="github-personal-identity">
        <PersonalIdentityStatus view={view} />
        {status.personal?.last_error && (
          <p className="text-xs text-destructive">{status.personal.last_error}</p>
        )}
        <PersonalIdentityActions
          status={status}
          busy={busy}
          onConnect={connect}
          onDisconnect={disconnect}
        />
      </div>
    </SettingsSection>
  );
}

/** Compatibility export used by older settings entrypoints. */
export function GitHubStatusCard({ workspaceId }: { workspaceId: string }) {
  return <GitHubAutomationSettings workspaceId={workspaceId} />;
}
