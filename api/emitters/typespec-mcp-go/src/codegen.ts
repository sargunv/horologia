import {
  type Enum,
  type Model,
  type Program,
  type Scalar,
  type Type,
  getDoc,
} from "@typespec/compiler";
import type { HttpOperation, HttpOperationResponse } from "@typespec/http";
import type { McpToolOptions } from "./decorator.js";

export interface ToolInfo {
  op: import("@typespec/compiler").Operation;
  httpOp: HttpOperation;
  toolOpts: McpToolOptions;
}

export interface EmitterOptions {
  /** Go import path for the ogen-generated API package (e.g. "github.com/foo/bar/internal/api/gen"). */
  ogenImportPath: string;
}

// --- Type classification ---

/**
 * Returns the non-null variant of a nullable union, or null if the type is not
 * a simple nullable union (i.e. has multiple non-null variants or contains
 * complex types). Returns the type itself for non-union simple types.
 */
function getSimpleNonNull(type: Type): Type | null {
  if (type.kind === "Scalar" || type.kind === "Enum") return type;
  if (type.kind === "Union") {
    const variants = [...(type as any).variants.values()].map((v: any) => v.type as Type);
    const nonNull = variants.filter(
      (v: Type) => !(v.kind === "Intrinsic" && (v as any).name === "null"),
    );
    if (nonNull.length === 1 && (nonNull[0].kind === "Scalar" || nonNull[0].kind === "Enum")) {
      return nonNull[0];
    }
  }
  return null;
}

// --- MCP type mapping ---

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
  return "mcp.WithString";
}

// --- Go type mapping ---

interface GoTypeInfo {
  /** The Go type to assert from `any` (e.g. "string", "float64", "bool"). */
  assertType: string;
  /** Expression to convert the asserted value, using %s as placeholder. Empty if no conversion needed. */
  convertExpr: string;
}

function goTypeInfo(type: Type, alias: string): GoTypeInfo {
  if (type.kind === "Scalar") {
    const name = (type as Scalar).name;
    if (name === "string" || name === "url" || name === "plainDate" || name === "utcDateTime") {
      return { assertType: "string", convertExpr: "" };
    }
    if (name === "int32") return { assertType: "float64", convertExpr: "int32(%s)" };
    if (name === "int64") return { assertType: "float64", convertExpr: "int64(%s)" };
    if (name === "float32" || name === "float64") return { assertType: "float64", convertExpr: "" };
    if (name === "boolean") return { assertType: "bool", convertExpr: "" };
  }
  if (type.kind === "Enum") {
    return { assertType: "string", convertExpr: `${alias}.${(type as Enum).name}(%s)` };
  }
  return { assertType: "string", convertExpr: "" };
}

// --- Parameter collection ---

interface GoParam {
  name: string;
  goField: string;
  mcpType: string;
  required: boolean;
  description: string;
  isPathParam: boolean;
  typeInfo: GoTypeInfo;
}

function collectParams(program: Program, httpOp: HttpOperation, alias: string): GoParam[] {
  const params: GoParam[] = [];
  for (const param of httpOp.parameters.parameters) {
    const p = param.param;
    const resolved = getSimpleNonNull(p.type);
    if (!resolved) continue;
    params.push({
      name: p.name,
      goField: capitalize(p.name),
      mcpType: mcpPropertyType(resolved),
      required: param.type === "path" || !p.optional,
      description: getDoc(program, p) ?? "",
      isPathParam: param.type === "path",
      typeInfo: goTypeInfo(resolved, alias),
    });
  }
  return params;
}

