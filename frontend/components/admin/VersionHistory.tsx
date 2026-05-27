"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RotateCcw } from "lucide-react";
import { useState } from "react";
import { adminApi } from "@/lib/api";
import { formatDate } from "@/lib/markdown";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

export function VersionHistory({ ihkId }: { ihkId: string }) {
  const queryClient = useQueryClient();
  const [versionId, setVersionId] = useState("");
  const [changeSummary, setChangeSummary] = useState("");
  const [confidenceLevel, setConfidenceLevel] = useState("medium");
  const versions = useQuery({
    queryKey: ["admin-ihk-versions", ihkId],
    queryFn: () => adminApi.listIHKVersions(ihkId),
  });
  const rollback = useMutation({
    mutationFn: () =>
      adminApi.rollbackIHKInfo(ihkId, {
        versionId,
        confidenceLevel,
        sourceSummary: "Rollback auf frühere redaktionelle Version",
        changeSummary,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-ihk-versions", ihkId] });
      await queryClient.invalidateQueries({ queryKey: ["public-ihks"] });
      setVersionId("");
      setChangeSummary("");
    },
  });

  return (
    <section className="rounded-lg border p-4">
      <h2 className="text-lg font-semibold">Versionen</h2>
      <p className="mt-1 text-sm text-muted-foreground">
        Rollback erzeugt eine neue Version. Alte Versionen werden nicht gelöscht.
      </p>
      <div className="mt-4 overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead className="border-b text-muted-foreground">
            <tr>
              <th className="py-2">Version</th>
              <th className="py-2">Datum</th>
              <th className="py-2">Geändert von</th>
              <th className="py-2">Change Summary</th>
              <th className="py-2">Aktion</th>
            </tr>
          </thead>
          <tbody>
            {versions.data?.items.map((version) => (
              <tr key={version.id} className="border-b">
                <td className="py-2">{version.versionNumber}</td>
                <td className="py-2">{formatDate(version.createdAt)}</td>
                <td className="py-2">{version.changedBy ?? "unbekannt"}</td>
                <td className="py-2">{version.changeSummary}</td>
                <td className="py-2">
                  <Button variant="secondary" onClick={() => setVersionId(version.id)}>
                    Auswählen
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {versionId ? (
        <form
          className="mt-4 space-y-3 rounded-md bg-muted p-4"
          onSubmit={(event) => {
            event.preventDefault();
            rollback.mutate();
          }}
        >
          <Field label="Vertrauensstufe" info="Gibt an, wie gut der geprüfte Infotext belegt ist: niedrig, mittel oder hoch.">
            <Select value={confidenceLevel} onChange={(event) => setConfidenceLevel(event.target.value)}>
              <option value="low">niedrig</option>
              <option value="medium">mittel</option>
              <option value="high">hoch</option>
            </Select>
          </Field>
          <Field label="Rollback Change Summary">
            <Textarea value={changeSummary} onChange={(event) => setChangeSummary(event.target.value)} />
          </Field>
          {rollback.error ? <p className="text-sm text-destructive">{rollback.error.message}</p> : null}
          <Button type="submit" disabled={rollback.isPending || !changeSummary.trim()}>
            <RotateCcw className="h-4 w-4" />
            Rollback veröffentlichen
          </Button>
        </form>
      ) : null}
    </section>
  );
}
