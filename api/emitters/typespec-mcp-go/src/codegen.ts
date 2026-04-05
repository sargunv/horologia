import {
  type Program,
  type Operation,
  type Model,
  type ModelProperty,
  type Scalar,
  type Type,
  getDoc,
} from "@typespec/compiler";
import type { HttpOperation } from "@typespec/http";
import type { McpToolOptions } from "./decorator.js";

export interface ToolInfo {
  op: Operation;
  httpOp: HttpOperation;
  toolOpts: McpToolOptions;
}

/**
 * Maps a TypeSpec scalar/model type to a mcp-go WithXxx call fragment.
 */
function mcpPropertyType(type: Type): string {
  if (type.kind === "Scalar") {
    const name = (type as Scalar).name;
    if (name === "string" || name === "url" || name === "plainDate" || name === "utcDateTime") {
      return "mcp.WithString";
    }
    if (
      name === "int32" ||
      name === "int64" ||
      name === "float32" ||
      name === "float64" ||
      name === "integer" ||
      name === "numeric"
    ) {
      return "mcp.WithNumber";
    }
    if (name === "boolean") {
      return "mcp.WithBoolean";
    }
  }
  // Union types (e.g. string | null) — treat as string
  if (type.kind === "Union") {
    return "mcp.WithString";
  }
  // Enums — string
  if (type.kind === "Enum") {
    return "mcp.WithString";
  }
  // Arrays
  if (type.kind === "Model" && (type as Model).indexer !== undefined) {
    return "mcp.WithArray";
  }
  // Default to string for complex models
  return "mcp.WithString";
}

/**
 * Resolves property type through optional/nullable union wrappers.
 */
function resolveType(type: Type): Type {
  if (type.kind === "Union") {
    const variants = [...(type as any).variants.values()].map((v: any) => v.type as Type);
    const nonNull = variants.find(
      (v: Type) => !(v.kind === "Intrinsic" && (v as any).name === "null"),
    );
    return nonNull ?? type;
  }
  return type;
}

interface GoParam {
  name: string;
  goField: string;
  mcpType: string;
  required: boolean;
  description: string;
  isPathParam: boolean;
}

/**
 * Collects path/query parameters for a tool from its HTTP operation.
 */
function collectParams(program: Program, httpOp: HttpOperation): GoParam[] {
  const params: GoParam[] = [];

  for (const param of httpOp.parameters.parameters) {
    const p = param.param;
    const resolvedType = resolveType(p.type);
    const mcpType = mcpPropertyType(resolvedType);
    const doc = getDoc(program, p) ?? "";
    const isPath = param.type === "path";

    params.push({
      name: p.name,
      goField: capitalize(p.name),
      mcpType,
      required: isPath || !p.optional,
      description: doc,
      isPathParam: isPath,
    });
  }

  return params;
}

/**
 * Collects body fields from the HTTP operation's body model.
 */
