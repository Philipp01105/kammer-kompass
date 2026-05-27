"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { adminApi } from "@/lib/api";
import { formatDate } from "@/lib/markdown";
import { SearchBar } from "@/components/SearchBar";

export default function AdminIHKsPage() {
  const [query, setQuery] = useState("");
  const handleQueryChange = useCallback((value: string) => setQuery(value), []);
  const ihks = useQuery({
    queryKey: ["admin-ihks", query],
    queryFn: () => adminApi.listIHKs(query),
  });

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-3xl font-semibold">IHKs</h2>
      </div>
      <SearchBar value={query} onChange={handleQueryChange} />
      <div className="overflow-x-auto rounded-lg border">
        <table className="w-full min-w-[900px] text-left text-base">
          <thead className="border-b text-muted-foreground">
            <tr>
              <th className="p-3">Name</th>
              <th className="p-3">Slug</th>
              <th className="p-3">Stadt</th>
              <th className="p-3">Bundesland</th>
              <th className="p-3">Aktiv</th>
              <th className="p-3">Aktualisiert</th>
              <th className="p-3">Aktion</th>
            </tr>
          </thead>
          <tbody>
            {ihks.data?.items.map((item) => (
              <tr key={item.id} className="border-b">
                <td className="p-3">{item.name}</td>
                <td className="p-3">{item.slug}</td>
                <td className="p-3">{item.city ?? "-"}</td>
                <td className="p-3">{item.state}</td>
                <td className="p-3">{item.isActive ? "ja" : "nein"}</td>
                <td className="p-3">{formatDate(item.updatedAt)}</td>
                <td className="p-3">
                  <Link href={`/admin/ihks/${item.id}/edit?slug=${encodeURIComponent(item.slug)}`} className="underline">
                    Bearbeiten
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {ihks.error ? <p className="text-sm text-destructive">{ihks.error.message}</p> : null}
    </div>
  );
}
