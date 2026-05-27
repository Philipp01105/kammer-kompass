"use client";

import { useAbilities } from "@/lib/abilities";
import type { AdminAbility } from "@/types/api";

type PermissionGateProps = {
  ability: AdminAbility;
  children: React.ReactNode;
};

export function PermissionGate({ ability, children }: PermissionGateProps) {
  const { abilities } = useAbilities();
  if (!abilities[ability]) return null;
  return <>{children}</>;
}
