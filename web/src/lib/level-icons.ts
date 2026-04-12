/**
 * Curated set of Lucide icons available for effort and priority levels.
 * Keys are kebab-case icon names (matching the API/DB format).
 * Only icons relevant to effort/priority concepts are included to keep
 * the bundle small and the picker focused.
 */

import {
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  Award,
  Bell,
  Bolt,
  ChevronDown,
  ChevronUp,
  ChevronsUp,
  CircleHelp,
  Clock,
  Cloud,
  Crown,
  Diamond,
  Droplet,
  Feather,
  Flag,
  Flame,
  Gauge,
  Heart,
  Hexagon,
  Hourglass,
  Leaf,
  Minus,
  Mountain,
  Rocket,
  Shield,
  Signal,
  SignalHigh,
  SignalLow,
  SignalMedium,
  Sparkles,
  Star,
  Target,
  Timer,
  TrendingUp,
  Trophy,
  Weight,
  Wind,
  Zap,
  type LucideIcon,
} from "lucide-react";

/** All available level icons, keyed by their kebab-case API name. */
export const LEVEL_ICONS: Record<string, LucideIcon> = {
  // Intensity / scale
  feather: Feather,
  leaf: Leaf,
  droplet: Droplet,
  wind: Wind,
  cloud: Cloud,
  mountain: Mountain,
  flame: Flame,
  zap: Zap,
  bolt: Bolt,
  rocket: Rocket,
  sparkles: Sparkles,

  // Gauges / time
  gauge: Gauge,
  timer: Timer,
  clock: Clock,
  hourglass: Hourglass,
  weight: Weight,

  // Signal strength
  "signal-low": SignalLow,
  "signal-medium": SignalMedium,
  "signal-high": SignalHigh,
  signal: Signal,

  // Arrows / direction
  "arrow-down": ArrowDown,
  minus: Minus,
  "arrow-up": ArrowUp,
  "chevron-down": ChevronDown,
  "chevron-up": ChevronUp,
  "chevrons-up": ChevronsUp,
  "trending-up": TrendingUp,

  // Priority / alerts
  flag: Flag,
  "alert-triangle": AlertTriangle,
  bell: Bell,

  // Shapes / abstract
  diamond: Diamond,
  hexagon: Hexagon,
  star: Star,
  heart: Heart,

  // Achievement
  target: Target,
  shield: Shield,
  crown: Crown,
  trophy: Trophy,
  award: Award,
};

/** Ordered list of icon names for the picker UI. */
export const LEVEL_ICON_NAMES = Object.keys(LEVEL_ICONS);

/** Fallback icon shown when an icon name is unrecognized. */
export const FALLBACK_ICON: LucideIcon = CircleHelp;

/**
 * Resolve an icon name to its Lucide component.
 * Returns the fallback icon for null/undefined/unrecognized names.
 */
export function getLevelIcon(name: string | null | undefined): LucideIcon {
  if (!name) return FALLBACK_ICON;
  return LEVEL_ICONS[name] ?? FALLBACK_ICON;
}
