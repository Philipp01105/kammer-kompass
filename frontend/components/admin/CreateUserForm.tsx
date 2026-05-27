"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { UserPlus } from "lucide-react";
import { useState } from "react";
import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";

export function CreateUserForm() {
  const queryClient = useQueryClient();
  const roles = useQuery({
    queryKey: ["role-templates"],
    queryFn: adminApi.listRoleTemplates,
  });
  const [form, setForm] = useState({
    email: "",
    displayName: "",
    password: "",
    roleTemplateId: "",
    scopeType: "global",
    scopeId: "",
  });
  const create = useMutation({
    mutationFn: () =>
      adminApi.createUser({
        email: form.email,
        displayName: form.displayName,
        password: form.password,
        roleTemplateId: form.roleTemplateId || undefined,
        scopeType: form.roleTemplateId ? form.scopeType : undefined,
        scopeId: form.roleTemplateId && form.scopeType !== "global" ? form.scopeId : undefined,
      }),
    onSuccess: async () => {
      setForm({
        email: "",
        displayName: "",
        password: "",
        roleTemplateId: "",
        scopeType: "global",
        scopeId: "",
      });
      await queryClient.invalidateQueries({ queryKey: ["admin-users"] });
    },
  });

  return (
    <form
      className="rounded-lg border p-4"
      onSubmit={(event) => {
        event.preventDefault();
        create.mutate();
      }}
    >
      <div className="flex flex-col gap-1">
        <h3 className="font-semibold">Nutzer oder Admin hinzufügen</h3>
        <p className="text-sm text-muted-foreground">
          Rolle optional direkt zuweisen. Für einen normalen globalen Admin wähle `admin` und Scope `global`.
        </p>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-2">
        <Field label="E-Mail">
          <Input
            type="email"
            value={form.email}
            onChange={(event) => setForm({ ...form, email: event.target.value })}
          />
        </Field>
        <Field label="Display Name">
          <Input
            value={form.displayName}
            onChange={(event) => setForm({ ...form, displayName: event.target.value })}
          />
        </Field>
        <Field label="Initiales Passwort">
          <Input
            type="password"
            value={form.password}
            onChange={(event) => setForm({ ...form, password: event.target.value })}
          />
        </Field>
        <Field label="Rolle, optional">
          <Select
            value={form.roleTemplateId}
            onChange={(event) => setForm({ ...form, roleTemplateId: event.target.value })}
          >
            <option value="">Keine Rolle direkt zuweisen</option>
            {roles.data?.items.map((role) => (
              <option key={role.id} value={role.id}>
                {role.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Scope Type">
          <Select
            value={form.scopeType}
            disabled={!form.roleTemplateId}
            onChange={(event) => setForm({ ...form, scopeType: event.target.value })}
          >
            <option value="global">global</option>
            <option value="state">state</option>
            <option value="ihk">ihk</option>
          </Select>
        </Field>
        <Field label="Scope ID">
          <Input
            value={form.scopeId}
            disabled={!form.roleTemplateId || form.scopeType === "global"}
            onChange={(event) => setForm({ ...form, scopeId: event.target.value })}
          />
        </Field>
      </div>

      {create.error ? <p className="mt-3 text-sm text-destructive">{create.error.message}</p> : null}
      {create.isSuccess ? <p className="mt-3 text-sm text-primary">Nutzer wurde angelegt.</p> : null}

      <Button
        type="submit"
        className="mt-4"
        disabled={create.isPending || !form.email || !form.displayName || form.password.length < 10}
      >
        <UserPlus className="h-4 w-4" />
        Nutzer hinzufügen
      </Button>
    </form>
  );
}
