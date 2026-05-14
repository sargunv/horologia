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
  onLocalChange?: (markdown: string) => void;
  onBlur?: () => void;
  disabled?: boolean;
  syncExternalValue?: boolean;
  placeholder?: string;
  className?: string;
}

export function MarkdownEditor({
  value,
  onChange,
  onLocalChange,
  onBlur,
  disabled = false,
  syncExternalValue = true,
  placeholder = "Add a description...",
  className,
}: MarkdownEditorProps) {
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onLocalChangeRef = useRef(onLocalChange);
  onLocalChangeRef.current = onLocalChange;
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
      const md = editor.getMarkdown();
      onLocalChangeRef.current?.(md);
      clearDebounce();
      debounceRef.current = setTimeout(() => {
        onChangeRef.current(editor.getMarkdown());
      }, DEBOUNCE_MS);
    },
    onBlur: ({ editor }) => {
      clearDebounce();
      onChangeRef.current(editor.getMarkdown());
      onBlurRef.current?.();
    },
  });

  // Sync disabled state without emitting Tiptap update events. Disabled is hard read-only only.
  useEffect(() => {
    if (editor) editor.setEditable(!disabled, false);
  }, [editor, disabled]);

  // Sync external value changes (e.g. from refetch) only when the parent says local content is clean.
  useEffect(() => {
    if (!editor || !syncExternalValue) return;
    if (value !== editor.getMarkdown()) {
      editor.commands.setContent(value, { emitUpdate: false, contentType: "markdown" });
    }
  }, [editor, syncExternalValue, value]);

  // Flush pending save and clean up on unmount
  useEffect(() => {
    return () => {
      clearDebounce();
      if (editor && !editor.isDestroyed) {
        onChangeRef.current(editor.getMarkdown());
      }
    };
  }, [clearDebounce, editor]);

  if (!editor) return null;

  return (
    <div className={className}>
      <div className="rounded-box border border-base-300 transition-colors focus-within:border-primary">
        <EditorContent editor={editor} className="prose-editor min-h-[6rem] px-3 py-2 text-sm" />
      </div>
    </div>
  );
}
