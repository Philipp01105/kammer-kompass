"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { Header } from "@/components/Header";
import { InfoTextBox } from "@/components/InfoTextBox";
import { PendingHintsBlock } from "@/components/PendingHintsBlock";
import { InfoSuggestionModal } from "@/components/InfoSuggestionModal";
import { Button } from "@/components/ui/button";
import { publicApi } from "@/lib/api";
import { useState } from "react";

export default function PublicIHKDetailPage() {
  const params = useParams<{ slug: string }>();
  const [suggestOpen, setSuggestOpen] = useState(false);
  const ihk = useQuery({
    queryKey: ["public-ihk", params.slug],
    queryFn: () => publicApi.getIHK(params.slug),
  });

  return (
    <main>
      <Header />
      <section className="mx-auto max-w-7xl px-5 py-10">
        <Link className="mb-5 inline-flex items-center gap-2 text-base underline" href="/">
          <ArrowLeft className="h-4 w-4" />
          Zur Suche
        </Link>
        {ihk.isLoading ? <p className="text-base text-muted-foreground">IHK wird geladen...</p> : null}
        {ihk.error ? <p className="text-base text-destructive">{ihk.error.message}</p> : null}
        {ihk.data ? (
          <article>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h1 className="text-4xl font-semibold">{ihk.data.name}</h1>
                <p className="text-muted-foreground">
                  {[ihk.data.city, ihk.data.state].filter(Boolean).join(", ")}
                </p>
              </div>
              <Button onClick={() => setSuggestOpen(true)}>Korrektur oder Ergänzung vorschlagen</Button>
            </div>
            <InfoTextBox item={ihk.data} />
            <PendingHintsBlock hints={ihk.data.pendingHints ?? []} />
          </article>
        ) : null}
      </section>
      <InfoSuggestionModal
        open={suggestOpen}
        ihkId={ihk.data?.id}
        ihkName={ihk.data?.name}
        onOpenChange={setSuggestOpen}
      />
    </main>
  );
}