function collectBodyFields(
  program: Program,
  httpOp: HttpOperation,
  alias: string,
  pathParamNames: Set<string>,
): GoParam[] {
  const body = httpOp.parameters.body;
  if (!body) return [];
  const bodyType = body.type;
  if (bodyType.kind !== "Model") return [];

  const fields: GoParam[] = [];
  const usedNames = new Set<string>(pathParamNames);
  for (const [name, prop] of (bodyType as Model).properties) {
    const resolved = getSimpleNonNull(prop.type);
    if (!resolved) continue;
    // If a body field collides with an existing name, prefix with "body" for the MCP param name
    let mcpName = name;
    if (usedNames.has(mcpName)) {
      mcpName = `body${capitalize(mcpName)}`;
      if (usedNames.has(mcpName)) {
        throw new Error(
          `MCP param name collision: "${mcpName}" already used (from body field "${name}")`,
        );
      }
    }
    usedNames.add(mcpName);
    fields.push({
      name: mcpName,
      goField: capitalize(name),
      mcpType: mcpPropertyType(resolved),
      required: !prop.optional,
      description: getDoc(program, prop) ?? "",
      isPathParam: false,
      typeInfo: goTypeInfo(resolved, alias),
    });
  }
  return fields;
}

// --- ogen name derivation ---

interface DerivedOp {
  methodName: string;
  paramsType: string;
  bodyTypeName: string | null;
  responseTypeName: string | null;
  hasParams: boolean;
}

function deriveOp(httpOp: HttpOperation): DerivedOp {
  const iface = httpOp.container ? ((httpOp.container as any).name ?? "") : "";
  const methodName = `${iface}${capitalize(httpOp.operation.name)}`;
  return {
    methodName,
    paramsType: `${methodName}Params`,
    bodyTypeName: deriveBodyTypeName(httpOp),
    responseTypeName: deriveResponseTypeName(httpOp),
    hasParams: httpOp.parameters.parameters.length > 0,
  };
}

function deriveBodyTypeName(httpOp: HttpOperation): string | null {
  const body = httpOp.parameters.body;
  if (!body) return null;
  const bodyType = body.type;
  if (bodyType.kind === "Model" && (bodyType as Model).name) {
    return (bodyType as Model).name;
  }
  return null;
}

function deriveResponseTypeName(httpOp: HttpOperation): string | null {
  for (const resp of httpOp.responses) {
    const statusCodes = resp.statusCodes;
    const isSuccess =
      statusCodes === "*" ||
      (typeof statusCodes === "number" && statusCodes >= 200 && statusCodes < 300) ||
      (typeof statusCodes === "object" &&
        "start" in statusCodes &&
        statusCodes.start >= 200 &&
        statusCodes.end < 300);
    if (!isSuccess) continue;
    return responseBodyTypeName(resp);
  }
  if (httpOp.responses.length > 0) {
    return responseBodyTypeName(httpOp.responses[0]);
  }
  return null;
}

function responseBodyTypeName(resp: HttpOperationResponse): string | null {
  for (const content of resp.responses) {
    if (!content.body) continue;
    const bodyType = content.body.type;
    if (bodyType.kind === "Model" && (bodyType as Model).name) {
      return (bodyType as Model).name;
    }
  }
  return null;
}

// --- String utilities ---

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

function requiredOpt(required: boolean): string {
  return required ? ", mcp.Required()" : "";
}

function descOpt(desc: string): string {
  if (!desc) return "";
  return `, mcp.Description("${escapeDQ(desc)}")`;
}

/** Returns the Go expression for converting an extracted value, or just the var name. */
function convertExpr(varName: string, info: GoTypeInfo): string {
  if (!info.convertExpr) return varName;
  return info.convertExpr.replace(/%s/g, varName);
}

/** Emits the Go guard for a required field extraction. String fields check for empty; others just check ok. */
function emitRequiredGuard(lines: string[], varName: string, typeInfo: GoTypeInfo): void {
  if (typeInfo.assertType === "string") {
    lines.push(`\t\tif !ok || ${varName} == "" {`);
  } else {
    lines.push(`\t\tif !ok {`);
  }
}

// --- Handler body emission ---

