import ReactMarkdown from "react-markdown";
import rehypeSanitize from "rehype-sanitize";

export function MarkdownViewer({ content }: { content: string }) {
  if (!content.trim()) {
    return <p className="text-sm text-muted-foreground">Noch kein geprüfter Infotext vorhanden.</p>;
  }

  return (
    <div className="prose-lite text-base">
      <ReactMarkdown rehypePlugins={[rehypeSanitize]}>{content}</ReactMarkdown>
    </div>
  );
}
