import {
  type DecoratorContext,
  type Operation,
  type Program,
  setTypeSpecNamespace,
} from "@typespec/compiler";
import * as v from "valibot";

// State key for storing @mcpTool metadata
const mcpToolKey = Symbol.for("@tend/typespec-mcp-go::mcpTool");

export interface McpToolOptions {
  name: string;
  description: string;
}

const McpToolOptionsSchema = v.object({
  name: v.string(),
  description: v.string(),
});

/** Extract a raw JS string from a TypeSpec decorator argument (may be a string or a value object). */
const StringArgSchema = v.union([
  v.string(),
  v.pipe(
    v.object({ value: v.string() }),
    v.transform((o) => o.value),
  ),
]);

export function $mcpTool(
  context: DecoratorContext,
  target: Operation,
  name: string,
  description: string,
): void {
  const nameStr = v.parse(StringArgSchema, name);
  const descStr = v.parse(StringArgSchema, description);
  context.program.stateMap(mcpToolKey).set(target, { name: nameStr, description: descStr });
}

export function getMcpTool(program: Program, op: Operation): McpToolOptions | undefined {
  const value: unknown = program.stateMap(mcpToolKey).get(op);
  if (value === undefined) return undefined;
  return v.parse(McpToolOptionsSchema, value);
}

setTypeSpecNamespace("MCP", $mcpTool);
