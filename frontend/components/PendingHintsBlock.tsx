import type { PendingHint } from "@/types/api";
import { formatDate } from "@/lib/markdown";

type PendingHintsBlockProps = {
  hints: PendingHint[];
};

export function PendingHintsBlock({ hints }: PendingHintsBlockProps) {
  if (hints.length === 0) return null;

  return (
    <section className="mt-5 rounded-lg border border-dashed border-primary/70 bg-primary/10 p-5">
      <h4 className="font-semibold text-foreground">Ungeprüfte Community-Hinweise</h4>
      <p className="mt-1 text-base text-muted-foreground">
        Diese Hinweise wurden automatisch vorgefiltert, aber noch nicht redaktionell geprüft.
        Sie können falsch, veraltet oder unvollständig sein.
      </p>

      <ul className="mt-3 space-y-3">
        {hints.map((hint) => (
          <li key={hint.id} className="rounded-md border bg-background p-4">
            <p className="whitespace-pre-wrap text-base">{hint.publicPendingText}</p>
            <div className="mt-2 flex flex-wrap gap-3 text-sm text-muted-foreground">
              <span>{formatDate(hint.createdAt)}</span>
              {hint.sourceNote ? <span>{hint.sourceNote}</span> : null}
              {hint.sourceUrl ? (
                <a
                  href={hint.sourceUrl}
                  target="_blank"
                  rel="nofollow noopener noreferrer"
                  className="underline"
                >
                  Quelle öffnen
                </a>
              ) : null}
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}
