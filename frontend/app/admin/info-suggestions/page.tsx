"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { adminApi } from "@/lib/api";
import { STATUS_OPTIONS } from "@/lib/constants";
import { formatDate } from "@/lib/markdown";
import type { ModerationStatus } from "@/types/api";
import { StatusBadge } from "@/components/admin/StatusBadge";
import { Select } from "@/components/ui/select";

export default function InfoSuggestionsPage() {
  const [status, setStatus] = useState<ModerationStatus>(() => {
    if (typeof window === "undefined") return "submitted";
    return (new URLSearchParams(window.location.search).get("status") || "submitted") as ModerationStatus;
  });
  const publicPendingVisible =
    typeof window !== "undefined" && new URLSearchParams(window.location.search).get("publicPendingVisible") === "true"
      ? true
      : undefined;
  const query = useQuery({
    queryKey: ["admin-info-suggestions", status, publicPendingVisible],
    queryFn: () => adminApi.listInfoSuggestions(status, publicPendingVisible),
  });

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-3xl font-semibold">Info-Vorschläge</h2>
        <Select
          value={status}
          onChange={(event) => {
            const next = event.target.value as ModerationStatus;
            setStatus(next);
            window.history.replaceState(null, "", `/admin/info-suggestions?status=${next}`);
          }}
          className="sm:w-56"
        >
          {STATUS_OPTIONS.map((item) => (
            <option key={item} value={item}>
              {item}
            </option>
          ))}
        </Select>
      </div>
      <div className="overflow-x-auto rounded-lg border">
        <table className="w-full min-w-[900px] text-left text-base">
          <thead className="border-b text-muted-foreground">
            <tr>
              <th className="p-3">IHK ID</th>
              <th className="p-3">Status</th>
              <th className="p-3">Pending sichtbar</th>
              <th className="p-3">Eingereicht</th>
              <th className="p-3">Aktion</th>
            </tr>
          </thead>
          <tbody>
            {query.data?.items.map((item) => (
              <tr key={item.id} className="border-b">
                <td className="p-3 font-mono text-xs">{item.ihkId}</td>
                <td className="p-3"><StatusBadge status={item.status} /></td>
                <td className="p-3">{item.publicPendingVisible ? "ja" : "nein"}</td>
                <td className="p-3">{formatDate(item.createdAt)}</td>
                <td className="p-3">
                  <Link href={`/admin/info-suggestions/${item.id}`} className="underline">
                    Öffnen
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {query.error ? <p className="text-sm text-destructive">{query.error.message}</p> : null}
    </div>
  );
}
