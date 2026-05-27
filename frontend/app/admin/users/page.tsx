"use client";

import { useQuery } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { adminApi } from "@/lib/api";
import { SearchBar } from "@/components/SearchBar";
import { CreateUserForm } from "@/components/admin/CreateUserForm";
import { UserRoleEditor } from "@/components/admin/UserRoleEditor";

export default function UsersPage() {
  const [query, setQuery] = useState("");
  const handleQueryChange = useCallback((value: string) => setQuery(value), []);
  const users = useQuery({
    queryKey: ["admin-users", query],
    queryFn: () => adminApi.listUsers(query),
  });

  return (
    <div className="space-y-4">
      <h2 className="text-2xl font-semibold">Users</h2>
      <CreateUserForm />
      <SearchBar value={query} onChange={handleQueryChange} />
      <div className="space-y-4">
        {users.data?.items.map((user) => (
          <UserRoleEditor key={user.id} user={user} />
        ))}
      </div>
      {users.error ? <p className="text-sm text-destructive">{users.error.message}</p> : null}
    </div>
  );
}
