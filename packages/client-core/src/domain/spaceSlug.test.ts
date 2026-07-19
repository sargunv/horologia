import { describe, expect, it } from "vitest";

import { slugifySpaceName } from "./spaceSlug.ts";

describe("slugifySpaceName", () => {
  it("keeps unicode letters while normalizing separators", () => {
    expect(slugifySpaceName("  Café & Garden Plans  ")).toBe("café-garden-plans");
  });
});
