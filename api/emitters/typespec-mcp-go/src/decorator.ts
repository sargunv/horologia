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

export function $mcpTool(
  context: DecoratorContext,
  target: Operation,
  name: string,
  description: string,
): void {
  // TypeSpec passes string literal arguments as value objects; extract raw JS strings.
  const nameStr: string = typeof name === "string" ? name : (name as any).value;
  const descStr: string =
    typeof description === "string" ? description : (description as any).value;
  context.program.stateMap(mcpToolKey).set(target, { name: nameStr, description: descStr });
}

export function getMcpTool(program: Program, op: Operation): McpToolOptions | undefined {
  return program.stateMap(mcpToolKey).get(op);
}

setTypeSpecNamespace("MCP", $mcpTool);
