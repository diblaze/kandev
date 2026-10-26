"use client";

import { useEffect, useState } from "react";
import { IconEye, IconEyeOff } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { Input } from "@kandev/ui/input";
import { Label } from "@kandev/ui/label";
import { useTranslation } from "react-i18next";
import { settingsCredentialClassName } from "@/components/settings/settings-control";
import { DEFAULT_GITHUB_HOST } from "@/lib/github/host";

export function GitHubPATForm({
  workspaceId,
  value,
  onChange,
  host,
  onHostChange,
  hostInvalid,
  disabled,
}: {
  workspaceId: string;
  value: string;
  onChange: (value: string) => void;
  host: string;
  onHostChange: (value: string) => void;
  hostInvalid: boolean;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    setVisible(false);
  }, [workspaceId]);

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="github-workspace-host">{t("github:githubHost")}</Label>
        <p className="text-xs text-muted-foreground">{t("github:githubHostHelp")}</p>
        <Input
          id="github-workspace-host"
          data-testid="github-workspace-host"
          value={host}
          onChange={(event) => onHostChange(event.target.value)}
          placeholder={DEFAULT_GITHUB_HOST}
          autoComplete="off"
          spellCheck={false}
          disabled={disabled}
          aria-invalid={hostInvalid || undefined}
        />
        {hostInvalid && (
          <p role="alert" className="text-xs text-destructive">
            {t("github:enterAValidGithubHost")}
          </p>
        )}
      </div>
      <div className="space-y-1">
        <Label htmlFor="github-workspace-token">{t("github:personalAccessToken")}</Label>
        <p className="text-xs text-muted-foreground">{t("github:kandevStoresThisTokenForThis")}</p>
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-stretch">
        <div className="relative min-w-0 flex-1">
          <Input
            id="github-workspace-token"
            type={visible ? "text" : "password"}
            value={value}
            onChange={(event) => onChange(event.target.value)}
            placeholder="ghp_xxxxxxxxxxxx"
            autoComplete="off"
            disabled={disabled}
            className={settingsCredentialClassName("pr-11")}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="absolute right-0 top-0 cursor-pointer"
            onClick={() => setVisible((current) => !current)}
            disabled={disabled}
            aria-label={visible ? t("github:hideToken") : t("github:showToken")}
          >
            {visible ? <IconEyeOff className="h-4 w-4" /> : <IconEye className="h-4 w-4" />}
          </Button>
        </div>
      </div>
    </div>
  );
}
