"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Send } from "lucide-react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { publicApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

const schema = z.object({
  suggestedChange: z.string().min(20, "Mindestens 20 Zeichen.").max(3000),
  reason: z.string().max(2000).optional(),
  sourceUrl: z.string().url("Bitte eine gültige URL verwenden.").optional().or(z.literal("")),
  sourceNote: z.string().max(2000).optional(),
  submittedEmail: z.string().email("Bitte eine gültige E-Mail verwenden.").optional().or(z.literal("")),
  honeypot: z.string().max(0),
});

type Values = z.infer<typeof schema>;

type Props = {
  open: boolean;
  ihkId?: string;
  ihkName?: string;
  onOpenChange: (open: boolean) => void;
};

export function InfoSuggestionModal({ open, ihkId, ihkName, onOpenChange }: Props) {
  const queryClient = useQueryClient();
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      suggestedChange: "",
      reason: "",
      sourceUrl: "",
      sourceNote: "",
      submittedEmail: "",
      honeypot: "",
    },
  });
  const mutation = useMutation({
    mutationFn: (values: Values) =>
      publicApi.submitInfoSuggestion({
        ihkId,
        suggestedChange: values.suggestedChange,
        reason: values.reason || undefined,
        sourceUrl: values.sourceUrl || undefined,
        sourceNote: values.sourceNote || undefined,
        submittedEmail: values.submittedEmail || undefined,
        honeypot: values.honeypot,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["public-ihks"] });
      form.reset();
    },
  });

  const successMessage = mutation.isSuccess
    ? "Danke. Dein Hinweis wurde eingereicht und erscheint bis zur Prüfung als ungeprüfter Community-Hinweis."
    : null;

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Korrektur oder Ergänzung vorschlagen"
      description={ihkName ? `Für ${ihkName}` : undefined}
    >
      <form
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
      >
        <p className="rounded-md bg-muted p-3 text-sm text-muted-foreground">
          Dein Hinweis muss auf Deutsch sein. Beleidigende, diskriminierende oder unsachliche
          Inhalte werden automatisch abgelehnt. Wenn der Hinweis die automatische Prüfung
          besteht, erscheint er bis zur redaktionellen Prüfung als ungeprüfter Community-Hinweis.
        </p>
        <input type="text" className="hidden" tabIndex={-1} {...form.register("honeypot")} />
        <Field label="Wie sollte es geändert oder ergänzt werden?" error={form.formState.errors.suggestedChange?.message}>
          <Textarea {...form.register("suggestedChange")} />
        </Field>
        <Field label="Quelle / Begründung" error={form.formState.errors.reason?.message}>
          <Textarea {...form.register("reason")} />
        </Field>
        <Field label="Link zur Quelle, optional" error={form.formState.errors.sourceUrl?.message}>
          <Input {...form.register("sourceUrl")} />
        </Field>
        <Field label="Quellenhinweis, optional" error={form.formState.errors.sourceNote?.message}>
          <Input {...form.register("sourceNote")} />
        </Field>
        <Field label="E-Mail für Rückfragen, optional" error={form.formState.errors.submittedEmail?.message}>
          <Input type="email" {...form.register("submittedEmail")} />
        </Field>
        {mutation.error ? <p className="text-sm text-destructive">{mutation.error.message}</p> : null}
        {successMessage ? <p className="text-sm text-primary">{successMessage}</p> : null}
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Schließen
          </Button>
          <Button type="submit" disabled={mutation.isPending || !ihkId}>
            <Send className="h-4 w-4" />
            Einreichen
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