function emitHandlerBody(
  lines: string[],
  op: DerivedOp,
  pathParams: GoParam[],
  queryParams: GoParam[],
  bodyFields: GoParam[],
  alias: string,
): void {
  // Track declared variable names to avoid redeclaration
  const declaredVars = new Set<string>();

  // 1. Extract and validate required path params
  for (const p of pathParams) {
    lines.push(`\t\t${p.name}, ok := args["${p.name}"].(${p.typeInfo.assertType})`);
    declaredVars.add(p.name);
    emitRequiredGuard(lines, p.name, p.typeInfo);
    lines.push(`\t\t\treturn mcp.NewToolResultError("${p.name} is required"), nil`);
    lines.push(`\t\t}`);
  }

  // 2. Extract and validate required body fields
  const requiredBodyFields = bodyFields.filter((f) => f.required);
  const optionalBodyFields = bodyFields.filter((f) => !f.required);

  for (const f of requiredBodyFields) {
    if (declaredVars.has(f.name)) {
      // Variable already declared by path param — use = instead of :=
      lines.push(`\t\t${f.name}, ok = args["${f.name}"].(${f.typeInfo.assertType})`);
    } else {
      lines.push(`\t\t${f.name}, ok := args["${f.name}"].(${f.typeInfo.assertType})`);
      declaredVars.add(f.name);
    }
    emitRequiredGuard(lines, f.name, f.typeInfo);
    lines.push(`\t\t\treturn mcp.NewToolResultError("${f.name} is required"), nil`);
    lines.push(`\t\t}`);
  }

  // 3. Build params struct (if there are params)
  if (op.hasParams) {
    const inlineFields = pathParams
      .map((p) => `${p.goField}: ${convertExpr(p.name, p.typeInfo)}`)
      .join(", ");
    lines.push(`\t\tparams := ${alias}.${op.paramsType}{${inlineFields}}`);

    for (const qp of queryParams) {
      const assertType = qp.typeInfo.assertType;
      const varExpr = convertExpr("v", qp.typeInfo);
      if (assertType === "string") {
        lines.push(`\t\tif v, ok := args["${qp.name}"].(string); ok && v != "" {`);
      } else {
        lines.push(`\t\tif v, ok := args["${qp.name}"].(${assertType}); ok {`);
      }
      lines.push(`\t\t\tparams.${qp.goField}.SetTo(${varExpr})`);
      lines.push(`\t\t}`);
    }
  }

  // 4. Build body struct (if there's a body)
  if (op.bodyTypeName && bodyFields.length > 0) {
    if (requiredBodyFields.length > 0) {
      const inlineFields = requiredBodyFields
        .map((f) => `${f.goField}: ${convertExpr(f.name, f.typeInfo)}`)
        .join(", ");
      lines.push(`\t\tbody := &${alias}.${op.bodyTypeName}{${inlineFields}}`);
    } else {
      lines.push(`\t\tbody := &${alias}.${op.bodyTypeName}{}`);
    }

    for (const f of optionalBodyFields) {
      const assertType = f.typeInfo.assertType;
      const varExpr = convertExpr("v", f.typeInfo);
      lines.push(`\t\tif v, ok := args["${f.name}"].(${assertType}); ok {`);
      lines.push(`\t\t\tbody.${f.goField}.SetTo(${varExpr})`);
      lines.push(`\t\t}`);
    }
  }

  // 5. Call handler
  const callArgs: string[] = ["ctx"];
  if (op.bodyTypeName && bodyFields.length > 0) callArgs.push("body");
  if (op.hasParams) callArgs.push("params");

  if (op.responseTypeName) {
    lines.push(`\t\tresult, err := h.${op.methodName}(${callArgs.join(", ")})`);
    lines.push(`\t\tif err != nil {`);
    lines.push(`\t\t\treturn toolResultFromError(err), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\treturn mustToolResultJSON(result), nil`);
  } else {
    lines.push(`\t\tif err := h.${op.methodName}(${callArgs.join(", ")}); err != nil {`);
    lines.push(`\t\t\treturn toolResultFromError(err), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\treturn mcp.NewToolResultText("ok"), nil`);
  }
}

// --- Main generator ---

const ALIAS = "apigen";
const PKG = "mcpgen";
const ERROR_TYPE = "ApiErrorStatusCode";
const ERROR_MSG_FIELD = "Response.Message";

