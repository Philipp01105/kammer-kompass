"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";

export default function PermissionRequestsPage() {
  const [status, setStatus] = useState("pending");
  const queryClient = useQueryClient();
  const requests = useQuery({
    queryKey: ["permission-requests", status],
    queryFn: () => adminApi.listPermissionRequests(status),
  });
  const decide = useMutation({
    mutationFn: ({ id, action }: { id: string; action: "approve" | "reject" }) =>
      adminApi.decidePermissionRequest(id, action),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["permission-requests"] }),
  });

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-2xl font-semibold">Rechteanfragen</h2>
        <Select value={status} onChange={(event) => setStatus(event.target.value)} className="max-w-56">
          <option value="pending">Offen</option>
          <option value="approved">Akzeptiert</option>
          <option value="rejected">Abgelehnt</option>
        </Select>
      </div>
      <div className="overflow-hidden rounded-md border">
        <table className="w-full text-left text-sm">
          <thead className="bg-muted text-muted-foreground">
            <tr>
              <th className="p-3">Nutzer</th>
              <th className="p-3">Typ</th>
              <th className="p-3">Rolle</th>
              <th className="p-3">Scope</th>
              <th className="p-3">Nachweis</th>
              <th className="p-3">Aktion</th>
            </tr>
          </thead>
          <tbody>
            {requests.data?.items.map((item) => (
              <tr key={item.id} className="border-t">
                <td className="p-3">
                  <div className="font-medium">{item.displayName}</div>
                  <div className="text-muted-foreground">{item.email}</div>
                </td>
                <td className="p-3">{item.requestType === "registration" ? "Registrierung" : "Rechteanfrage"}</td>
                <td className="p-3">{item.requestedRoleName}</td>
                <td className="p-3">
                  {item.requestedScopeType}
                  {item.requestedScopeId ? `: ${item.requestedScopeId}` : ""}
                </td>
                <td className="p-3">{item.proofNote || "-"}</td>
                <td className="p-3">
                  <div className="flex flex-wrap gap-2">
                    <Link className="underline" href={`/admin/permission-requests/${item.id}`}>
                      Details
                    </Link>
                    {item.status === "pending" ? (
                      <>
                        <Button onClick={() => decide.mutate({ id: item.id, action: "approve" })}>
                          Akzeptieren
                        </Button>
                        <Button variant="secondary" onClick={() => decide.mutate({ id: item.id, action: "reject" })}>
                          Ablehnen
                        </Button>
                      </>
                    ) : null}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {requests.error ? <p className="text-sm text-destructive">{requests.error.message}</p> : null}
      {decide.error ? <p className="text-sm text-destructive">{decide.error.message}</p> : null}
    </div>
  );
}
