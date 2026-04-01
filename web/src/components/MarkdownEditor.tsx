import CodeBlockLowlight from "@tiptap/extension-code-block-lowlight";
import Link from "@tiptap/extension-link";
import Placeholder from "@tiptap/extension-placeholder";
import { Markdown } from "@tiptap/markdown";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { common, createLowlight } from "lowlight";
import { useCallback, useEffect, useRef } from "react";

const lowlight = createLowlight(common);

const DEBOUNCE_MS = 1500;

interface MarkdownEditorProps {
  value: string;
  onChange: (markdown: string) => void;
  onBlur?: () => void;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
}

export function MarkdownEditor({
  value,
  onChange,
  onBlur,
  disabled = false,
  placeholder = "Add a description...",
  className,
}: MarkdownEditorProps) {
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastSavedRef = useRef(value);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onBlurRef = useRef(onBlur);
  onBlurRef.current = onBlur;

  const clearDebounce = useCallback(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }
  }, []);

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ codeBlock: false }),
      CodeBlockLowlight.configure({ lowlight }),
      Link.configure({ openOnClick: false }),
      Placeholder.configure({ placeholder }),
      Markdown,
    ],
    content: value,
    contentType: "markdown",
    editable: !disabled,
    onUpdate: ({ editor }) => {
      clearDebounce();
      debounceRef.current = setTimeout(() => {
        const md = editor.getMarkdown();
        lastSavedRef.current = md;
        onChangeRef.current(md);
      }, DEBOUNCE_MS);
    },
    onBlur: ({ editor }) => {
      clearDebounce();
      const md = editor.getMarkdown();
      if (md !== lastSavedRef.current) {
        lastSavedRef.current = md;
        onChangeRef.current(md);
      }
      onBlurRef.current?.();
    },
  });

  // Sync disabled state
  useEffect(() => {
    if (editor) editor.setEditable(!disabled);
  }, [editor, disabled]);

  // Sync external value changes (e.g. from refetch) without disrupting cursor
  useEffect(() => {
    if (!editor) return;
    if (value !== lastSavedRef.current) {
      editor.commands.setContent(value, { emitUpdate: false, contentType: "markdown" });
      lastSavedRef.current = value;
    }
  }, [editor, value]);

  // Flush pending save and clean up on unmount
  useEffect(() => {
    return () => {
      clearDebounce();
      if (editor && !editor.isDestroyed) {
        const md = editor.getMarkdown();
        if (md !== lastSavedRef.current) {
          lastSavedRef.current = md;
          onChangeRef.current(md);
        }
      }
    };
  }, [clearDebounce, editor]);

  if (!editor) return null;

  return (
    <div className={className}>
      <div className="rounded-base border border-surface-200-800 transition-colors focus-within:border-primary-500">
        <EditorContent editor={editor} className="prose-editor min-h-[6rem] px-3 py-2 text-sm" />
      </div>
    </div>
  );
}
