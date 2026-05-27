"use client";

import { useAdminMe } from "@/lib/abilities";

export function AdminTopbar() {
  const me = useAdminMe();

  return (
    <header className="border-b bg-background px-8 py-5">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-base text-muted-foreground">Internes Admin-Dashboard</p>
          <h1 className="text-2xl font-semibold">IHK-Info-Plattform</h1>
        </div>
        {me.data ? (
          <p className="text-base text-muted-foreground">
            {me.data.user.displayName} · {me.data.user.email}
          </p>
        ) : null}
      </div>
    </header>
  );
}
