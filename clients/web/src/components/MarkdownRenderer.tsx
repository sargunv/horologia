import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";
import rehypeExternalLinks from "rehype-external-links";
import rehypeHighlight from "rehype-highlight";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import remarkGfm from "remark-gfm";
import remarkSupersub from "remark-supersub";

const components: Components = {
  h1: ({ children }) => <h1 className="text-xl font-semibold">{children}</h1>,
  h2: ({ children }) => <h2 className="text-lg font-semibold">{children}</h2>,
  h3: ({ children }) => <h3 className="text-base font-semibold">{children}</h3>,
  h4: ({ children }) => <h4 className="text-sm font-bold">{children}</h4>,
  h5: ({ children }) => <h5 className="text-sm font-bold">{children}</h5>,
  h6: ({ children }) => <h6 className="text-sm font-bold">{children}</h6>,
  a: ({ children, href, target, rel }) => (
    <a className="text-primary hover:underline" href={href} target={target} rel={rel}>
      {children}
    </a>
  ),
  blockquote: ({ children }) => (
    <blockquote className="border-l-4 border-base-300 pl-4 italic text-base-content/80">
      {children}
    </blockquote>
  ),
  pre: ({ children }) => (
    <pre className="bg-base-200 rounded-box p-3 overflow-x-auto text-sm">{children}</pre>
  ),
  code: ({ children, className }) => (
    <code className={className ?? "bg-base-200 rounded px-1 py-0.5 text-sm"}>{children}</code>
  ),
  hr: () => <hr className="border-base-300" />,
  ul: ({ children, className }) => (
    <ul
      className={
        className === "contains-task-list"
          ? "space-y-2 pl-1 [&>li]:flex [&>li]:items-center [&>li]:gap-2"
          : "list-disc pl-6 space-y-1"
      }
    >
      {children}
    </ul>
  ),
  ol: ({ children }) => <ol className="list-decimal pl-6 space-y-1">{children}</ol>,
  input: ({ checked, type }) => {
    if (type === "checkbox") {
      return (
        <input
          className="checkbox pointer-events-none"
          type="checkbox"
          checked={checked}
          readOnly
        />
      );
    }
    return <input type={type} />;
  },
  del: ({ children }) => <del className="line-through">{children}</del>,
  table: ({ children }) => (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-left">{children}</table>
    </div>
  ),
  th: ({ children }) => (
    <th className="border border-base-300 px-3 py-1.5 font-bold">{children}</th>
  ),
  td: ({ children }) => <td className="border border-base-300 px-3 py-1.5">{children}</td>,
};

interface MarkdownRendererProps {
  children: string;
  className?: string;
}

export function MarkdownRenderer({ children, className }: MarkdownRendererProps) {
  return (
    <div className={["space-y-4 text-sm", className].filter(Boolean).join(" ")}>
      <ReactMarkdown
        components={components}
        remarkPlugins={[[remarkGfm, { singleTilde: false }], remarkSupersub]}
        rehypePlugins={[
          [rehypeSanitize, defaultSchema],
          rehypeHighlight,
          [rehypeExternalLinks, { target: "_blank", rel: ["noopener", "noreferrer"] }],
        ]}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
