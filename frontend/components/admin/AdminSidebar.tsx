"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  ClipboardList,
  History,
  Home,
  Landmark,
  ScrollText,
  Shield,
  ShieldCheck,
  Users,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAbilities } from "@/lib/abilities";
import type { AdminAbility } from "@/types/api";

const links: Array<{
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  ability?: AdminAbility;
}> = [
  { href: "/admin/dashboard", label: "Dashboard", icon: Home },
  {
    href: "/admin/info-suggestions",
    label: "Info-Vorschläge",
    icon: ClipboardList,
    ability: "canReviewInfoSuggestions",
  },
  { href: "/admin/ihks", label: "IHKs", icon: Landmark, ability: "canPublishIHKInfo" },
  { href: "/admin/moderation", label: "Moderation", icon: Shield, ability: "canManageModerationTerms" },
  {
    href: "/admin/permission-requests",
    label: "Rechteanfragen",
    icon: ShieldCheck,
    ability: "canManagePermissionRequests",
  },
  { href: "/admin/users", label: "Users", icon: Users, ability: "canManageUsers" },
  { href: "/admin/audit", label: "Audit Logs", icon: ScrollText, ability: "canReadAuditLogs" },
  { href: "/admin/ihks", label: "Versionen", icon: History, ability: "canPublishIHKInfo" },
];

export function AdminSidebar() {
  const pathname = usePathname();
  const { abilities } = useAbilities();

  return (
    <aside className="border-r bg-card">
      <div className="sticky top-0 p-5">
        <Link href="/admin/dashboard" className="mb-6 block text-xl font-semibold">
          KammerKompass Admin
        </Link>
        <nav className="space-y-1">
          {links
            .filter((link) => !link.ability || abilities[link.ability])
            .map((link) => {
              const Icon = link.icon;
              const active = pathname === link.href || pathname.startsWith(`${link.href}/`);
              return (
                <Link
                  key={link.href + link.label}
                  href={link.href}
                  className={cn(
                    "flex items-center gap-3 rounded-md px-3.5 py-2.5 text-base hover:bg-muted",
                    active && "bg-muted font-medium"
                  )}
                >
                  <Icon className="h-4 w-4" />
                  {link.label}
                </Link>
              );
            })}
        </nav>
      </div>
    </aside>
  );
}
