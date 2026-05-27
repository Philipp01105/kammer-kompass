"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { AdminSidebar } from "@/components/admin/AdminSidebar";
import { AdminTopbar } from "@/components/admin/AdminTopbar";
import { useAdminMe } from "@/lib/abilities";

export function AdminShell({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const me = useAdminMe();

  useEffect(() => {
    if (me.isError) {
      router.replace("/login");
    }
  }, [me.isError, router]);

  if (me.isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted">
        <div className="rounded-lg border bg-background p-7 text-base text-muted-foreground">
          Admin-Session wird geprüft...
        </div>
      </div>
    );
  }

  if (me.isError) {
    return null;
  }

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[280px_1fr]">
      <AdminSidebar />
      <div className="min-w-0">
        <AdminTopbar />
        <main className="p-5 lg:p-8">{children}</main>
      </div>
    </div>
  );
}