function collectBodyFields(program: Program, httpOp: HttpOperation): GoParam[] {
  const body = httpOp.parameters.body;
  if (!body) return [];

  const bodyType = body.type;
  if (bodyType.kind !== "Model") return [];

  const fields: GoParam[] = [];
  for (const [name, prop] of (bodyType as Model).properties) {
    const resolvedType = resolveType(prop.type);
    const mcpType = mcpPropertyType(resolvedType);
    const doc = getDoc(program, prop) ?? "";

    fields.push({
      name,
      goField: capitalize(name),
      mcpType,
      required: !prop.optional,
      description: doc,
      isPathParam: false,
    });
  }
  return fields;
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function escapeDQ(s: string): string {
  return s.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

function snakeToCamel(s: string): string {
  return s
    .split("_")
    .map((part, i) => (i === 0 ? part : part.charAt(0).toUpperCase() + part.slice(1)))
    .join("");
}

/** Produces , mcp.Required() if required */
function requiredOpt(required: boolean): string {
  return required ? ", mcp.Required()" : "";
}

/** Produces , mcp.Description("...") if non-empty */
function descOpt(desc: string): string {
  if (!desc) return "";
  return `, mcp.Description("${escapeDQ(desc)}")`;
}

/**
 * Derives the ogen Handler interface method signature.
 */
function deriveHandlerMethod(opName: string, httpOp: HttpOperation): string {
  const iface = httpOp.container ? ((httpOp.container as any).name ?? "") : "";
  // ogen capitalizes the operation name
  const methodName = `${iface}${capitalize(opName)}`;

  switch (methodName) {
    case "SpaceTasksCreate":
      return "SpaceTasksCreate(ctx context.Context, req *apigen.TaskCreate, params apigen.SpaceTasksCreateParams) (*apigen.Task, error)";
    case "SpaceTasksList":
      return "SpaceTasksList(ctx context.Context, params apigen.SpaceTasksListParams) (*apigen.TaskPage, error)";
    case "SpaceTasksRead":
      return "SpaceTasksRead(ctx context.Context, params apigen.SpaceTasksReadParams) (*apigen.Task, error)";
    case "SpaceTasksUpdate":
      return "SpaceTasksUpdate(ctx context.Context, req *apigen.TaskUpdate, params apigen.SpaceTasksUpdateParams) (*apigen.Task, error)";
    default:
      return `${methodName}(ctx context.Context, params any) (any, error)`;
  }
}

const COMPLEX_FIELDS = new Set([
  "due",
  "overdueActionRule",
  "recurrenceType",
  "recurrenceRule",
  "assigneeIds",
  "rotationPool",
  "tags",
]);

/**
 * Emits the Go call logic for a particular tool into `lines`.
 */
function emitGoCall(lines: string[], httpOp: HttpOperation, bodyFields: GoParam[]): void {
  const opName = httpOp.operation.name;

  if (opName === "list") {
    lines.push(`\t\tparams := apigen.SpaceTasksListParams{SpaceSlug: spaceSlug}`);
    lines.push(`\t\tif v, ok := args["cursor"].(string); ok && v != "" {`);
    lines.push(`\t\t\tparams.Cursor.SetTo(v)`);
    lines.push(`\t\t}`);
    lines.push(`\t\tif v, ok := args["limit"].(float64); ok {`);
    lines.push(`\t\t\tparams.Limit.SetTo(int32(v))`);
    lines.push(`\t\t}`);
    lines.push(`\t\tpage, err := h.SpaceTasksList(ctx, params)`);
    lines.push(`\t\tif err != nil {`);
    lines.push(`\t\t\treturn toolResultFromError(err), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\treturn mustToolResultJSON(page), nil`);
  } else if (opName === "create") {
    lines.push(`\t\ttitle, ok2 := args["title"].(string)`);
    lines.push(`\t\tif !ok2 || title == "" {`);
    lines.push(`\t\t\treturn mcp.NewToolResultError("title is required"), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\tbody := &apigen.TaskCreate{Title: title}`);
    for (const f of bodyFields) {
      if (f.name === "title" || f.mcpType === "mcp.WithArray" || COMPLEX_FIELDS.has(f.name))
        continue;
      lines.push(`\t\tif v, ok := args["${f.name}"].(string); ok {`);
      lines.push(`\t\t\tbody.${f.goField}.SetTo(v)`);
      lines.push(`\t\t}`);
    }
    lines.push(
      `\t\ttask, err := h.SpaceTasksCreate(ctx, body, apigen.SpaceTasksCreateParams{SpaceSlug: spaceSlug})`,
    );
    lines.push(`\t\tif err != nil {`);
    lines.push(`\t\t\treturn toolResultFromError(err), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\treturn mustToolResultJSON(task), nil`);
  } else if (opName === "read") {
    lines.push(`\t\ttaskId, ok2 := args["taskId"].(string)`);
    lines.push(`\t\tif !ok2 || taskId == "" {`);
    lines.push(`\t\t\treturn mcp.NewToolResultError("taskId is required"), nil`);
    lines.push(`\t\t}`);
    lines.push(
      `\t\ttask, err := h.SpaceTasksRead(ctx, apigen.SpaceTasksReadParams{SpaceSlug: spaceSlug, TaskId: taskId})`,
    );
    lines.push(`\t\tif err != nil {`);
    lines.push(`\t\t\treturn toolResultFromError(err), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\treturn mustToolResultJSON(task), nil`);
  } else if (opName === "update") {
    lines.push(`\t\ttaskId, ok2 := args["taskId"].(string)`);
    lines.push(`\t\tif !ok2 || taskId == "" {`);
    lines.push(`\t\t\treturn mcp.NewToolResultError("taskId is required"), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\tbody := &apigen.TaskUpdate{}`);
    for (const f of bodyFields) {
      if (f.mcpType === "mcp.WithArray" || COMPLEX_FIELDS.has(f.name)) continue;
      lines.push(`\t\tif v, ok := args["${f.name}"].(string); ok {`);
      lines.push(`\t\t\tbody.${f.goField}.SetTo(v)`);
      lines.push(`\t\t}`);
    }
    lines.push(
      `\t\ttask, err := h.SpaceTasksUpdate(ctx, body, apigen.SpaceTasksUpdateParams{SpaceSlug: spaceSlug, TaskId: taskId})`,
    );
    lines.push(`\t\tif err != nil {`);
    lines.push(`\t\t\treturn toolResultFromError(err), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\treturn mustToolResultJSON(task), nil`);
  }
}

/**
 * Generates the RegisterTools Go source file content.
 */
export function generateGoFile(program: Program, tools: ToolInfo[]): string {
  const lines: string[] = [];

  lines.push("// Code generated by typespec-mcp-go. DO NOT EDIT.");
  lines.push("// Source: api/src/tasks.tsp");
  lines.push("// Regenerate: cd api && mise run generate");
  lines.push("");
  lines.push("package mcpgen");
  lines.push("");
  lines.push("import (");
  lines.push('\t"context"');
  lines.push('\t"fmt"');
  lines.push("");
  lines.push('\t"github.com/mark3labs/mcp-go/mcp"');
  lines.push('\tmcpserver "github.com/mark3labs/mcp-go/server"');
  lines.push("");
  lines.push('\tapigen "github.com/sargunv/tend/server/internal/api/gen"');
  lines.push(")");
  lines.push("");

  // Handlers interface
  lines.push("// Handlers is the interface MCP tool calls are dispatched through.");
  lines.push("// *api.Handler satisfies this interface.");
  lines.push("type Handlers interface {");
  for (const { httpOp } of tools) {
    const opName = httpOp.operation.name;
    lines.push(`\t${deriveHandlerMethod(opName, httpOp)}`);
  }
  lines.push("}");
  lines.push("");

  // RegisterTools
  lines.push("// RegisterTools registers all @mcpTool-annotated operations with the MCP server.");
  lines.push("func RegisterTools(s *mcpserver.MCPServer, h Handlers) {");
  for (const { toolOpts } of tools) {
    const camel = snakeToCamel(toolOpts.name);
    lines.push(`\ts.AddTool(${camel}Tool(), ${camel}Handler(h))`);
  }
  lines.push("}");
  lines.push("");

  // Per-tool definitions
  for (const { toolOpts, httpOp } of tools) {
    const camel = snakeToCamel(toolOpts.name);
    const pathParams = collectParams(program, httpOp).filter((p) => p.isPathParam);
    const queryParams = collectParams(program, httpOp).filter((p) => !p.isPathParam);
    const bodyFields = collectBodyFields(program, httpOp);

    lines.push(`// --- ${toolOpts.name} ---`);
    lines.push("");
    lines.push(`func ${camel}Tool() mcp.Tool {`);
    lines.push(`\treturn mcp.NewTool("${toolOpts.name}",`);
    lines.push(`\t\tmcp.WithDescription("${escapeDQ(toolOpts.description)}"),`);

    for (const p of pathParams) {
      lines.push(
        `\t\t${p.mcpType}("${p.name}"${requiredOpt(p.required)}${descOpt(p.description)}),`,
      );
    }
    for (const p of queryParams) {
      lines.push(
        `\t\t${p.mcpType}("${p.name}"${requiredOpt(p.required)}${descOpt(p.description)}),`,
      );
    }
    for (const f of bodyFields) {
      if (f.mcpType === "mcp.WithArray" || COMPLEX_FIELDS.has(f.name)) continue;
      lines.push(
        `\t\t${f.mcpType}("${f.name}"${requiredOpt(f.required)}${descOpt(f.description)}),`,
      );
    }

    lines.push("\t)");
    lines.push("}");
    lines.push("");

    lines.push(`func ${camel}Handler(h Handlers) mcpserver.ToolHandlerFunc {`);
    lines.push(
      `\treturn func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {`,
    );
    lines.push(`\t\targs := req.GetArguments()`);

    // Required path params (spaceSlug always first)
    for (const p of pathParams) {
      if (p.name === "taskId") continue; // handled per-op below
      lines.push(`\t\tspaceSlug, ok := args["${p.name}"].(string)`);
      lines.push(`\t\tif !ok || spaceSlug == "" {`);
      lines.push(`\t\t\treturn mcp.NewToolResultError("${p.name} is required"), nil`);
      lines.push(`\t\t}`);
    }

    emitGoCall(lines, httpOp, bodyFields);

    lines.push(`\t}`);
    lines.push(`}`);
    lines.push("");
  }

  // Helpers
  lines.push("// --- helpers ---");
  lines.push("");
  lines.push("func toolResultFromError(err error) *mcp.CallToolResult {");
  lines.push("\treturn mcp.NewToolResultError(err.Error())");
  lines.push("}");
  lines.push("");
  lines.push("func mustToolResultJSON(v any) *mcp.CallToolResult {");
  lines.push("\tres, err := mcp.NewToolResultJSON(v)");
  lines.push("\tif err != nil {");
  lines.push('\t\treturn mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err))');
  lines.push("\t}");
  lines.push("\treturn res");
  lines.push("}");
  lines.push("");

  return lines.join("\n");
}
