"use client";

import Link from "@/components/routing/app-link";
import { useTranslation } from "react-i18next";

function githubSettingsHref(workspaceId?: string | null) {
  return workspaceId
    ? `/settings/workspaces/${workspaceId}/integrations/github`
    : "/settings/integrations/github";
}

/** Tells the user that a pasted GitHub URL is on a host the workspace has not connected. */
export function GitHubHostNotConnectedNotice({
  host,
  workspaceId,
}: {
  host: string;
  workspaceId?: string | null;
}) {
  const { t } = useTranslation();
  return (
    <p role="status" className="text-xs text-destructive" data-testid="github-host-not-connected">
      {t("github:githubHostNotConnected", { host })}{" "}
      <Link
        href={githubSettingsHref(workspaceId)}
        className="cursor-pointer font-medium underline"
        data-testid="github-host-not-connected-settings"
      >
        {t("github:openGithubSettings")}
      </Link>
    </p>
  );
}
