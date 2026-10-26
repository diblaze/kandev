"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@kandev/ui/button";
import { Spinner } from "@kandev/ui/spinner";
import { useToast } from "@/components/toast-provider";
import type { TaskGitCredentialsState } from "@/hooks/domains/github/use-task-git-credentials";
import {
  setGitHubWorkspaceConnection,
  type SetGitHubConnectionRequest,
} from "@/lib/api/domains/github-api";
import type {
  GitHubCLIAccount,
  GitHubConnectionSource,
  TaskGitCredentialsMode,
} from "@/lib/types/github";
import { normalizeGitHubHost } from "@/lib/github/host";
import { cn } from "@/lib/utils";
import { GitHubAppConnectionPanel } from "./github-app-connection-panel";
import { GitHubAuthMethodList, type GitHubAutomationMethod } from "./github-auth-method-list";
import { GitHubCLIForm } from "./github-cli-form";
import { GitHubPATForm } from "./github-pat-form";
import { GitHubTaskAccessForm } from "./github-task-credentials-section";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

/** The existing connection of the host the form edits; null when the host has none. */
export type GitHubHostConnection = {
  source: GitHubConnectionSource;
  github_host: string;
  login?: string;
} | null;

export function methodForConnection(connection: GitHubHostConnection): GitHubAutomationMethod {
  if (connection?.source === "github_app_installation") return "app";
  if (connection?.source === "gh_cli") return "cli";
  return "pat";
}

// `t` is threaded in: these are plain functions, and their messages are shown
// to the user through the caller's toast.
function errorMessage(t: TFunction, error: unknown) {
  return error instanceof Error ? error.message : t("github:theSettingsCouldNotBeSaved");
}

// A partial save reports which change failed alongside the ones that landed.
function saveFailureToast(t: TFunction, reason: unknown, successes: number) {
  const detail = errorMessage(t, reason);
  return {
    description: successes > 0 ? t("github:someSettingsWereSavedButAnother", { detail }) : detail,
    variant: "error" as const,
  };
}

function runSaveOperations({
  t,
  workspaceId,
  connectionRequest,
  taskMode,
  saveTask,
}: {
  t: TFunction;
  workspaceId: string;
  connectionRequest: SetGitHubConnectionRequest | null;
  taskMode: TaskGitCredentialsMode | null;
  saveTask: TaskGitCredentialsState["save"];
}) {
  const operations: Promise<unknown>[] = [];
  if (connectionRequest) {
    operations.push(setGitHubWorkspaceConnection(workspaceId, connectionRequest));
  }
  if (taskMode) {
    operations.push(
      saveTask(taskMode).then((saved) => {
        if (!saved) throw new Error(t("github:theWorkspaceChangedBeforeTaskAccess"));
      }),
    );
  }
  return Promise.allSettled(operations);
}

function buildConnectionRequest({
  method,
  currentMethod,
  token,
  patHost,
  cliAccount,
  connection,
}: {
  method: GitHubAutomationMethod;
  currentMethod: GitHubAutomationMethod;
  token: string;
  patHost: string | null;
  cliAccount: GitHubCLIAccount | null;
  connection: GitHubHostConnection;
}): SetGitHubConnectionRequest | null {
  if (method === "pat" && token.trim() && patHost) {
    return { source: "pat", token: token.trim(), host: patHost };
  }
  if (
    method === "cli" &&
    cliAccount &&
    (currentMethod !== "cli" ||
      connection?.github_host !== cliAccount.host ||
      connection.login !== cliAccount.login)
  ) {
    return { source: "gh_cli", host: cliAccount.host, login: cliAccount.login };
  }
  return null;
}

function isConnectionDraftInvalid(draft: {
  method: GitHubAutomationMethod;
  currentMethod: GitHubAutomationMethod;
  patHost: string | null;
  token: string;
  cliAccount: GitHubCLIAccount | null;
}) {
  if (draft.method === "app") return draft.currentMethod !== "app";
  if (draft.method === "cli") return !draft.cliAccount;
  return !draft.patHost || (draft.currentMethod !== "pat" && !draft.token.trim());
}

type SettingsFormProps = {
  connection: GitHubHostConnection;
  /** Host the token form starts with: the edited host, or github.com for a new one. */
  defaultHost: string;
  /** Hosts that already have a connection. */
  connectedHosts: readonly string[];
  method: GitHubAutomationMethod;
  workspaceId: string;
  open: boolean;
  onMethodChange: (method: GitHubAutomationMethod) => void;
  onSaved: () => void;
  onComplete: () => void;
  taskAccess: TaskGitCredentialsState;
  isMobile: boolean;
};

