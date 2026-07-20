import { useWindowDimensions } from "react-native";

export type WindowClass = "compact" | "medium" | "expanded";

export const MEDIUM_WINDOW_MIN_WIDTH = 600;
export const EXPANDED_WINDOW_MIN_WIDTH = 840;

export interface ListDetailGeometry {
  windowClass: WindowClass;
  presentation: "single-pane" | "list-detail";
  listPaneWidth: number;
  detailPaneWidth: number;
}

export interface ListDetailGeometryOptions {
  minimumListPaneWidth?: number;
  maximumListPaneWidth?: number;
  preferredListFraction?: number;
}

export function classifyWindow(width: number): WindowClass {
  assertUsableWidth(width);
  if (width < MEDIUM_WINDOW_MIN_WIDTH) return "compact";
  if (width < EXPANDED_WINDOW_MIN_WIDTH) return "medium";
  return "expanded";
}

export function getListDetailGeometry(
  width: number,
  options: ListDetailGeometryOptions = {},
): ListDetailGeometry {
  const windowClass = classifyWindow(width);
  if (windowClass !== "expanded") {
    return {
      windowClass,
      presentation: "single-pane",
      listPaneWidth: width,
      detailPaneWidth: width,
    };
  }

  const minimumListPaneWidth = options.minimumListPaneWidth ?? 320;
  const maximumListPaneWidth = options.maximumListPaneWidth ?? 440;
  const preferredListFraction = options.preferredListFraction ?? 0.38;
  if (minimumListPaneWidth <= 0 || maximumListPaneWidth < minimumListPaneWidth) {
    throw new Error("List/detail pane widths must define a positive range");
  }
  if (preferredListFraction <= 0 || preferredListFraction >= 1) {
    throw new Error("List/detail preferred fraction must be between zero and one");
  }

  const listPaneWidth = Math.min(
    maximumListPaneWidth,
    Math.max(minimumListPaneWidth, width * preferredListFraction),
  );
  return {
    windowClass,
    presentation: "list-detail",
    listPaneWidth,
    detailPaneWidth: width - listPaneWidth,
  };
}

export function useWindowClass(): WindowClass {
  const { width } = useWindowDimensions();
  return classifyWindow(width);
}

export function useListDetailGeometry(options: ListDetailGeometryOptions = {}): ListDetailGeometry {
  const { width } = useWindowDimensions();
  return getListDetailGeometry(width, options);
}

function assertUsableWidth(width: number): void {
  if (!Number.isFinite(width) || width <= 0) {
    throw new Error("Window width must be a positive finite number");
  }
}
