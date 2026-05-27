"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";

export default function PermissionRequestDetailPage() {
  const params = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const detail = useQuery({
    queryKey: ["permission-request", params.id],
    queryFn: () => adminApi.getPermissionRequest(params.id),
  });
  const decide = useMutation({
    mutationFn: (action: "approve" | "reject") => adminApi.decidePermissionRequest(params.id, action),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["permission-request", params.id] });
      queryClient.invalidateQueries({ queryKey: ["permission-requests"] });
    },
  });
  const item = detail.data;

  return (
    <div className="space-y-5">
      <Link href="/admin/permission-requests" className="text-sm underline">
        Zurück zu Rechteanfragen
      </Link>
      {item ? (
        <>
          <div className="rounded-md border p-5">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <h2 className="text-2xl font-semibold">{item.displayName}</h2>
                <p className="text-muted-foreground">{item.email}</p>
              </div>
              <span className="rounded-md bg-muted px-3 py-1 text-sm">{item.status}</span>
            </div>
            <dl className="mt-5 grid gap-4 md:grid-cols-2">
              <Info label="Typ" value={item.requestType === "registration" ? "Registrierung mit Rechten" : "Rechteanfrage"} />
              <Info label="Rolle" value={item.requestedRoleName} />
              <Info label="Scope" value={`${item.requestedScopeType}${item.requestedScopeId ? `: ${item.requestedScopeId}` : ""}`} />
              <Info label="Nachweis-Datei" value={item.proofFileName || "-"} />
            </dl>
            {item.proofContentBase64 ? (
              <a
                className="mt-4 inline-flex underline"
                href={`data:${item.proofMimeType || "application/octet-stream"};base64,${item.proofContentBase64}`}
                download={item.proofFileName || "nachweis"}
              >
                Nachweis öffnen
              </a>
            ) : null}
            {item.proofNote ? (
              <div className="mt-5">
                <h3 className="font-medium">Nachweis / Begründung</h3>
                <p className="mt-2 whitespace-pre-wrap text-muted-foreground">{item.proofNote}</p>
              </div>
            ) : null}
            {item.status === "pending" ? (
              <div className="mt-5 flex gap-2">
                <Button onClick={() => decide.mutate("approve")}>Akzeptieren</Button>
                <Button variant="secondary" onClick={() => decide.mutate("reject")}>
                  Ablehnen
                </Button>
              </div>
            ) : null}
          </div>

          <section className="rounded-md border p-5">
            <h3 className="text-xl font-semibold">Bisherige Vorschläge</h3>
            <div className="mt-4 space-y-3">
              {item.activities.length === 0 ? <p className="text-muted-foreground">Keine Vorschläge gefunden.</p> : null}
              {item.activities.map((activity) => (
                <div key={`${activity.type}-${activity.id}`} className="flex flex-wrap items-center justify-between gap-3 rounded-md bg-muted p-3">
                  <div>
                    <div className="font-medium">{activity.type === "info_suggestion" ? "Info-Vorschlag" : "IHK-Vorschlag"}</div>
                    <div className="text-sm text-muted-foreground">Status: {activity.status}</div>
                  </div>
                  <Link href={activity.href} className="underline">
                    Änderung öffnen
                  </Link>
                </div>
              ))}
            </div>
          </section>
        </>
      ) : null}
      {detail.error ? <p className="text-sm text-destructive">{detail.error.message}</p> : null}
      {decide.error ? <p className="text-sm text-destructive">{decide.error.message}</p> : null}
    </div>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className="font-medium">{value}</dd>
    </div>
  );
}