function useConnectionSettingsDraft({
  connection,
  defaultHost,
  method,
  workspaceId,
  open,
  onSaved,
  onComplete,
  taskAccess,
}: SettingsFormProps) {
  const { t } = useTranslation();
  const [token, setToken] = useState("");
  const [host, setHost] = useState(defaultHost);
  const [cliAccount, setCLIAccount] = useState<GitHubCLIAccount | null>(null);
  const [taskMode, setTaskMode] = useState<TaskGitCredentialsMode>(taskAccess.mode);
  const [saving, setSaving] = useState(false);
  const activeWorkspaceId = useRef(workspaceId);
  const { toast } = useToast();
  activeWorkspaceId.current = workspaceId;
  useEffect(() => {
    if (!open) return;
    setToken("");
    setHost(defaultHost);
    setSaving(false);
  }, [defaultHost, open, workspaceId]);
  useEffect(() => {
    if (open) setTaskMode(taskAccess.mode);
  }, [open, taskAccess.mode, workspaceId]);
  const currentMethod = methodForConnection(connection);
  const taskDirty = taskMode !== taskAccess.mode;
  const patHost = normalizeGitHubHost(host);
  const connectionRequest = useMemo(
    () => buildConnectionRequest({ method, currentMethod, token, patHost, cliAccount, connection }),
    [cliAccount, connection, currentMethod, method, patHost, token],
  );
  const methodNeedsWorkflow = method === "app" && currentMethod !== "app";
  const hostInvalid = method === "pat" && !patHost;
  const draft = { method, currentMethod, patHost, token, cliAccount };
  const connectionInvalid = isConnectionDraftInvalid(draft);
  const canSave =
    !saving &&
    !taskAccess.loading &&
    !taskAccess.error &&
    !connectionInvalid &&
    (Boolean(connectionRequest) || taskDirty);
  const save = useCallback(async () => {
    if (!canSave) return;
    const savingWorkspaceId = workspaceId;
    setSaving(true);
    const results = await runSaveOperations({
      t,
      workspaceId,
      connectionRequest,
      taskMode: taskDirty ? taskMode : null,
      saveTask: taskAccess.save,
    });
    if (activeWorkspaceId.current !== savingWorkspaceId) return;
    setSaving(false);
    const failures = results.filter(
      (result): result is PromiseRejectedResult => result.status === "rejected",
    );
    const successes = results.length - failures.length;
    if (successes > 0) onSaved();
    if (failures.length === 0) {
      setToken("");
      toast({ description: t("github:githubAccessSettingsSaved"), variant: "success" });
      onComplete();
      return;
    }
    toast(saveFailureToast(t, failures[0].reason, successes));
  }, [
    canSave,
    connectionRequest,
    onComplete,
    onSaved,
    t,
    taskAccess,
    taskDirty,
    taskMode,
    toast,
    workspaceId,
  ]);
  return {
    token,
    setToken,
    host,
    setHost,
    hostInvalid,
    setCLIAccount,
    taskMode,
    setTaskMode,
    saving,
    methodNeedsWorkflow,
    canSave,
    save,
  };
}

export function GitHubConnectionSettingsForm(props: SettingsFormProps) {
  const { t } = useTranslation();
  const { method, workspaceId, onMethodChange, taskAccess, isMobile } = props;
  const {
    token,
    setToken,
    host,
    setHost,
    hostInvalid,
    setCLIAccount,
    taskMode,
    setTaskMode,
    saving,
    methodNeedsWorkflow,
    canSave,
    save,
  } = useConnectionSettingsDraft(props);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="relative flex min-h-0 flex-1 flex-col overflow-hidden">
        <div
          data-testid="github-connection-scroll"
          className={cn(
            "min-h-0 flex-1 overflow-y-auto overscroll-contain pb-8",
            isMobile ? "px-4 pt-4" : "pr-1",
          )}
        >
          <div className="space-y-5">
            <GitHubAuthMethodList value={method} onChange={onMethodChange} />
            <div className="border-t pt-5">
              {method === "pat" && (
                <GitHubPATForm
                  workspaceId={workspaceId}
                  value={token}
                  onChange={setToken}
                  host={host}
                  onHostChange={setHost}
                  hostInvalid={hostInvalid}
                  disabled={saving}
                />
              )}
              {method === "cli" && (
                <GitHubCLIForm
                  workspaceId={workspaceId}
                  onAccountChange={setCLIAccount}
                  editedHost={props.connection?.github_host}
                  connectedHosts={props.connectedHosts}
                  disabled={saving}
                />
              )}
              {method === "app" && <GitHubAppConnectionPanel workspaceId={workspaceId} />}
            </div>
            <GitHubTaskAccessForm
              taskAccess={taskAccess}
              value={taskMode}
              onChange={setTaskMode}
              disabled={saving}
            />
            {methodNeedsWorkflow && (
              <p className="text-xs leading-5 text-muted-foreground">
                {t("github:installAGithubAppAboveTo")}
              </p>
            )}
          </div>
        </div>
        <div
          data-testid="github-connection-scroll-fade"
          aria-hidden="true"
          className="pointer-events-none absolute inset-x-0 bottom-0 h-6 bg-gradient-to-t from-popover to-transparent"
        />
      </div>
      <div
        data-testid="github-connection-footer"
        className={cn(
          "flex shrink-0 justify-end border-t bg-popover pt-3",
          isMobile ? "px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom,0px))]" : "pb-0",
        )}
      >
        <Button
          type="button"
          disabled={!canSave}
          onClick={save}
          className="w-full cursor-pointer sm:w-auto"
          data-dialog-default-action
        >
          {saving && <Spinner className="mr-2 h-4 w-4" />}
          {saving ? t("github:savingChanges") : t("github:saveChanges")}
        </Button>
      </div>
    </div>
  );
}
