"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";

const cards = [
  { title: "Neue Info-Vorschläge", href: "/admin/info-suggestions", query: "submitted" },
  { title: "Info-Vorschläge in Prüfung", href: "/admin/info-suggestions", query: "under_review" },
  { title: "Akzeptierte Info-Vorschläge", href: "/admin/info-suggestions", query: "accepted" },
];

export default function AdminDashboardPage() {
  const pendingHints = useQuery({
    queryKey: ["admin-info-suggestions", "pending-visible"],
    queryFn: () => adminApi.listInfoSuggestions(undefined, true),
  });

  return (
    <div className="space-y-6">
      <h2 className="text-3xl font-semibold">Dashboard</h2>
      <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
        {cards.map((card) => (
          <Link
            key={card.title}
            href={`${card.href}?status=${card.query}`}
            className="rounded-lg border p-5 hover:bg-muted"
          >
            <p className="text-base text-muted-foreground">Queue</p>
            <h3 className="mt-2 text-lg font-semibold">{card.title}</h3>
          </Link>
        ))}
        <Link href="/admin/info-suggestions?publicPendingVisible=true" className="rounded-lg border p-5 hover:bg-muted">
          <p className="text-base text-muted-foreground">Öffentlich sichtbar</p>
          <h3 className="mt-2 text-lg font-semibold">Pending-Hinweise</h3>
          <p className="mt-2 text-3xl font-semibold">{pendingHints.data?.items.length ?? "-"}</p>
        </Link>
        <Link href="/admin/ihks" className="rounded-lg border p-5 hover:bg-muted">
          <p className="text-base text-muted-foreground">Redaktion</p>
          <h3 className="mt-2 text-lg font-semibold">Letzte Veröffentlichungen</h3>
        </Link>
      </div>
    </div>
  );
}
