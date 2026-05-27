"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { adminApi } from "@/lib/api";
import type { AdminUser } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";

export function UserRoleEditor({ user }: { user: AdminUser }) {
  const queryClient = useQueryClient();
  const [roleTemplateId, setRoleTemplateId] = useState("");
  const [scopeType, setScopeType] = useState("global");
  const [scopeId, setScopeId] = useState("");
  const roles = useQuery({
    queryKey: ["role-templates"],
    queryFn: adminApi.listRoleTemplates,
  });
  const assignments = useQuery({
    queryKey: ["user-roles", user.id],
    queryFn: () => adminApi.listUserRoles(user.id),
  });
  const assign = useMutation({
    mutationFn: () =>
      adminApi.assignUserRole(user.id, {
        roleTemplateId,
        scopeType,
        scopeId: scopeType === "global" ? undefined : scopeId,
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["user-roles", user.id] }),
  });
  const revoke = useMutation({
    mutationFn: (assignmentId: string) => adminApi.revokeUserRole(user.id, assignmentId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["user-roles", user.id] }),
  });

  return (
    <section className="rounded-lg border p-4">
      <h3 className="font-semibold">{user.displayName}</h3>
      <p className="text-sm text-muted-foreground">{user.email}</p>
      <form
        className="mt-4 grid gap-3 md:grid-cols-[1fr_160px_1fr_auto]"
        onSubmit={(event) => {
          event.preventDefault();
          assign.mutate();
        }}
      >
        <Field label="Role Template">
          <Select value={roleTemplateId} onChange={(event) => setRoleTemplateId(event.target.value)}>
            <option value="">Auswählen</option>
            {roles.data?.items.map((role) => (
              <option key={role.id} value={role.id}>
                {role.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Scope Type">
          <Select value={scopeType} onChange={(event) => setScopeType(event.target.value)}>
            <option value="global">global</option>
            <option value="state">state</option>
            <option value="ihk">ihk</option>
          </Select>
        </Field>
        <Field label="Scope ID">
          <Input value={scopeId} onChange={(event) => setScopeId(event.target.value)} disabled={scopeType === "global"} />
        </Field>
        <Button type="submit" className="self-end" disabled={!roleTemplateId || assign.isPending}>
          <Plus className="h-4 w-4" />
          Rolle hinzufügen
        </Button>
      </form>
      <ul className="mt-4 space-y-2 text-sm">
        {assignments.data?.items.map((assignment) => (
          <li key={assignment.id} className="flex flex-wrap items-center justify-between gap-3 rounded-md bg-muted p-3">
            <span>
              {assignment.roleName} · {assignment.scopeType}
              {assignment.scopeId ? `:${assignment.scopeId}` : ""}
            </span>
            <Button variant="secondary" onClick={() => revoke.mutate(assignment.id)}>
              <Trash2 className="h-4 w-4" />
              Entfernen
            </Button>
          </li>
        ))}
      </ul>
    </section>
  );
}
