import { type Model, type Program, type Type, getDoc, isArrayModelType } from "@typespec/compiler";
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
    const variants = [...type.variants.values()].map((v) => v.type);
    const nonNull = variants.filter((v) => !(v.kind === "Intrinsic" && v.name === "null"));
    if (nonNull.length === 1 && (nonNull[0].kind === "Scalar" || nonNull[0].kind === "Enum")) {
      return nonNull[0];
    }
  }
  return null;
}

// --- MCP type mapping ---

function mcpPropertyType(type: Type): string {
  if (type.kind === "Scalar") {
    const name = type.name;
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
    const name = type.name;
    if (name === "string" || name === "url" || name === "plainDate" || name === "utcDateTime") {
      return { assertType: "string", convertExpr: "" };
    }
    if (name === "int32") return { assertType: "float64", convertExpr: "int32(%s)" };
    if (name === "int64") return { assertType: "float64", convertExpr: "int64(%s)" };
    if (name === "float32" || name === "float64") return { assertType: "float64", convertExpr: "" };
    if (name === "boolean") return { assertType: "bool", convertExpr: "" };
  }
  if (type.kind === "Enum") {
    return { assertType: "string", convertExpr: `${alias}.${type.name}(%s)` };
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

// --- Body field types (discriminated union) ---

interface GoArrayElementField {
  name: string;
  goField: string;
  typeInfo: GoTypeInfo;
  /** Enum member names, if the field type is an enum. */
  enumValues?: string[];
}

interface GoScalarBodyField {
  kind: "scalar";
  name: string;
  goField: string;
  mcpType: string;
  required: boolean;
  description: string;
  typeInfo: GoTypeInfo;
}

interface GoArrayBodyField {
  kind: "array";
  name: string;
  goField: string;
  required: boolean;
  description: string;
  elementTypeName: string;
  elementFields: GoArrayElementField[];
}

type GoBodyField = GoScalarBodyField | GoArrayBodyField;

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

function collectArrayElementFields(elementType: Model, alias: string): GoArrayElementField[] {
  const fields: GoArrayElementField[] = [];
  for (const [name, prop] of elementType.properties) {
    const resolved = getSimpleNonNull(prop.type);
    if (!resolved) {
      throw new Error(
        `Array element field "${name}" in "${elementType.name}" is not a simple scalar/enum — unsupported`,
      );
    }
    if (prop.optional) {
      throw new Error(
        `Array element field "${name}" in "${elementType.name}" is optional — unsupported`,
      );
    }
    fields.push({
      name,
      goField: capitalize(name),
      typeInfo: goTypeInfo(resolved, alias),
      enumValues:
        resolved.kind === "Enum" ? [...resolved.members.values()].map((m) => m.name) : undefined,
    });
  }
  return fields;
}

function collectBodyFields(
  program: Program,
  httpOp: HttpOperation,
  alias: string,
  pathParamNames: Set<string>,
): GoBodyField[] {
  const body = httpOp.parameters.body;
  if (!body) return [];
  const bodyType = body.type;
  if (bodyType.kind !== "Model") return [];

  const fields: GoBodyField[] = [];
  const usedNames = new Set<string>(pathParamNames);
  for (const [name, prop] of bodyType.properties) {
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

    // Try scalar first
    const resolved = getSimpleNonNull(prop.type);
    if (resolved) {
      fields.push({
        kind: "scalar",
        name: mcpName,
        goField: capitalize(name),
        mcpType: mcpPropertyType(resolved),
        required: !prop.optional,
        description: getDoc(program, prop) ?? "",
        typeInfo: goTypeInfo(resolved, alias),
      });
      continue;
    }

    // Try array of objects (skip scalar arrays — unsupported)
    if (prop.type.kind === "Model" && isArrayModelType(prop.type)) {
      const elementType = prop.type.indexer.value;
      if (elementType.kind === "Model" && elementType.name) {
        fields.push({
          kind: "array",
          name: mcpName,
          goField: capitalize(name),
          required: !prop.optional,
          description: getDoc(program, prop) ?? "",
          elementTypeName: elementType.name,
          elementFields: collectArrayElementFields(elementType, alias),
        });
        continue;
      }
    }

    // Warn about skipped unsupported body field types
    program.reportDiagnostic({
      code: "typespec-mcp-go/unsupported-body-field",
      severity: "warning",
      message: `Body field "${name}" has unsupported type "${prop.type.kind}" and will be omitted from the MCP tool.`,
      target: prop,
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
  const iface = httpOp.container?.name ?? "";
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
  if (bodyType.kind === "Model" && bodyType.name) {
    return bodyType.name;
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
    if (bodyType.kind === "Model" && bodyType.name) {
      return bodyType.name;
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
function emitRequiredGuard(
  lines: string[],
  varName: string,
  typeInfo: GoTypeInfo,
  indent = "\t\t",
): void {
  if (typeInfo.assertType === "string") {
    lines.push(`${indent}if !ok || ${varName} == "" {`);
  } else {
    lines.push(`${indent}if !ok {`);
  }
}

// --- Handler body emission ---

function emitArrayExtraction(lines: string[], field: GoArrayBodyField, alias: string): void {
  const varName = `raw${capitalize(field.name)}`;
  const sliceVar = field.name;
  const elemType = `${alias}.${field.elementTypeName}`;

  if (field.required) {
    lines.push(`\t\t${varName}, ok := args["${field.name}"].([]any)`);
    lines.push(`\t\tif !ok {`);
    lines.push(`\t\t\treturn mcp.NewToolResultError("${field.name} is required"), nil`);
    lines.push(`\t\t}`);
  } else {
    lines.push(`\t\t${varName}, _ := args["${field.name}"].([]any)`);
  }

  lines.push(`\t\t${sliceVar} := make([]${elemType}, len(${varName}))`);
  lines.push(`\t\tfor i, raw := range ${varName} {`);
  lines.push(`\t\t\tm, ok := raw.(map[string]any)`);
  lines.push(`\t\t\tif !ok {`);
  lines.push(
    `\t\t\t\treturn mcp.NewToolResultError(fmt.Sprintf("${field.name}[%d] must be an object", i)), nil`,
  );
  lines.push(`\t\t\t}`);
  for (const ef of field.elementFields) {
    const extracted = `m["${ef.name}"].(${ef.typeInfo.assertType})`;
    lines.push(`\t\t\tv${capitalize(ef.name)}, ok := ${extracted}`);
    emitRequiredGuard(lines, `v${capitalize(ef.name)}`, ef.typeInfo, "\t\t\t");
    lines.push(
      `\t\t\t\treturn mcp.NewToolResultError(fmt.Sprintf("${field.name}[%d].${ef.name} is required", i)), nil`,
    );
    lines.push(`\t\t\t}`);
  }
  const structFields = field.elementFields
    .map((ef) => `${ef.goField}: ${convertExpr(`v${capitalize(ef.name)}`, ef.typeInfo)}`)
    .join(", ");
  lines.push(`\t\t\t${sliceVar}[i] = ${elemType}{${structFields}}`);
  lines.push(`\t\t}`);
}

function emitHandlerBody(
  lines: string[],
  op: DerivedOp,
  pathParams: GoParam[],
  queryParams: GoParam[],
  bodyFields: GoBodyField[],
  hasBody: boolean,
  alias: string,
): void {
  // Track declared variable names to avoid redeclaration
  const declaredVars = new Set<string>();

  const scalarBodyFields = bodyFields.filter((f): f is GoScalarBodyField => f.kind === "scalar");
  const arrayBodyFields = bodyFields.filter((f): f is GoArrayBodyField => f.kind === "array");

  // 1. Extract and validate required path params
  for (const p of pathParams) {
    lines.push(`\t\t${p.name}, ok := args["${p.name}"].(${p.typeInfo.assertType})`);
    declaredVars.add(p.name);
    emitRequiredGuard(lines, p.name, p.typeInfo);
    lines.push(`\t\t\treturn mcp.NewToolResultError("${p.name} is required"), nil`);
    lines.push(`\t\t}`);
  }

  // 2. Extract and validate required scalar body fields
  const requiredScalarBodyFields = scalarBodyFields.filter((f) => f.required);
  const optionalScalarBodyFields = scalarBodyFields.filter((f) => !f.required);

  for (const f of requiredScalarBodyFields) {
    if (declaredVars.has(f.name)) {
      lines.push(`\t\t${f.name}, ok = args["${f.name}"].(${f.typeInfo.assertType})`);
    } else {
      lines.push(`\t\t${f.name}, ok := args["${f.name}"].(${f.typeInfo.assertType})`);
      declaredVars.add(f.name);
    }
    emitRequiredGuard(lines, f.name, f.typeInfo);
    lines.push(`\t\t\treturn mcp.NewToolResultError("${f.name} is required"), nil`);
    lines.push(`\t\t}`);
  }

  // 3. Extract and validate array body fields
  for (const f of arrayBodyFields) {
    emitArrayExtraction(lines, f, alias);
    declaredVars.add(f.name);
  }

  // 4. Build params struct (if there are params)
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

  // 5. Build body struct (if there's a body)
  if (hasBody) {
    // Collect inline fields: required scalars go inline, array fields reference pre-built slices
    const inlineFields: string[] = [];
    for (const f of requiredScalarBodyFields) {
      inlineFields.push(`${f.goField}: ${convertExpr(f.name, f.typeInfo)}`);
    }
    for (const f of arrayBodyFields) {
      inlineFields.push(`${f.goField}: ${f.name}`);
    }

    if (inlineFields.length > 0) {
      lines.push(`\t\tbody := &${alias}.${op.bodyTypeName}{${inlineFields.join(", ")}}`);
    } else {
      lines.push(`\t\tbody := &${alias}.${op.bodyTypeName}{}`);
    }

    for (const f of optionalScalarBodyFields) {
      const assertType = f.typeInfo.assertType;
      const varExpr = convertExpr("v", f.typeInfo);
      lines.push(`\t\tif v, ok := args["${f.name}"].(${assertType}); ok {`);
      lines.push(`\t\t\tbody.${f.goField}.SetTo(${varExpr})`);
      lines.push(`\t\t}`);
    }
  }

  // 6. Call handler
  const callArgs: string[] = ["ctx"];
  if (hasBody) callArgs.push("body");
  if (op.hasParams) callArgs.push("params");

  if (op.responseTypeName) {
    lines.push(`\t\tresult, err := h.${op.methodName}(${callArgs.join(", ")})`);
    lines.push(`\t\tif err != nil {`);
    lines.push(`\t\t\treturn mcp.NewToolResultError(h.ConvertError(ctx, err)), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\treturn mustToolResultJSON(result), nil`);
  } else {
    lines.push(`\t\tif err := h.${op.methodName}(${callArgs.join(", ")}); err != nil {`);
    lines.push(`\t\t\treturn mcp.NewToolResultError(h.ConvertError(ctx, err)), nil`);
    lines.push(`\t\t}`);
    lines.push(`\t\treturn mcp.NewToolResultText("ok"), nil`);
  }
}

// --- Main generator ---

const ALIAS = "apigen";
const PKG = "mcpgen";

export function generateGoFile(program: Program, tools: ToolInfo[], opts: EmitterOptions): string {
  const lines: string[] = [];

  lines.push("// Code generated by typespec-mcp-go. DO NOT EDIT.");
  lines.push("");
  lines.push(`package ${PKG}`);
  lines.push("");
  lines.push("import (");
  lines.push('\t"context"');
  lines.push('\t"fmt"');
  lines.push("");
  lines.push('\t"github.com/mark3labs/mcp-go/mcp"');
  lines.push('\tmcpserver "github.com/mark3labs/mcp-go/server"');
  lines.push("");
  lines.push(`\t${ALIAS} "${opts.ogenImportPath}"`);
  lines.push(")");
  lines.push("");

  // Precompute all per-tool data
  const toolData = tools.map(({ httpOp, toolOpts }) => {
    const op = deriveOp(httpOp);
    const camel = snakeToCamel(toolOpts.name);
    const allParams = collectParams(program, httpOp, ALIAS);
    const pathParams = allParams.filter((p) => p.isPathParam);
    const queryParams = allParams.filter((p) => !p.isPathParam);
    const pathParamNames = new Set(pathParams.map((p) => p.name));
    const bodyFields = collectBodyFields(program, httpOp, ALIAS, pathParamNames);
    const hasBody = op.bodyTypeName !== null && bodyFields.length > 0;
    return { op, camel, toolOpts, httpOp, pathParams, queryParams, bodyFields, hasBody };
  });

  // Handlers interface
  lines.push("// Handlers is the interface MCP tool calls are dispatched through.");
  lines.push("type Handlers interface {");
  lines.push("\t// ConvertError maps a handler error to a user-facing message.");
  lines.push("\tConvertError(ctx context.Context, err error) string");
  for (const { op, hasBody } of toolData) {
    const parts: string[] = [`ctx context.Context`];
    if (hasBody) parts.push(`req *${ALIAS}.${op.bodyTypeName}`);
    if (op.hasParams) parts.push(`params ${ALIAS}.${op.paramsType}`);
    const returnType = op.responseTypeName ? `(*${ALIAS}.${op.responseTypeName}, error)` : `error`;
    lines.push(`\t${op.methodName}(${parts.join(", ")}) ${returnType}`);
  }
  lines.push("}");
  lines.push("");

  // RegisterTools
  lines.push("// RegisterTools registers all @mcpTool-annotated operations with the MCP server.");
  lines.push("func RegisterTools(s *mcpserver.MCPServer, h Handlers) {");
  for (const { camel } of toolData) {
    lines.push(`\ts.AddTool(${camel}Tool(), ${camel}Handler(h))`);
  }
  lines.push("}");
  lines.push("");

  // Per-tool definitions
  for (const td of toolData) {
    const { toolOpts, op, camel, pathParams, queryParams, bodyFields, hasBody } = td;

    lines.push(`// --- ${toolOpts.name} ---`);
    lines.push("");
    lines.push(`func ${camel}Tool() mcp.Tool {`);
    lines.push(`\treturn mcp.NewTool("${toolOpts.name}",`);
    lines.push(`\t\tmcp.WithDescription("${escapeDQ(toolOpts.description)}"),`);

    for (const p of [...pathParams, ...queryParams]) {
      lines.push(
        `\t\t${p.mcpType}("${p.name}"${requiredOpt(p.required)}${descOpt(p.description)}),`,
      );
    }
    for (const f of bodyFields) {
      if (f.kind === "scalar") {
        lines.push(
          `\t\t${f.mcpType}("${f.name}"${requiredOpt(f.required)}${descOpt(f.description)}),`,
        );
      } else {
        lines.push(
          `\t\tmcp.WithArray("${f.name}"${requiredOpt(f.required)}${descOpt(f.description)},`,
        );
        lines.push(`\t\t\tmcp.Items(map[string]any{`);
        lines.push(`\t\t\t\t"type": "object",`);
        lines.push(`\t\t\t\t"properties": map[string]any{`);
        for (const ef of f.elementFields) {
          const jsonType =
            ef.typeInfo.assertType === "string"
              ? "string"
              : ef.typeInfo.assertType === "float64"
                ? "number"
                : ef.typeInfo.assertType === "bool"
                  ? "boolean"
                  : "string";
          if (ef.enumValues) {
            const enumList = ef.enumValues.map((v) => `"${escapeDQ(v)}"`).join(", ");
            lines.push(
              `\t\t\t\t\t"${ef.name}": map[string]any{"type": "${jsonType}", "enum": []any{${enumList}}},`,
            );
          } else {
            lines.push(`\t\t\t\t\t"${ef.name}": map[string]any{"type": "${jsonType}"},`);
          }
        }
        lines.push(`\t\t\t\t},`);
        const requiredFields = f.elementFields.map((ef) => `"${ef.name}"`).join(", ");
        lines.push(`\t\t\t\t"required": []string{${requiredFields}},`);
        lines.push(`\t\t\t}),`);
        lines.push(`\t\t),`);
      }
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

    emitHandlerBody(lines, op, pathParams, queryParams, bodyFields, hasBody, ALIAS);

    lines.push(`\t}`);
    lines.push(`}`);
    lines.push("");
  }

  // Helpers
  lines.push("// --- helpers ---");
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
