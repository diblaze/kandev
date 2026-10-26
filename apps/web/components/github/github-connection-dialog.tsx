"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { IconPlug } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@kandev/ui/dialog";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@kandev/ui/drawer";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import type { TaskGitCredentialsState } from "@/hooks/domains/github/use-task-git-credentials";
import type { GitHubStatus } from "@/lib/types/github";
import { DEFAULT_GITHUB_HOST } from "@/lib/github/host";
import type { GitHubAutomationMethod } from "./github-auth-method-list";
import {
  GitHubConnectionSettingsForm,
  methodForConnection,
  type GitHubHostConnection,
} from "./github-connection-settings-form";
import { useTranslation } from "react-i18next";
import { controlSizingClassName } from "@kandev/ui/control-sizing";

function hostConnection(
  status: GitHubStatus,
  host: string | undefined,
  addHost: boolean,
): GitHubHostConnection {
  if (addHost) return null;
  if (host && host !== DEFAULT_GITHUB_HOST) {
    return status.hosts?.find((entry) => entry.github_host === host) ?? null;
  }
  return status.automation ?? null;
}

type GitHubConnectionDialogProps = {
  status: GitHubStatus;
  workspaceId: string;
  onSaved: () => void;
  taskAccess: TaskGitCredentialsState;
  /** Host whose connection the dialog changes; absent means github.com. */
  host?: string;
  /** Connect a host that has no connection yet. */
  addHost?: boolean;
  /** Controlled open state; when set, the dialog renders no trigger of its own. */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
};

function useDialogOpenState(
  props: Pick<GitHubConnectionDialogProps, "open" | "onOpenChange" | "workspaceId">,
) {
  const [localOpen, setLocalOpen] = useState(false);
  const controlled = props.open !== undefined;
  const open = controlled ? Boolean(props.open) : localOpen;
  const { onOpenChange } = props;
  const setOpen = useCallback(
    (next: boolean) => (controlled ? onOpenChange?.(next) : setLocalOpen(next)),
    [controlled, onOpenChange],
  );
  useEffect(() => {
    setLocalOpen(false);
  }, [props.workspaceId]);
  return { open, setOpen, controlled };
}

function triggerLabelKey(addHost: boolean, connected: boolean) {
  if (addHost) return "github:connectAnotherGithubHost";
  return connected ? "github:changeConnection" : "github:connectGithub";
}

function titleKey(addHost: boolean, connected: boolean) {
  if (addHost) return "github:connectAnotherGithubHost";
  return connected ? "github:changeGithubConnection" : "github:connectGithub";
}

export function GitHubConnectionDialog(props: GitHubConnectionDialogProps) {
  const { status, workspaceId, onSaved, taskAccess, host, addHost = false } = props;
  const { t } = useTranslation();
  const { open, setOpen, controlled } = useDialogOpenState(props);
  const connection = useMemo(() => hostConnection(status, host, addHost), [addHost, host, status]);
  const connectedHosts = useMemo(
    () => (status.hosts ?? []).map((entry) => entry.github_host),
    [status.hosts],
  );
  const [method, setMethod] = useState<GitHubAutomationMethod>(() =>
    methodForConnection(connection),
  );
  const { isMobile } = useResponsiveBreakpoint();
  const connected = Boolean(connection);

  useEffect(() => {
    setMethod(methodForConnection(connection));
  }, [connection]);

  const openChange = useCallback(
    (next: boolean) => {
      if (next) setMethod(methodForConnection(connection));
      setOpen(next);
    },
    [connection, setOpen],
  );
  const trigger = controlled ? null : (
    <Button
      variant={connected || addHost ? "outline" : "default"}
      className={controlSizingClassName("standard", "cursor-pointer")}
      data-testid={addHost ? "github-connect-another-host" : undefined}
    >
      <IconPlug className="mr-2 h-4 w-4" />
      {t(triggerLabelKey(addHost, connected))}
    </Button>
  );
  const title = t(titleKey(addHost, connected));
  const body = (
    <GitHubConnectionSettingsForm
      connection={connection}
      defaultHost={host ?? DEFAULT_GITHUB_HOST}
      connectedHosts={connectedHosts}
      method={method}
      workspaceId={workspaceId}
      open={open}
      onMethodChange={setMethod}
      onSaved={onSaved}
      onComplete={() => setOpen(false)}
      taskAccess={taskAccess}
      isMobile={isMobile}
    />
  );

  if (isMobile) {
    return (
      <Drawer open={open} onOpenChange={openChange}>
        {trigger && <DrawerTrigger asChild>{trigger}</DrawerTrigger>}
        <DrawerContent
          data-testid="github-connection-mobile"
          className="h-[calc(100dvh-16px-env(safe-area-inset-bottom,0px))] max-h-[calc(100dvh-16px-env(safe-area-inset-bottom,0px))] overflow-hidden"
        >
          <DrawerHeader className="shrink-0 border-b text-left">
            <DrawerTitle>{title}</DrawerTitle>
            <DrawerDescription>{t("github:thisWorkspaceUsesOneCredentialFor")}</DrawerDescription>
          </DrawerHeader>
          {body}
        </DrawerContent>
      </Drawer>
    );
  }

  return (
    <Dialog open={open} onOpenChange={openChange}>
      {trigger && <DialogTrigger asChild>{trigger}</DialogTrigger>}
      <DialogContent
        data-testid="github-connection-desktop"
        className="flex max-h-[85dvh] flex-col overflow-hidden sm:max-w-4xl"
      >
        <DialogHeader className="shrink-0">
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{t("github:thisWorkspaceUsesOneCredentialFor")}</DialogDescription>
        </DialogHeader>
        {body}
      </DialogContent>
    </Dialog>
  );
}
