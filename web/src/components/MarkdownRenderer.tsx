import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";
import rehypeExternalLinks from "rehype-external-links";
import rehypeHighlight from "rehype-highlight";
import rehypeKatex from "rehype-katex";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import remarkSupersub from "remark-supersub";

const components: Components = {
  h1: ({ children }) => <h1 className="h3">{children}</h1>,
  h2: ({ children }) => <h2 className="h4">{children}</h2>,
  h3: ({ children }) => <h3 className="h5">{children}</h3>,
  h4: ({ children }) => <h4 className="h6">{children}</h4>,
  h5: ({ children }) => <h5 className="h6">{children}</h5>,
  h6: ({ children }) => <h6 className="text-sm font-bold">{children}</h6>,
  a: ({ children, href, target, rel }) => (
    <a className="anchor" href={href} target={target} rel={rel}>
      {children}
    </a>
  ),
  blockquote: ({ children }) => <blockquote className="blockquote">{children}</blockquote>,
  pre: ({ children }) => <pre className="pre">{children}</pre>,
  code: ({ children, className }) => <code className={className ?? "code"}>{children}</code>,
  hr: () => <hr className="hr" />,
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
    <th className="border border-surface-300-700 px-3 py-1.5 font-bold">{children}</th>
  ),
  td: ({ children }) => <td className="border border-surface-300-700 px-3 py-1.5">{children}</td>,
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
        remarkPlugins={[[remarkGfm, { singleTilde: false }], remarkMath, remarkSupersub]}
        rehypePlugins={[
          [rehypeSanitize, defaultSchema],
          rehypeHighlight,
          rehypeKatex,
          [rehypeExternalLinks, { target: "_blank", rel: ["noopener", "noreferrer"] }],
        ]}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