export function generateGoFile(program: Program, tools: ToolInfo[], opts: EmitterOptions): string {
  const lines: string[] = [];

  lines.push("// Code generated by typespec-mcp-go. DO NOT EDIT.");
  lines.push("");
  lines.push(`package ${PKG}`);
  lines.push("");
  lines.push("import (");
  lines.push('\t"context"');
  lines.push('\t"errors"');
  lines.push('\t"fmt"');
  lines.push('\t"log/slog"');
  lines.push("");
  lines.push('\t"github.com/mark3labs/mcp-go/mcp"');
  lines.push('\tmcpserver "github.com/mark3labs/mcp-go/server"');
  lines.push("");
  lines.push(`\t${ALIAS} "${opts.ogenImportPath}"`);
  lines.push(")");
  lines.push("");

  // Derive all ops upfront
  const derived = tools.map(({ httpOp }) => deriveOp(httpOp));

  // Handlers interface
  lines.push("// Handlers is the interface MCP tool calls are dispatched through.");
  lines.push("type Handlers interface {");
  for (const op of derived) {
    const parts: string[] = [`ctx context.Context`];
    if (op.bodyTypeName) parts.push(`req *${ALIAS}.${op.bodyTypeName}`);
    if (op.hasParams) parts.push(`params ${ALIAS}.${op.paramsType}`);
    const returnType = op.responseTypeName ? `(*${ALIAS}.${op.responseTypeName}, error)` : `error`;
    lines.push(`\t${op.methodName}(${parts.join(", ")}) ${returnType}`);
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
  for (let i = 0; i < tools.length; i++) {
    const { toolOpts, httpOp } = tools[i];
    const op = derived[i];
    const camel = snakeToCamel(toolOpts.name);
    const allParams = collectParams(program, httpOp, ALIAS);
    const pathParams = allParams.filter((p) => p.isPathParam);
    const queryParams = allParams.filter((p) => !p.isPathParam);
    const pathParamNames = new Set(pathParams.map((p) => p.name));
    const bodyFields = collectBodyFields(program, httpOp, ALIAS, pathParamNames);

    lines.push(`// --- ${toolOpts.name} ---`);
    lines.push("");
    lines.push(`func ${camel}Tool() mcp.Tool {`);
    lines.push(`\treturn mcp.NewTool("${toolOpts.name}",`);
    lines.push(`\t\tmcp.WithDescription("${escapeDQ(toolOpts.description)}"),`);

    for (const p of [...pathParams, ...queryParams, ...bodyFields]) {
      lines.push(
        `\t\t${p.mcpType}("${p.name}"${requiredOpt(p.required)}${descOpt(p.description)}),`,
      );
    }

    lines.push("\t)");
    lines.push("}");
    lines.push("");

    lines.push(`func ${camel}Handler(h Handlers) mcpserver.ToolHandlerFunc {`);
    lines.push(
      `\treturn func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {`,
    );
    const needsArgs = pathParams.length > 0 || queryParams.length > 0 || bodyFields.length > 0;
    if (needsArgs) {
      lines.push(`\t\targs := req.GetArguments()`);
    }

    emitHandlerBody(lines, op, pathParams, queryParams, bodyFields, ALIAS);

    lines.push(`\t}`);
    lines.push(`}`);
    lines.push("");
  }

  // Helpers
  lines.push("// --- helpers ---");
  lines.push("");
  lines.push("func toolResultFromError(err error) *mcp.CallToolResult {");
  lines.push(`\tvar apiErr *${ALIAS}.${ERROR_TYPE}`);
  lines.push("\tif errors.As(err, &apiErr) {");
  lines.push(`\t\treturn mcp.NewToolResultError(apiErr.${ERROR_MSG_FIELD})`);
  lines.push("\t}");
  lines.push('\tslog.Error("mcp: tool handler error", "error", err)');
  lines.push('\treturn mcp.NewToolResultError("internal error")');
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
