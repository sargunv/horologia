import {
  type DecoratorContext,
  type Operation,
  type Program,
  setTypeSpecNamespace,
} from "@typespec/compiler";

// State key for storing @mcpTool metadata
const mcpToolKey = Symbol.for("@tend/typespec-mcp-go::mcpTool");

export interface McpToolOptions {
  name: string;
  description: string;
}

/** Extract a raw JS string from a TypeSpec decorator argument (may be a string or a value object). */
function extractString(value: unknown): string {
  if (typeof value === "string") return value;
  if (
    value !== null &&
    typeof value === "object" &&
    "value" in value &&
    typeof value.value === "string"
  ) {
    return value.value;
  }
  throw new Error(`Expected string or {value: string}, got ${typeof value}`);
}

export function $mcpTool(
  context: DecoratorContext,
  target: Operation,
  name: string,
  description: string,
): void {
  const nameStr = extractString(name);
  const descStr = extractString(description);
  context.program.stateMap(mcpToolKey).set(target, { name: nameStr, description: descStr });
}

export function getMcpTool(program: Program, op: Operation): McpToolOptions | undefined {
  return program.stateMap(mcpToolKey).get(op);
}

setTypeSpecNamespace("MCP", $mcpTool);
