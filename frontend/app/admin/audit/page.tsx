"use client";

import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { formatDate } from "@/lib/markdown";

export default function AuditPage() {
  const logs = useQuery({
    queryKey: ["audit-logs"],
    queryFn: adminApi.listAuditLogs,
  });

  return (
    <div className="space-y-4">
      <h2 className="text-2xl font-semibold">Audit Logs</h2>
      <div className="overflow-x-auto rounded-lg border">
        <table className="w-full min-w-[920px] text-left text-sm">
          <thead className="border-b text-muted-foreground">
            <tr>
              <th className="p-3">Zeit</th>
              <th className="p-3">Actor</th>
              <th className="p-3">Action</th>
              <th className="p-3">Resource Type</th>
              <th className="p-3">Resource ID</th>
              <th className="p-3">Details</th>
            </tr>
          </thead>
          <tbody>
            {logs.data?.items.map((log) => (
              <tr key={log.id} className="border-b align-top">
                <td className="p-3">{formatDate(log.createdAt)}</td>
                <td className="p-3 font-mono text-xs">{log.actorUserId ?? "-"}</td>
                <td className="p-3">{log.action}</td>
                <td className="p-3">{log.resourceType}</td>
                <td className="p-3 font-mono text-xs">{log.resourceId ?? "-"}</td>
                <td className="p-3">
                  <details>
                    <summary className="cursor-pointer underline">Öffnen</summary>
                    <pre className="mt-2 max-w-md overflow-x-auto rounded-md bg-muted p-3 text-xs">
                      {JSON.stringify(
                        {
                          oldValue: log.oldValue,
                          newValue: log.newValue,
                          ipHash: log.ipHash,
                          userAgentHash: log.userAgentHash,
                        },
                        null,
                        2
                      )}
                    </pre>
                  </details>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {logs.error ? <p className="text-sm text-destructive">{logs.error.message}</p> : null}
    </div>
  );
}
