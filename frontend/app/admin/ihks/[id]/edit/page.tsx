"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { IHKEditor } from "@/components/admin/IHKEditor";

export default function EditIHKPage() {
  const params = useParams<{ id: string }>();
  const slug =
    typeof window === "undefined"
      ? null
      : new URLSearchParams(window.location.search).get("slug");
  const ihks = useQuery({
    queryKey: ["admin-ihks"],
    queryFn: () => adminApi.listIHKs(),
  });
  const ihk = ihks.data?.items.find((item) => item.id === params.id || item.slug === slug);

  if (ihks.isLoading) return <p className="text-sm text-muted-foreground">IHK wird geladen...</p>;
  if (ihks.error) return <p className="text-sm text-destructive">{ihks.error.message}</p>;
  if (!ihk) {
    return (
      <p className="text-sm text-destructive">
        IHK nicht gefunden. Öffne den Editor über die IHK-Liste, damit der Slug für den MVP-Editor bekannt ist.
      </p>
    );
  }
  return <IHKEditor ihk={ihk} />;
}
