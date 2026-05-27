import { cn } from "@/lib/utils";

type FieldProps = {
  label: string;
  info?: string;
  error?: string;
  children: React.ReactNode;
  className?: string;
};

export function Field({ label, info, error, children, className }: FieldProps) {
  return (
    <label className={cn("block space-y-2", className)}>
      <span className="flex items-center gap-2 text-base font-medium">
        {label}
        {info ? (
          <span
            tabIndex={0}
            title={info}
            aria-label={info}
            className="inline-flex h-5 w-5 items-center justify-center rounded-full border bg-muted text-xs font-semibold text-muted-foreground"
          >
            i
          </span>
        ) : null}
      </span>
      {children}
      {info ? <span className="block text-sm text-muted-foreground">{info}</span> : null}
      {error ? <span className="block text-base text-destructive">{error}</span> : null}
    </label>
  );
}
