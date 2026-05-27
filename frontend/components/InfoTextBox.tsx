import { MarkdownViewer } from "@/components/MarkdownViewer";
import { confidenceLabel, formatDate } from "@/lib/markdown";
import type { PublicIHKItem } from "@/types/api";

export function InfoTextBox({ item }: { item: PublicIHKItem }) {
  return (
      <div style={{paddingTop: "1rem"}}>
          <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-base text-muted-foreground">
              <h4 className="font-semibold text-foreground">Geprüfter Infotext</h4>
              <span>Stand: {formatDate(item.info.updatedAt)}</span>
              <span>Vertrauen: {confidenceLabel(item.info.confidenceLevel)}</span>
              {item.info.sourceSummary ? <span>Quelle: {item.info.sourceSummary}</span> : null}
          </div>
          <section className="mt-5 rounded-lg border p-5">
              <MarkdownViewer content={item.info.currentText}/>
          </section>
      </div>
  );
}
