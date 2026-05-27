export function EmptyState({ title, text }: { title: string; text?: string }) {
  return (
    <div className="rounded-lg border border-dashed p-8 text-center">
      <h3 className="font-semibold">{title}</h3>
      {text ? <p className="mt-2 text-sm text-muted-foreground">{text}</p> : null}
    </div>
  );
}
