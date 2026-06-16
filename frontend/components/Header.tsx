"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Building2, LogIn } from "lucide-react";
import { adminApi } from "@/lib/api";

export function Header() {
  const me = useQuery({
    queryKey: ["auth-me"],
    queryFn: adminApi.authMe,
    retry: false,
  });
  const isLoggedIn = Boolean(me.data?.user);

  return (
    <header className="border-b bg-background/95">
      <div className="mx-auto flex max-w-7xl flex-col gap-5 px-5 py-7 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <Building2 className="h-5 w-5" aria-hidden="true" />
          </div>
          <div>
            <p className="text-3xl font-semibold">KammerKompass</p>
            <p className="text-base text-muted-foreground">
              Inoffizielle Community-Datenbank. Keine verbindliche Auskunft einer IHK.
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {isLoggedIn ? (
            <Link href="/admin/dashboard" className="inline-flex min-h-11 items-center justify-center rounded-md bg-secondary px-5 py-2.5 text-base font-medium text-secondary-foreground transition-colors hover:bg-secondary/80 focus:outline-none focus:ring-2 focus:ring-ring">
              Admin
            </Link>
          ) : (
            <Link
              href="/login"
              className="inline-flex min-h-11 items-center justify-center gap-2 rounded-md bg-secondary px-5 py-2.5 text-base font-medium text-secondary-foreground transition-colors hover:bg-secondary/80 focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <LogIn className="h-4 w-4" />
              Anmelden
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
