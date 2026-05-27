import { Textarea } from "@/components/ui/textarea";

type MarkdownEditorProps = {
  value: string;
  onChange: (value: string) => void;
  minRows?: number;
};

export function MarkdownEditor({ value, onChange, minRows = 14 }: MarkdownEditorProps) {
  return (
    <Textarea
      value={value}
      onChange={(event) => onChange(event.target.value)}
      rows={minRows}
      className="font-mono text-sm"
    />
  );
}
