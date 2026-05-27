"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ShieldCheck } from "lucide-react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

const schema = z.object({
  requestedRoleTemplateId: z.string().uuid("Bitte Rolle auswählen."),
  requestedScopeType: z.enum(["global", "state", "ihk"]),
  requestedScopeId: z.string().max(200).optional(),
  proofFileName: z.string().max(255).optional(),
  proofMimeType: z.string().max(100).optional(),
  proofContentBase64: z.string().max(2_000_000).optional(),
  proofNote: z.string().max(2000).optional(),
});

type Values = z.infer<typeof schema>;

export function PermissionRequestModal({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const roles = useQuery({
    queryKey: ["requestable-role-templates"],
    queryFn: adminApi.listRequestableRoleTemplates,
    enabled: open,
  });
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      requestedRoleTemplateId: "",
      requestedScopeType: "state",
      requestedScopeId: "",
      proofFileName: "",
      proofMimeType: "",
      proofContentBase64: "",
      proofNote: "",
    },
  });
  const scopeType = useWatch({ control: form.control, name: "requestedScopeType" });
  const mutation = useMutation({
    mutationFn: (values: Values) =>
      adminApi.requestPermissions({
        ...values,
        requestedScopeId: values.requestedScopeType === "global" ? undefined : values.requestedScopeId || undefined,
        proofFileName: values.proofFileName || undefined,
        proofMimeType: values.proofMimeType || undefined,
        proofContentBase64: values.proofContentBase64 || undefined,
        proofNote: values.proofNote || undefined,
      }),
  });

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Rechte anfragen"
      description="Die Anfrage wird von einem Admin geprüft. Du erhältst danach eine E-Mail."
    >
      <form className="space-y-4" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
        <Field label="Gewünschte Rolle" error={form.formState.errors.requestedRoleTemplateId?.message}>
          <Select {...form.register("requestedRoleTemplateId")}>
            <option value="">Rolle auswählen</option>
            {roles.data?.items.map((role) => (
              <option key={role.id} value={role.id}>
                {role.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Scope" error={form.formState.errors.requestedScopeType?.message}>
          <Select {...form.register("requestedScopeType")}>
            <option value="state">Bundesland</option>
            <option value="ihk">IHK</option>
            <option value="global">Global</option>
          </Select>
        </Field>
        {scopeType !== "global" ? (
          <Field label={scopeType === "state" ? "Bundesland" : "IHK-ID"} error={form.formState.errors.requestedScopeId?.message}>
            <Input {...form.register("requestedScopeId")} />
          </Field>
        ) : null}
        <Field label="Nachweis-Dateiname, optional" error={form.formState.errors.proofFileName?.message}>
          <Input
            type="file"
            accept="application/pdf,image/jpeg,image/png,image/webp"
            onChange={async (event) => {
              const file = event.target.files?.[0];
              form.setValue("proofFileName", file?.name ?? "");
              form.setValue("proofMimeType", file?.type ?? "");
              form.setValue("proofContentBase64", file ? await fileToBase64(file) : "");
            }}
          />
        </Field>
        <Field label="Nachweis / Begründung, optional" error={form.formState.errors.proofNote?.message}>
          <Textarea {...form.register("proofNote")} />
        </Field>
        {mutation.error ? <p className="text-sm text-destructive">{mutation.error.message}</p> : null}
        {mutation.isSuccess ? <p className="text-sm text-primary">Deine Rechteanfrage wurde eingereicht.</p> : null}
        <Button type="submit" disabled={mutation.isPending} className="w-full">
          <ShieldCheck className="h-4 w-4" />
          Anfrage einreichen
        </Button>
      </form>
    </Dialog>
  );
}

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const value = String(reader.result ?? "");
      resolve(value.includes(",") ? value.split(",")[1] ?? "" : value);
    };
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}
