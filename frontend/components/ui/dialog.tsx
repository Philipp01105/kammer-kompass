"use client";

import { X } from "lucide-react";
import { Button } from "@/components/ui/button";

type DialogProps = {
  title: string;
  description?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
};

export function Dialog({ title, description, open, onOpenChange, children }: DialogProps) {
  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="dialog-title"
    >
      <div className="max-h-[90vh] w-full max-w-3xl overflow-y-auto rounded-lg border bg-background p-7 shadow-xl">
        <div className="mb-5 flex items-start justify-between gap-4">
          <div>
            <h2 id="dialog-title" className="text-xl font-semibold">
              {title}
            </h2>
            {description ? (
              <p className="mt-1 text-sm text-muted-foreground">{description}</p>
            ) : null}
          </div>
          <Button
            variant="ghost"
            className="h-9 w-9 px-0"
            onClick={() => onOpenChange(false)}
            aria-label="Dialog schließen"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
        {children}
      </div>
    </div>
  );
}
