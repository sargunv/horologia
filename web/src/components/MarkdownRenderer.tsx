import ReactMarkdown from "react-markdown";
import rehypeHighlight from "rehype-highlight";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";

interface MarkdownRendererProps {
  children: string;
  className?: string;
}

export function MarkdownRenderer({ children, className }: MarkdownRendererProps) {
  return (
    <div
      className={[
        "prose prose-sm prose-stone dark:prose-invert max-w-none prose-pre:!bg-transparent prose-pre:!p-0",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <ReactMarkdown rehypePlugins={[[rehypeSanitize, defaultSchema], rehypeHighlight]}>
        {children}
      </ReactMarkdown>
    </div>
  );
}
