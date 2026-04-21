/**
 * Avatar primitive — Radix Avatar in a daisyUI `avatar` wrapper. Renders the
 * image when `src` loads, falls back to `fallback` (typically initials).
 * Extra props pass through to the Radix Root so callers can set
 * `id`/`aria-*`/`title` etc.
 */
import { Avatar as RxAvatar } from "radix-ui";
import type { ComponentProps, ReactNode } from "react";
import { cx } from "./cx.ts";

export type AvatarSize = "xs" | "sm" | "md" | "lg";

const SIZE_CLASSES: Record<AvatarSize, string> = {
  xs: "w-6 text-[0.625rem]",
  sm: "w-8 text-xs",
  md: "w-10 text-sm",
  lg: "w-12 text-base",
};

export interface AvatarProps extends ComponentProps<typeof RxAvatar.Root> {
  src?: string;
  alt?: string;
  fallback: ReactNode;
  size?: AvatarSize;
}

export function Avatar({ src, alt, fallback, size = "md", className, ...rest }: AvatarProps) {
  return (
    <RxAvatar.Root className={cx("avatar", className)} {...rest}>
      <div className={cx("rounded-full bg-base-200 text-base-content/80", SIZE_CLASSES[size])}>
        {src && <RxAvatar.Image src={src} alt={alt} className="rounded-full object-cover" />}
        <RxAvatar.Fallback
          delayMs={src ? 200 : 0}
          className="flex h-full w-full items-center justify-center font-medium"
        >
          {fallback}
        </RxAvatar.Fallback>
      </div>
    </RxAvatar.Root>
  );
}
