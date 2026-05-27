"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";

export function ModerationTermsTable() {
  const queryClient = useQueryClient();
  const [term, setTerm] = useState("");
  const [category, setCategory] = useState("insult");
  const [severity, setSeverity] = useState("medium");
  const terms = useQuery({
    queryKey: ["moderation-terms"],
    queryFn: adminApi.listModerationTerms,
  });
  const create = useMutation({
    mutationFn: () => adminApi.createModerationTerm({ term, category, severity }),
    onSuccess: async () => {
      setTerm("");
      await queryClient.invalidateQueries({ queryKey: ["moderation-terms"] });
    },
  });
  const remove = useMutation({
    mutationFn: (id: string) => adminApi.deleteModerationTerm(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["moderation-terms"] }),
  });

  return (
    <div className="space-y-6">
      <form
        className="rounded-lg border p-4"
        onSubmit={(event) => {
          event.preventDefault();
          create.mutate();
        }}
      >
        <h2 className="font-semibold">Begriff hinzufügen</h2>
        <div className="mt-4 grid gap-4 md:grid-cols-[1fr_180px_180px_auto]">
          <Field label="Term">
            <Input value={term} onChange={(event) => setTerm(event.target.value)} />
          </Field>
          <Field label="Category">
            <Select value={category} onChange={(event) => setCategory(event.target.value)}>
              <option value="insult">insult</option>
              <option value="slur">slur</option>
              <option value="threat">threat</option>
              <option value="sexual">sexual</option>
              <option value="spam">spam</option>
              <option value="other">other</option>
            </Select>
          </Field>
          <Field label="Severity">
            <Select value={severity} onChange={(event) => setSeverity(event.target.value)}>
              <option value="low">low</option>
              <option value="medium">medium</option>
              <option value="high">high</option>
            </Select>
          </Field>
          <Button type="submit" className="self-end" disabled={create.isPending || !term.trim()}>
            <Plus className="h-4 w-4" />
            Hinzufügen
          </Button>
        </div>
        {create.error ? <p className="mt-3 text-sm text-destructive">{create.error.message}</p> : null}
      </form>
      <section className="overflow-x-auto rounded-lg border">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead className="border-b text-muted-foreground">
            <tr>
              <th className="p-3">Begriff</th>
              <th className="p-3">Kategorie</th>
              <th className="p-3">Severity</th>
              <th className="p-3">Aktiv</th>
              <th className="p-3">Aktion</th>
            </tr>
          </thead>
          <tbody>
            {terms.data?.items.map((item) => (
              <tr key={item.id} className="border-b">
                <td className="p-3">{item.term}</td>
                <td className="p-3">{item.category}</td>
                <td className="p-3">{item.severity}</td>
                <td className="p-3">{item.isActive ? "ja" : "nein"}</td>
                <td className="p-3">
                  <Button variant="secondary" onClick={() => remove.mutate(item.id)}>
                    <Trash2 className="h-4 w-4" />
                    Deaktivieren
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
