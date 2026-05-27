"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Save } from "lucide-react";
import { useState } from "react";
import { adminApi, publicApi } from "@/lib/api";
import type { AdminIHK, PublicIHKItem } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { MarkdownEditor } from "@/components/admin/MarkdownEditor";
import { VersionHistory } from "@/components/admin/VersionHistory";

type ConfidenceLevel = PublicIHKItem["info"]["confidenceLevel"];

export function IHKEditor({ ihk }: { ihk: AdminIHK }) {
  const queryClient = useQueryClient();
  const [officialUrl, setOfficialUrl] = useState(ihk.officialUrl ?? "");
  const detail = useQuery({
    queryKey: ["public-ihk", ihk.slug],
    queryFn: () => publicApi.getIHK(ihk.slug),
  });
  const updateCore = useMutation({
    mutationFn: () =>
      adminApi.updateIHK(ihk.id, {
        officialUrl: officialUrl.trim() || null,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-ihks"] });
      await queryClient.invalidateQueries({ queryKey: ["public-ihk", ihk.slug] });
      await queryClient.invalidateQueries({ queryKey: ["public-ihks"] });
    },
  });

  return (
    <div className="space-y-6">
      <section className="rounded-lg border p-4">
        <h2 className="text-xl font-semibold">{ihk.name}</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          {[ihk.city, ihk.state].filter(Boolean).join(", ")} · {ihk.slug}
        </p>
      </section>

      <section className="rounded-lg border p-4">
        <h3 className="font-semibold">Stammdaten</h3>
        <form
          className="mt-4 space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            updateCore.mutate();
          }}
        >
          <Field label="Official URL">
            <Input value={officialUrl} onChange={(event) => setOfficialUrl(event.target.value)} />
          </Field>
          {updateCore.error ? <p className="text-sm text-destructive">{updateCore.error.message}</p> : null}
          {updateCore.isSuccess ? <p className="text-sm text-primary">Gespeichert.</p> : null}
          <Button type="submit" disabled={updateCore.isPending}>
            <Save className="h-4 w-4" />
            Stammdaten speichern
          </Button>
        </form>
      </section>

      <section className="rounded-lg border p-4">
        <h3 className="font-semibold">Infotext direkt veröffentlichen</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Kein Autosave. Live-Daten ändern sich erst nach explizitem Publish.
        </p>
        {detail.data ? (
          <IHKEditorForm key={`${detail.data.id}-${detail.data.info.updatedAt}`} ihk={ihk} detail={detail.data} />
        ) : (
          <div className="mt-4 rounded-md bg-muted p-3 text-sm text-muted-foreground">Infotext wird geladen...</div>
        )}
      </section>
      <VersionHistory ihkId={ihk.id} />
    </div>
  );
}

function IHKEditorForm({ ihk, detail }: { ihk: AdminIHK; detail: PublicIHKItem }) {
  const queryClient = useQueryClient();
  const [newText, setNewText] = useState(detail.info.currentText);
  const [confidenceLevel, setConfidenceLevel] = useState(detail.info.confidenceLevel);
  const [sourceSummary, setSourceSummary] = useState(detail.info.sourceSummary ?? "");
  const [changeSummary, setChangeSummary] = useState("");

  const publish = useMutation({
    mutationFn: () =>
      adminApi.publishIHKInfo(ihk.id, {
        newText,
        confidenceLevel,
        sourceSummary: sourceSummary || undefined,
        changeSummary,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["public-ihk", ihk.slug] });
      await queryClient.invalidateQueries({ queryKey: ["public-ihks"] });
      await queryClient.invalidateQueries({ queryKey: ["admin-ihk-versions", ihk.id] });
      setChangeSummary("");
    },
  });

  return (
    <form
      className="mt-4 space-y-4"
      onSubmit={(event) => {
        event.preventDefault();
        publish.mutate();
      }}
    >
      <Field label="Infotext Markdown">
        <MarkdownEditor value={newText} onChange={setNewText} />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Vertrauensstufe" info="Gibt an, wie gut der geprüfte Infotext belegt ist: niedrig, mittel oder hoch.">
          <Select value={confidenceLevel} onChange={(event) => setConfidenceLevel(event.target.value as ConfidenceLevel)}>
            <option value="low">niedrig</option>
            <option value="medium">mittel</option>
            <option value="high">hoch</option>
          </Select>
        </Field>
        <Field label="Source Summary">
          <Input value={sourceSummary} onChange={(event) => setSourceSummary(event.target.value)} />
        </Field>
      </div>
      <Field label="Change Summary">
        <Textarea value={changeSummary} onChange={(event) => setChangeSummary(event.target.value)} />
      </Field>
      {publish.error ? <p className="text-sm text-destructive">{publish.error.message}</p> : null}
      {publish.isSuccess ? <p className="text-sm text-primary">Veröffentlicht.</p> : null}
      <Button type="submit" disabled={publish.isPending || !changeSummary.trim()}>
        <Save className="h-4 w-4" />
        Veröffentlichen
      </Button>
    </form>
  );
}
