"use client";

import { Search } from "lucide-react";
import { useEffect, useState } from "react";
import { Input } from "@/components/ui/input";

type SearchBarProps = {
  value: string;
  onChange: (value: string) => void;
};

export function SearchBar({ value, onChange }: SearchBarProps) {
  const [localValue, setLocalValue] = useState(value);

  useEffect(() => {
    const timer = window.setTimeout(() => onChange(localValue), 300);
    return () => window.clearTimeout(timer);
  }, [localValue, onChange]);

  return (
    <label className="relative block">
      <span className="sr-only">IHK, Stadt oder Bundesland suchen</span>
      <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        value={localValue}
        onChange={(event) => setLocalValue(event.target.value)}
        placeholder="IHK, Stadt oder Bundesland suchen..."
        className="pl-9"
      />
    </label>
  );
}
