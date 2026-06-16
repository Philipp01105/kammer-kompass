"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Check, EyeOff, Play, RotateCcw, Send, X } from "lucide-react";
import { useState } from "react";
import { adminApi } from "@/lib/api";
import { formatDate } from "@/lib/markdown";
import type { AdminInfoSuggestionDetail as Detail } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { MarkdownEditor } from "@/components/admin/MarkdownEditor";
import { ModerationResultBadge } from "@/components/admin/ModerationResultBadge";
import { StatusBadge } from "@/components/admin/StatusBadge";
import { useAbilities } from "@/lib/abilities";

export function InfoSuggestionDetail({ detail }: { detail: Detail }) {
  const queryClient = useQueryClient();
  const { abilities } = useAbilities();
  const [hideReason, setHideReason] = useState("");
  const [hiddenPendingSuggestionIds, setHiddenPendingSuggestionIds] = useState<Set<string>>(() => new Set());
  const [newText, setNewText] = useState(detail.liveCurrentText || detail.currentTextSnapshot);
  const [confidenceLevel, setConfidenceLevel] = useState("medium");
  const [sourceSummary, setSourceSummary] = useState("");
  const [changeSummary, setChangeSummary] = useState("");

  const pendingVisible = detail.publicPendingVisible && !hiddenPendingSuggestionIds.has(detail.id);
  const canStartReview = detail.status === "submitted" && abilities.canReviewInfoSuggestions;
  const canReview = detail.status === "under_review" && abilities.canReviewInfoSuggestions;
  const canReopen = (detail.status === "needs_more_info" || detail.status === "rejected") && abilities.canReviewInfoSuggestions;
  const canHidePending = abilities.canHidePendingHints && pendingVisible;
  const showWorkflow = canStartReview || canReview || canReopen || canHidePending;

  const action = useMutation({
    mutationFn: ({ endpoint, body }: { endpoint: string; body?: unknown }) =>
      adminApi.postInfoSuggestionAction(detail.id, endpoint, body),
    onSuccess: async (_data, variables) => {
      if (variables.endpoint === "hide-pending") {
        setHiddenPendingSuggestionIds((ids) => new Set(ids).add(detail.id));
        setHideReason("");
      }
      await queryClient.invalidateQueries({ queryKey: ["admin-info-suggestion", detail.id] });
      await queryClient.invalidateQueries({ queryKey: ["admin-info-suggestions"] });
    },
  });
  const apply = useMutation({
    mutationFn: () =>
      adminApi.applyInfoSuggestion(detail.id, {
        newText,
        confidenceLevel,
        sourceSummary: sourceSummary || undefined,
        changeSummary,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-info-suggestion", detail.id] });
      await queryClient.invalidateQueries({ queryKey: ["public-ihks"] });
    },
  });

  return (
    <div className="space-y-6">
      <section className="rounded-lg border p-4">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="text-xl font-semibold">{detail.ihk.name}</h2>
          <StatusBadge status={detail.status} />
          <ModerationResultBadge status={detail.preModerationStatus} />
        </div>
        <p className="mt-2 text-sm text-muted-foreground">
          Sprache: de · Confidence: {String(detail.languageConfidence)} · Pending sichtbar:{" "}
          {pendingVisible ? "ja" : "nein"}
        </p>
      </section>

      <section className="grid gap-4 lg:grid-cols-2">
        <TextPanel title="Live-Text aktuell" text={detail.liveCurrentText} />
        <TextPanel title="Snapshot zum Vorschlagszeitpunkt" text={detail.currentTextSnapshot} />
        <TextPanel title="Vorschlag" text={detail.suggestedChange} />
        <TextPanel title="Public Pending Text" text={detail.publicPendingText} />
      </section>

      {showWorkflow ? (
        <section className="rounded-lg border p-4">
          <h3 className="font-semibold">Workflow</h3>
          <div className="mt-4 flex flex-wrap gap-2">
            {canStartReview ? (
              <Button onClick={() => action.mutate({ endpoint: "start-review" })}>
                <Play className="h-4 w-4" />
                In Prüfung nehmen
              </Button>
            ) : null}
            {canReview ? (
              <>
                <Button onClick={() => action.mutate({ endpoint: "accept" })}>
                  <Check className="h-4 w-4" />
                  Akzeptieren
                </Button>
                <Button variant="danger" onClick={() => action.mutate({ endpoint: "reject" })}>
                  <X className="h-4 w-4" />
                  Ablehnen
                </Button>
                <Button variant="secondary" onClick={() => action.mutate({ endpoint: "needs-more-info" })}>
                  Mehr Infos nötig
                </Button>
                <Button variant="secondary" onClick={() => action.mutate({ endpoint: "mark-spam" })}>
                  Spam
                </Button>
              </>
            ) : null}
            {canReopen ? (
              <Button variant="secondary" onClick={() => action.mutate({ endpoint: "reopen" })}>
                <RotateCcw className="h-4 w-4" />
                Erneut öffnen
              </Button>
            ) : null}
          </div>
          {canHidePending ? (
            <form
              className="mt-4 flex flex-col gap-3 rounded-md bg-muted p-3"
              onSubmit={(event) => {
                event.preventDefault();
                action.mutate({ endpoint: "hide-pending", body: { reason: hideReason } });
              }}
            >
              <Field label="Warum soll dieser ungeprüfte Hinweis ausgeblendet werden?">
                <Textarea value={hideReason} onChange={(event) => setHideReason(event.target.value)} />
              </Field>
              <Button type="submit" variant="secondary" disabled={!hideReason.trim() || action.isPending}>
                <EyeOff className="h-4 w-4" />
                {action.isPending ? "Wird ausgeblendet..." : "Pending-Hinweis ausblenden"}
              </Button>
            </form>
          ) : null}
          {action.error ? <p className="mt-3 text-sm text-destructive">{action.error.message}</p> : null}
        </section>
      ) : null}

      {detail.status === "accepted" && abilities.canApplyInfoSuggestions ? (
        <section className="rounded-lg border p-4">
          <h3 className="font-semibold">In Live-Text einarbeiten</h3>
          <form
            className="mt-4 space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              apply.mutate();
            }}
          >
            <Field label="Neuer Live-Text">
              <MarkdownEditor value={newText} onChange={setNewText} />
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Vertrauensstufe" info="Gibt an, wie gut der geprüfte Infotext belegt ist: niedrig, mittel oder hoch.">
                <Select value={confidenceLevel} onChange={(event) => setConfidenceLevel(event.target.value)}>
                  <option value="low">niedrig</option>
                  <option value="medium">mittel</option>
                  <option value="high">hoch</option>
                </Select>
              </Field>
              <Field label="Source Summary">
                <Textarea value={sourceSummary} onChange={(event) => setSourceSummary(event.target.value)} />
              </Field>
            </div>
            <Field label="Change Summary">
              <Textarea value={changeSummary} onChange={(event) => setChangeSummary(event.target.value)} />
            </Field>
            {apply.error ? <p className="text-sm text-destructive">{apply.error.message}</p> : null}
            <Button type="submit" disabled={apply.isPending || !changeSummary.trim()}>
              <Send className="h-4 w-4" />
              Veröffentlichen
            </Button>
          </form>
        </section>
      ) : null}

      <section className="rounded-lg border p-4">
        <h3 className="font-semibold">Review-Historie</h3>
        <ul className="mt-3 space-y-2 text-sm">
          {detail.reviewEvents.map((event) => (
            <li key={event.id} className="rounded-md bg-muted p-3">
              {formatDate(event.createdAt)} · {event.action} · {event.oldStatus ?? "-"} →{" "}
              {event.newStatus ?? "-"}
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}

function TextPanel({ title, text }: { title: string; text: string }) {
  return (
    <section className="rounded-lg border p-4">
      <h3 className="font-semibold">{title}</h3>
      <pre className="mt-3 whitespace-pre-wrap rounded-md bg-muted p-3 text-sm">{text || "-"}</pre>
    </section>
  );
}
