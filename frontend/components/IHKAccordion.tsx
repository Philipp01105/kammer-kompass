"use client";

import Link from "next/link";
import { ChevronDown, ExternalLink, MessageSquarePlus } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { InfoSuggestionModal } from "@/components/InfoSuggestionModal";
import { InfoTextBox } from "@/components/InfoTextBox";
import { PendingHintsBlock } from "@/components/PendingHintsBlock";
import type { PublicIHKItem } from "@/types/api";

type IHKAccordionProps = {
  items: PublicIHKItem[];
};

export function IHKAccordion({ items }: IHKAccordionProps) {
  const [selected, setSelected] = useState<PublicIHKItem | null>(null);

  return (
    <div className="space-y-4">
      {items.map((item) => (
        <details key={item.id} className="rounded-lg border bg-card p-6" open={items.length === 1}>
          <summary className="flex cursor-pointer list-none items-center justify-between gap-4">
            <div>
              <h3 className="text-2xl font-semibold">{item.name}</h3>
              <p className="text-base text-muted-foreground">
                {[item.city, item.state].filter(Boolean).join(", ")}
              </p>
            </div>
            <ChevronDown className="h-5 w-5 shrink-0 text-muted-foreground" />
          </summary>

          <InfoTextBox item={item} />
          <PendingHintsBlock hints={item.pendingHints ?? []} />

          <div className="mt-5 flex flex-wrap gap-2">
            <Button onClick={() => setSelected(item)}>
              <MessageSquarePlus className="h-4 w-4" />
              Korrektur oder Ergänzung vorschlagen
            </Button>
            <Link
              className="inline-flex min-h-11 items-center justify-center rounded-md bg-secondary px-5 py-2.5 text-base font-medium hover:bg-secondary/80"
              href={`/ihk/${item.slug}`}
            >
              Detailseite öffnen
            </Link>
            {item.officialUrl ? (
              <a
                className="inline-flex min-h-11 items-center gap-2 rounded-md px-5 py-2.5 text-base font-medium underline"
                href={item.officialUrl}
                target="_blank"
                rel="nofollow noopener noreferrer"
              >
                Offizielle Website
                <ExternalLink className="h-4 w-4" />
              </a>
            ) : null}
          </div>
        </details>
      ))}
      <InfoSuggestionModal
        open={Boolean(selected)}
        ihkId={selected?.id}
        ihkName={selected?.name}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
      />
    </div>
  );
}
