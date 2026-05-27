"use client";

import { useQuery } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { Header } from "@/components/Header";
import { IHKAccordion } from "@/components/IHKAccordion";
import { SearchBar } from "@/components/SearchBar";
import { publicApi } from "@/lib/api";

export default function PublicHomePage() {
  const [query, setQuery] = useState("");
  const handleQueryChange = useCallback((value: string) => setQuery(value), []);
  const ihks = useQuery({
    queryKey: ["public-ihks", query],
    queryFn: () => publicApi.listIHKs(query),
  });

  return (
    <main>
      <Header />
      <section className="mx-auto max-w-7xl px-5 py-10">
        <div className="mb-6">
          <h1 className="text-4xl font-semibold">Inoffizielle Community-Datenbank für IHK-spezifische Hinweise.</h1>
          <p className="mt-3 max-w-4xl text-lg text-muted-foreground">
            Verbindliche Informationen erhältst du immer bei deiner zuständigen Kammer.
          </p>
        </div>
        <SearchBar value={query} onChange={handleQueryChange} />
        <div className="mt-6">
          {ihks.isLoading ? <p className="text-base text-muted-foreground">IHKs werden geladen...</p> : null}
          {ihks.error ? <p className="text-base text-destructive">{ihks.error.message}</p> : null}
          {ihks.data?.items.length ? (
            <IHKAccordion items={ihks.data.items} />
          ) : !ihks.isLoading ? (
            <EmptyState title="Keine IHKs gefunden" text="Passe die Suche an." />
          ) : null}
        </div>
      </section>
    </main>
  );
}
