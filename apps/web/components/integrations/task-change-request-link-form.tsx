"use client";

import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { Button } from "@kandev/ui/button";
import { DialogFooter } from "@kandev/ui/dialog";
import { Input } from "@kandev/ui/input";
import { Label } from "@kandev/ui/label";
import { useToast } from "@/components/toast-provider";
import { t } from "@/lib/i18n";

export type TaskChangeRequestLinkFormProps = {
  inputLabel: string;
  placeholder?: string;
  emptyError: string;
  failureMessage: string;
  successMessage: string;
  inputTestId?: string;
  errorTestId?: string;
  submitTestId?: string;
  resetKey?: unknown;
  /** Explains why the current input cannot be linked; while it returns content, submit is blocked. */
  inputNotice?: (input: string) => ReactNode;
  onSubmit(reference: string, signal: AbortSignal): Promise<void>;
  onCancel(): void;
  onSuccess(): void;
};

function LinkFormFooter({
  submitting,
  blocked,
  submitTestId,
  onCancel,
}: {
  submitting: boolean;
  blocked: boolean;
  submitTestId?: string;
  onCancel(): void;
}) {
  return (
    <DialogFooter className="gap-2">
      <Button type="button" variant="outline" className="cursor-pointer" onClick={onCancel}>
        {t("common:cancel")}
      </Button>
      <Button
        type="submit"
        className="cursor-pointer"
        disabled={submitting || blocked}
        data-testid={submitTestId}
        data-dialog-default-action
      >
        {submitting ? t("integrations:saving") : t("common:save")}
      </Button>
    </DialogFooter>
  );
}

function LinkFormError({ error, testId }: { error: string | null; testId?: string }) {
  if (!error) return null;
  return (
    <p className="text-xs text-destructive" data-testid={testId}>
      {error}
    </p>
  );
}

/**
 * Host-owned body for task change-request linking. Provider integrations own
 * validation and transport; Kandev owns the interaction, feedback, and layout.
 */
export function TaskChangeRequestLinkForm({
  inputLabel,
  placeholder,
  emptyError,
  failureMessage,
  successMessage,
  inputTestId,
  errorTestId,
  submitTestId,
  resetKey,
  inputNotice,
  onSubmit,
  onCancel,
  onSuccess,
}: TaskChangeRequestLinkFormProps) {
  const inputId = useId();
  const { toast } = useToast();
  const [input, setInput] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const submitController = useRef<AbortController | null>(null);

  useEffect(
    () => () => {
      submitController.current?.abort();
      submitController.current = null;
    },
    [],
  );

  useEffect(() => {
    submitController.current?.abort();
    submitController.current = null;
    setInput("");
    setError(null);
    setSubmitting(false);
  }, [resetKey]);

  const notice = inputNotice?.(input.trim()) ?? null;

  const submit = async () => {
    const reference = input.trim();
    if (notice) return;
    if (!reference) {
      setError(emptyError);
      return;
    }
    submitController.current?.abort();
    const controller = new AbortController();
    submitController.current = controller;
    setSubmitting(true);
    setError(null);
    try {
      await onSubmit(reference, controller.signal);
      if (controller.signal.aborted) return;
      toast({ description: successMessage, variant: "success" });
      onSuccess();
    } catch (err) {
      if (controller.signal.aborted) return;
      setError(err instanceof Error ? err.message : failureMessage);
    } finally {
      if (submitController.current === controller) {
        submitController.current = null;
        if (!controller.signal.aborted) setSubmitting(false);
      }
    }
  };

  const cancel = () => {
    submitController.current?.abort();
    submitController.current = null;
    setSubmitting(false);
    onCancel();
  };

  return (
    <form
      className="contents"
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <div className="space-y-2">
        <Label htmlFor={inputId}>{inputLabel}</Label>
        <Input
          id={inputId}
          data-testid={inputTestId}
          value={input}
          onChange={(event) => setInput(event.target.value)}
          placeholder={placeholder}
          disabled={submitting}
          autoFocus
        />
        {notice ?? <LinkFormError error={error} testId={errorTestId} />}
      </div>
      <LinkFormFooter
        submitting={submitting}
        blocked={Boolean(notice)}
        submitTestId={submitTestId}
        onCancel={cancel}
      />
    </form>
  );
}
