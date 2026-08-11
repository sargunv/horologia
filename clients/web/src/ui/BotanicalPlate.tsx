/**
 * BotanicalPlate — the One Ornament.
 * An engraved herbarium specimen used only where emptiness or arrival
 * needs a single authenticating mark (login title-page, empty states).
 */
import { cx } from "./cx.ts";

export function BotanicalPlate({
  className,
  caption = "Rosmarinus officinalis",
}: {
  className?: string;
  caption?: string;
}) {
  return (
    <figure
      className={cx("botanical-plate mx-auto flex max-w-56 flex-col items-center gap-3", className)}
    >
      <svg
        viewBox="0 0 160 200"
        className="botanical-plate-drawing h-auto w-full text-primary"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        {/* Main stem */}
        <path
          d="M80 188 C78 150 76 110 80 72 C82 48 86 28 92 12"
          stroke="currentColor"
          strokeWidth="1.15"
          strokeLinecap="round"
        />
        {/* Left branch sprays */}
        <path
          d="M79 160 C62 152 48 148 34 150 M79 140 C60 130 46 122 32 118 M78 118 C58 112 44 102 30 96 M78 96 C60 88 48 78 38 68 M80 76 C66 68 56 58 48 46"
          stroke="currentColor"
          strokeWidth="1"
          strokeLinecap="round"
        />
        {/* Right branch sprays */}
        <path
          d="M81 168 C98 162 114 160 128 164 M81 148 C100 140 116 136 132 138 M80 126 C100 122 118 116 134 114 M80 104 C98 98 114 90 128 84 M82 82 C96 74 110 66 122 56"
          stroke="currentColor"
          strokeWidth="1"
          strokeLinecap="round"
        />
        {/* Needle leaves — left */}
        <path
          d="M52 150 L40 142 M58 144 L44 134 M64 136 L50 126 M46 120 L34 112 M54 112 L40 102 M60 104 L48 94 M42 92 L32 84 M50 84 L38 74 M56 72 L46 62 M68 68 L58 56"
          stroke="currentColor"
          strokeWidth="0.85"
          strokeLinecap="round"
        />
        {/* Needle leaves — right */}
        <path
          d="M108 162 L120 156 M112 150 L126 144 M116 140 L130 136 M104 128 L118 122 M110 118 L124 110 M116 108 L130 102 M102 100 L116 92 M108 90 L122 82 M114 78 L126 70 M96 70 L108 60"
          stroke="currentColor"
          strokeWidth="0.85"
          strokeLinecap="round"
        />
        {/* Small apical buds */}
        <path
          d="M92 12 C94 8 97 6 100 8 M88 18 C86 14 84 12 82 14"
          stroke="currentColor"
          strokeWidth="0.9"
          strokeLinecap="round"
        />
        {/* Ground line / mount rule */}
        <path d="M36 188 H124" stroke="currentColor" strokeWidth="0.7" opacity="0.45" />
      </svg>
      <figcaption className="catalog-label text-center text-3xs font-semibold uppercase tracking-caps text-base-content/55">
        {caption}
      </figcaption>
    </figure>
  );
}
