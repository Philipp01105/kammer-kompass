import { ModerationTermsTable } from "@/components/admin/ModerationTermsTable";

export default function ModerationPage() {
  return (
    <div className="space-y-4">
      <h2 className="text-2xl font-semibold">Moderationsbegriffe</h2>
      <ModerationTermsTable />
    </div>
  );
}
