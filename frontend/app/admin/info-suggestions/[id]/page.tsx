"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { InfoSuggestionDetail } from "@/components/admin/InfoSuggestionDetail";

export default function InfoSuggestionDetailPage() {
  const params = useParams<{ id: string }>();
  const detail = useQuery({
    queryKey: ["admin-info-suggestion", params.id],
    queryFn: () => adminApi.getInfoSuggestion(params.id),
  });

  if (detail.isLoading) return <p className="text-sm text-muted-foreground">Vorschlag wird geladen...</p>;
  if (detail.error) return <p className="text-sm text-destructive">{detail.error.message}</p>;
  if (!detail.data) return null;
  return <InfoSuggestionDetail detail={detail.data} />;
}
