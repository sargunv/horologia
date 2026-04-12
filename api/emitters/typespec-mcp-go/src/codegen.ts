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

interface ResolvedType<T extends Type> {
  type: T;
  nullable: boolean;
}

function splitNullable(type: Type): { nonNull: Type[]; nullable: boolean } {
  if (type.kind === "Union") {
    const variants = [...type.variants.values()].map((v) => v.type);
    const nonNull = variants.filter((v) => !(v.kind === "Intrinsic" && v.name === "null"));
    return { nonNull, nullable: nonNull.length !== variants.length };
  }
  return { nonNull: [type], nullable: false };
}

function getSimpleType(type: Type): ResolvedType<Type> | null {
  const { nonNull, nullable } = splitNullable(type);
  if (nonNull.length !== 1) return null;
  const resolved = nonNull[0];
  if (resolved.kind === "Scalar" || resolved.kind === "Enum") {
    return { type: resolved, nullable };
  }
  return null;
}

function getObjectType(type: Type): ResolvedType<Model> | null {
  const { nonNull, nullable } = splitNullable(type);
  if (nonNull.length !== 1) return null;
  const resolved = nonNull[0];
  if (resolved.kind === "Model" && !isArrayModelType(resolved)) {
    return { type: resolved, nullable };
  }
  return null;
}

function getArrayElementType(type: Type): ResolvedType<Type> | null {
  const { nonNull, nullable } = splitNullable(type);
  if (nonNull.length !== 1) return null;
  const resolved = nonNull[0];
  if (resolved.kind === "Model" && isArrayModelType(resolved)) {
    return { type: resolved.indexer.value, nullable };
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
  /** The concrete Go value type after conversion. */
  valueType: string;
  /** JSON schema type for the value. */
  schemaType: "string" | "number" | "boolean";
  /** TypeSpec scalar name, when the source type is a scalar. */
  scalarName?: string;
  /** Enum member names, if the field type is an enum. */
  enumValues?: string[];
}

function goTypeInfo(type: Type, alias: string): GoTypeInfo {
  if (type.kind === "Scalar") {
    const name = type.name;
    if (name === "string" || name === "url") {
      return {
        assertType: "string",
        convertExpr: "",
        valueType: "string",
        schemaType: "string",
        scalarName: name,
      };
    }
    if (name === "plainDate" || name === "utcDateTime") {
      return {
        assertType: "string",
        convertExpr: "",
        valueType: "time.Time",
        schemaType: "string",
        scalarName: name,
      };
    }
    if (name === "int32") {
      return {
        assertType: "float64",
        convertExpr: "int32(%s)",
        valueType: "int32",
        schemaType: "number",
        scalarName: name,
      };
    }
    if (name === "int64") {
      return {
        assertType: "float64",
        convertExpr: "int64(%s)",
        valueType: "int64",
        schemaType: "number",
        scalarName: name,
      };
    }
    if (name === "float32") {
      return {
        assertType: "float64",
        convertExpr: "float32(%s)",
        valueType: "float32",
        schemaType: "number",
        scalarName: name,
      };
    }
    if (name === "float64" || name === "integer" || name === "numeric") {
      return {
        assertType: "float64",
        convertExpr: "",
        valueType: "float64",
        schemaType: "number",
        scalarName: name,
      };
    }
    if (name === "boolean") {
      return {
        assertType: "bool",
        convertExpr: "",
        valueType: "bool",
        schemaType: "boolean",
        scalarName: name,
      };
    }
  }
  if (type.kind === "Enum") {
    return {
      assertType: "string",
      convertExpr: `${alias}.${type.name}(%s)`,
      valueType: `${alias}.${type.name}`,
      schemaType: "string",
      enumValues: [...type.members.values()].map((m) => m.name),
    };
  }
  return { assertType: "string", convertExpr: "", valueType: "string", schemaType: "string" };
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

interface GoObjectFieldMember {
  name: string;
  goField: string;
  required: boolean;
  description: string;
  typeInfo: GoTypeInfo;
  nullable: boolean;
}

interface GoScalarBodyField {
  kind: "scalar";
  name: string;
  goField: string;
  mcpType: string;
  required: boolean;
  description: string;
  typeInfo: GoTypeInfo;
  nullable: boolean;
}

interface GoScalarArrayBodyField {
  kind: "scalarArray";
  name: string;
  goField: string;
  required: boolean;
  description: string;
  elementTypeInfo: GoTypeInfo;
}

interface GoObjectBodyField {
  kind: "object";
  name: string;
  goField: string;
  required: boolean;
  description: string;
  objectTypeName: string;
  members: GoObjectFieldMember[];
  nullable: boolean;
}

interface GoObjectArrayBodyField {
  kind: "objectArray";
  name: string;
  goField: string;
  required: boolean;
  description: string;
  elementTypeName: string;
  elementFields: GoObjectFieldMember[];
}

type GoBodyField =
  | GoScalarBodyField
  | GoScalarArrayBodyField
  | GoObjectBodyField
  | GoObjectArrayBodyField;

function collectParams(program: Program, httpOp: HttpOperation, alias: string): GoParam[] {
  const params: GoParam[] = [];
  for (const param of httpOp.parameters.parameters) {
    const p = param.param;
    const resolved = getSimpleType(p.type);
    if (!resolved) continue;
    params.push({
      name: p.name,
      goField: capitalize(p.name),
      mcpType: mcpPropertyType(resolved.type),
      required: param.type === "path" || !p.optional,
      description: getDoc(program, p) ?? "",
      isPathParam: param.type === "path",
      typeInfo: goTypeInfo(resolved.type, alias),
    });
  }
  return params;
}

function collectObjectFields(
  program: Program,
  objectType: Model,
  alias: string,
): GoObjectFieldMember[] {
  const fields: GoObjectFieldMember[] = [];
  for (const [name, prop] of objectType.properties) {
    const resolved = getSimpleType(prop.type);
    if (!resolved) {
      throw new Error(
        `Object field "${name}" in "${objectType.name}" is not a simple scalar/enum — unsupported`,
      );
    }
    fields.push({
      name,
      goField: capitalize(name),
      required: !prop.optional,
      description: getDoc(program, prop) ?? "",
      typeInfo: goTypeInfo(resolved.type, alias),
      nullable: resolved.nullable,
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

    const simple = getSimpleType(prop.type);
    if (simple) {
      fields.push({
        kind: "scalar",
        name: mcpName,
        goField: capitalize(name),
        mcpType: mcpPropertyType(simple.type),
        required: !prop.optional,
        description: getDoc(program, prop) ?? "",
        typeInfo: goTypeInfo(simple.type, alias),
        nullable: simple.nullable,
      });
      continue;
    }

    const array = getArrayElementType(prop.type);
    if (array) {
      const elementSimple = getSimpleType(array.type);
      if (elementSimple) {
        fields.push({
          kind: "scalarArray",
          name: mcpName,
          goField: capitalize(name),
          required: !prop.optional,
          description: getDoc(program, prop) ?? "",
          elementTypeInfo: goTypeInfo(elementSimple.type, alias),
        });
        continue;
      }
      if (array.type.kind === "Model" && array.type.name) {
        fields.push({
          kind: "objectArray",
          name: mcpName,
          goField: capitalize(name),
          required: !prop.optional,
          description: getDoc(program, prop) ?? "",
          elementTypeName: array.type.name,
          elementFields: collectObjectFields(program, array.type, alias),
        });
        continue;
      }
    }

    const object = getObjectType(prop.type);
    if (object?.type.name) {
      fields.push({
        kind: "object",
        name: mcpName,
        goField: capitalize(name),
        required: !prop.optional,
        description: getDoc(program, prop) ?? "",
        objectTypeName: object.type.name,
        members: collectObjectFields(program, object.type, alias),
        nullable: object.nullable,
      });
      continue;
    }

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
  return s.replace(/\\/g, "\\\\").replace(/\r/g, "").replace(/\n/g, "\\n").replace(/"/g, '\\"');
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

function enumOpt(values?: string[]): string {
  if (!values?.length) return "";
  return `, mcp.Enum(${values.map((v) => `"${escapeDQ(v)}"`).join(", ")})`;
}

function convertExpr(varName: string, info: GoTypeInfo): string {
  if (!info.convertExpr) return varName;
  return info.convertExpr.replace(/%s/g, varName);
}

function typeNeedsTime(info: GoTypeInfo): boolean {
  return info.scalarName === "plainDate" || info.scalarName === "utcDateTime";
}

function bodyFieldNeedsTime(field: GoBodyField): boolean {
  switch (field.kind) {
    case "scalar":
      return typeNeedsTime(field.typeInfo);
    case "scalarArray":
      return typeNeedsTime(field.elementTypeInfo);
    case "object":
      return field.members.some((member) => typeNeedsTime(member.typeInfo));
    case "objectArray":
      return field.elementFields.some((member) => typeNeedsTime(member.typeInfo));
  }
}

function emitSimpleSchemaMap(
  lines: string[],
  indent: string,
  info: GoTypeInfo,
  nullable: boolean,
  description: string,
): void {
  if (nullable) {
    lines.push(`${indent}"type": []any{"${info.schemaType}", "null"},`);
  } else {
    lines.push(`${indent}"type": "${info.schemaType}",`);
  }
  if (description) {
    lines.push(`${indent}"description": "${escapeDQ(description)}",`);
  }
  if (info.enumValues?.length) {
    lines.push(
      `${indent}"enum": []any{${info.enumValues.map((v) => `"${escapeDQ(v)}"`).join(", ")}},`,
    );
  }
}

function emitConvertedValue(
  lines: string[],
  indent: string,
  outVar: string,
  inVar: string,
  info: GoTypeInfo,
  invalidMessageExpr: string,
): void {
  if (info.scalarName === "plainDate") {
    lines.push(`${indent}${outVar}, err := time.Parse(time.DateOnly, ${inVar})`);
    lines.push(`${indent}if err != nil {`);
    lines.push(`${indent}\treturn mcp.NewToolResultError(${invalidMessageExpr}), nil`);
    lines.push(`${indent}}`);
    return;
  }
  if (info.scalarName === "utcDateTime") {
    lines.push(`${indent}${outVar}, err := time.Parse(time.RFC3339, ${inVar})`);
    lines.push(`${indent}if err != nil {`);
    lines.push(`${indent}\treturn mcp.NewToolResultError(${invalidMessageExpr}), nil`);
    lines.push(`${indent}}`);
    return;
  }
  lines.push(`${indent}${outVar} := ${convertExpr(inVar, info)}`);
}

function emitAssignValue(
  lines: string[],
  indent: string,
  target: string,
  valueVar: string,
  required: boolean,
  nullable: boolean,
): void {
  if (required && !nullable) {
    lines.push(`${indent}${target} = ${valueVar}`);
  } else {
    lines.push(`${indent}${target}.SetTo(${valueVar})`);
  }
}

function emitScalarFieldExtraction(lines: string[], field: GoScalarBodyField): void {
  const rawVar = `raw${field.goField}`;
  const hasVar = `has${field.goField}`;
  const valueVar = `v${field.goField}`;
  const convertedVar = `converted${field.goField}`;
  const requiredMessage = `"${escapeDQ(`${field.name} is required`)}"`;
  const invalidTypeMessage = `"${escapeDQ(`${field.name} must be a ${field.typeInfo.schemaType}`)}"`;
  const invalidFormatMessage =
    field.typeInfo.scalarName === "plainDate"
      ? `"${escapeDQ(`${field.name} must be a valid date`)}"`
      : field.typeInfo.scalarName === "utcDateTime"
        ? `"${escapeDQ(`${field.name} must be a valid RFC3339 timestamp`)}"`
        : invalidTypeMessage;

  lines.push(`\t\t${rawVar}, ${hasVar} := args["${field.name}"]`);
  lines.push(`\t\tif ${hasVar} {`);
  if (field.nullable) {
    lines.push(`\t\t\tif ${rawVar} == nil {`);
    lines.push(`\t\t\t\tbody.${field.goField}.SetToNull()`);
    lines.push(`\t\t\t} else {`);
  } else {
    lines.push(`\t\t\tif ${rawVar} == nil {`);
    lines.push(
      `\t\t\t\treturn mcp.NewToolResultError(${field.required ? requiredMessage : `"${escapeDQ(`${field.name} must not be null`)}"`}), nil`,
    );
    lines.push(`\t\t\t}`);
  }
  lines.push(`\t\t\t\t${valueVar}, ok := ${rawVar}.(${field.typeInfo.assertType})`);
  if (field.required && field.typeInfo.assertType === "string") {
    lines.push(`\t\t\t\tif !ok || ${valueVar} == "" {`);
    lines.push(`\t\t\t\t\treturn mcp.NewToolResultError(${requiredMessage}), nil`);
  } else {
    lines.push(`\t\t\t\tif !ok {`);
    lines.push(`\t\t\t\t\treturn mcp.NewToolResultError(${invalidTypeMessage}), nil`);
  }
  lines.push(`\t\t\t\t}`);
  emitConvertedValue(
    lines,
    "\t\t\t\t",
    convertedVar,
    valueVar,
    field.typeInfo,
    invalidFormatMessage,
  );
  emitAssignValue(
    lines,
    "\t\t\t\t",
    `body.${field.goField}`,
    convertedVar,
    field.required,
    field.nullable,
  );
  if (field.nullable) {
    lines.push(`\t\t\t}`);
  }
  lines.push(`\t\t} else if ${String(field.required)} {`);
  lines.push(`\t\t\treturn mcp.NewToolResultError(${requiredMessage}), nil`);
  lines.push(`\t\t}`);
}

function emitScalarArrayFieldExtraction(lines: string[], field: GoScalarArrayBodyField): void {
  const rawVar = `raw${field.goField}`;
  const hasVar = `has${field.goField}`;
  const rawItemsVar = `raw${field.goField}Items`;
  const itemsVar = `${field.name}Items`;
  const invalidArrayMessage = `"${escapeDQ(`${field.name} must be an array`)}"`;
  const invalidTypeMessageTemplate = `fmt.Sprintf("${escapeDQ(`${field.name}[%d] must be a ${field.elementTypeInfo.schemaType}`)}", i)`;
  const invalidFormatMessageTemplate =
    field.elementTypeInfo.scalarName === "plainDate"
      ? `fmt.Sprintf("${escapeDQ(`${field.name}[%d] must be a valid date`)}", i)`
      : field.elementTypeInfo.scalarName === "utcDateTime"
        ? `fmt.Sprintf("${escapeDQ(`${field.name}[%d] must be a valid RFC3339 timestamp`)}", i)`
        : invalidTypeMessageTemplate;

  lines.push(`\t\t${rawVar}, ${hasVar} := args["${field.name}"]`);
  lines.push(`\t\tif ${hasVar} {`);
  lines.push(`\t\t\tif ${rawVar} == nil {`);
  lines.push(`\t\t\t\treturn mcp.NewToolResultError(${invalidArrayMessage}), nil`);
  lines.push(`\t\t\t}`);
  lines.push(`\t\t\t${rawItemsVar}, ok := ${rawVar}.([]any)`);
  lines.push(`\t\t\tif !ok {`);
  lines.push(`\t\t\t\treturn mcp.NewToolResultError(${invalidArrayMessage}), nil`);
  lines.push(`\t\t\t}`);
  lines.push(
    `\t\t\t${itemsVar} := make([]${field.elementTypeInfo.valueType}, len(${rawItemsVar}))`,
  );
  lines.push(`\t\t\tfor i, raw := range ${rawItemsVar} {`);
  lines.push(`\t\t\t\tv, ok := raw.(${field.elementTypeInfo.assertType})`);
  lines.push(`\t\t\t\tif !ok {`);
  lines.push(`\t\t\t\t\treturn mcp.NewToolResultError(${invalidTypeMessageTemplate}), nil`);
  lines.push(`\t\t\t\t}`);
  emitConvertedValue(
    lines,
    "\t\t\t\t",
    "converted",
    "v",
    field.elementTypeInfo,
    invalidFormatMessageTemplate,
  );
  lines.push(`\t\t\t\t${itemsVar}[i] = converted`);
  lines.push(`\t\t\t}`);
  lines.push(`\t\t\tbody.${field.goField} = ${itemsVar}`);
  lines.push(`\t\t} else if ${String(field.required)} {`);
  lines.push(
    `\t\t\treturn mcp.NewToolResultError("${escapeDQ(`${field.name} is required`)}"), nil`,
  );
  lines.push(`\t\t}`);
}

function emitObjectMembers(
  lines: string[],
  members: GoObjectFieldMember[],
  objectVar: string,
  mapVar: string,
  labelPrefix: string,
  isIndexed: boolean,
): void {
  for (const member of members) {
    const rawVar = `raw${objectVar}${member.goField}`;
    const hasVar = `has${objectVar}${member.goField}`;
    const valueVar = `v${objectVar}${member.goField}`;
    const convertedVar = `converted${objectVar}${member.goField}`;
    const requiredMessage = isIndexed
      ? `fmt.Sprintf("${escapeDQ(`${labelPrefix}.${member.name} is required`)}", i)`
      : `"${escapeDQ(`${labelPrefix}.${member.name} is required`)}"`;
    const invalidNullMessage = isIndexed
      ? `fmt.Sprintf("${escapeDQ(`${labelPrefix}.${member.name} must not be null`)}", i)`
      : `"${escapeDQ(`${labelPrefix}.${member.name} must not be null`)}"`;
    const invalidTypeMessage = isIndexed
      ? `fmt.Sprintf("${escapeDQ(`${labelPrefix}.${member.name} must be a ${member.typeInfo.schemaType}`)}", i)`
      : `"${escapeDQ(`${labelPrefix}.${member.name} must be a ${member.typeInfo.schemaType}`)}"`;
    const invalidFormatMessage =
      member.typeInfo.scalarName === "plainDate"
        ? isIndexed
          ? `fmt.Sprintf("${escapeDQ(`${labelPrefix}.${member.name} must be a valid date`)}", i)`
          : `"${escapeDQ(`${labelPrefix}.${member.name} must be a valid date`)}"`
        : member.typeInfo.scalarName === "utcDateTime"
          ? isIndexed
            ? `fmt.Sprintf("${escapeDQ(`${labelPrefix}.${member.name} must be a valid RFC3339 timestamp`)}", i)`
            : `"${escapeDQ(`${labelPrefix}.${member.name} must be a valid RFC3339 timestamp`)}"`
          : invalidTypeMessage;

    lines.push(`\t\t\t\t${rawVar}, ${hasVar} := ${mapVar}["${member.name}"]`);
    lines.push(`\t\t\t\tif ${hasVar} {`);
    if (member.nullable) {
      lines.push(`\t\t\t\t\tif ${rawVar} == nil {`);
      lines.push(`\t\t\t\t\t\t${objectVar}.${member.goField}.SetToNull()`);
      lines.push(`\t\t\t\t\t} else {`);
    } else {
      lines.push(`\t\t\t\t\tif ${rawVar} == nil {`);
      lines.push(
        `\t\t\t\t\t\treturn mcp.NewToolResultError(${member.required ? requiredMessage : invalidNullMessage}), nil`,
      );
      lines.push(`\t\t\t\t\t}`);
    }
    lines.push(`\t\t\t\t\t\t${valueVar}, ok := ${rawVar}.(${member.typeInfo.assertType})`);
    if (member.required && member.typeInfo.assertType === "string") {
      lines.push(`\t\t\t\t\t\tif !ok || ${valueVar} == "" {`);
      lines.push(`\t\t\t\t\t\t\treturn mcp.NewToolResultError(${requiredMessage}), nil`);
    } else {
      lines.push(`\t\t\t\t\t\tif !ok {`);
      lines.push(`\t\t\t\t\t\t\treturn mcp.NewToolResultError(${invalidTypeMessage}), nil`);
    }
    lines.push(`\t\t\t\t\t\t}`);
    emitConvertedValue(
      lines,
      "\t\t\t\t\t\t",
      convertedVar,
      valueVar,
      member.typeInfo,
      invalidFormatMessage,
    );
    emitAssignValue(
      lines,
      "\t\t\t\t\t\t",
      `${objectVar}.${member.goField}`,
      convertedVar,
      member.required,
      member.nullable,
    );
    if (member.nullable) {
      lines.push(`\t\t\t\t\t}`);
    }
    lines.push(`\t\t\t\t} else if ${String(member.required)} {`);
    lines.push(`\t\t\t\t\treturn mcp.NewToolResultError(${requiredMessage}), nil`);
    lines.push(`\t\t\t\t}`);
  }
}

function emitObjectFieldExtraction(lines: string[], field: GoObjectBodyField, alias: string): void {
  const rawVar = `raw${field.goField}`;
  const hasVar = `has${field.goField}`;
  const mapVar = `m${field.goField}`;
  const objectVar = `value${field.goField}`;
  const requiredMessage = `"${escapeDQ(`${field.name} is required`)}"`;
  const invalidObjectMessage = `"${escapeDQ(`${field.name} must be an object`)}"`;

  lines.push(`\t\t${rawVar}, ${hasVar} := args["${field.name}"]`);
  lines.push(`\t\tif ${hasVar} {`);
  if (field.nullable) {
    lines.push(`\t\t\tif ${rawVar} == nil {`);
    lines.push(`\t\t\t\tbody.${field.goField}.SetToNull()`);
    lines.push(`\t\t\t} else {`);
  } else {
    lines.push(`\t\t\tif ${rawVar} == nil {`);
    lines.push(
      `\t\t\t\treturn mcp.NewToolResultError(${field.required ? requiredMessage : invalidObjectMessage}), nil`,
    );
    lines.push(`\t\t\t}`);
  }
  lines.push(`\t\t\t\t${mapVar}, ok := ${rawVar}.(map[string]any)`);
  lines.push(`\t\t\t\tif !ok {`);
  lines.push(`\t\t\t\t\treturn mcp.NewToolResultError(${invalidObjectMessage}), nil`);
  lines.push(`\t\t\t\t}`);
  lines.push(`\t\t\t\tvar ${objectVar} ${alias}.${field.objectTypeName}`);
  emitObjectMembers(lines, field.members, objectVar, mapVar, field.name, false);
  emitAssignValue(
    lines,
    "\t\t\t\t",
    `body.${field.goField}`,
    objectVar,
    field.required,
    field.nullable,
  );
  if (field.nullable) {
    lines.push(`\t\t\t}`);
  }
  lines.push(`\t\t} else if ${String(field.required)} {`);
  lines.push(`\t\t\treturn mcp.NewToolResultError(${requiredMessage}), nil`);
  lines.push(`\t\t}`);
}

function emitObjectArrayFieldExtraction(
  lines: string[],
  field: GoObjectArrayBodyField,
  alias: string,
): void {
  const rawVar = `raw${field.goField}`;
  const hasVar = `has${field.goField}`;
  const rawItemsVar = `raw${field.goField}Items`;
  const itemsVar = `${field.name}Items`;
  const invalidArrayMessage = `"${escapeDQ(`${field.name} must be an array`)}"`;
  const requiredMessage = `"${escapeDQ(`${field.name} is required`)}"`;

  lines.push(`\t\t${rawVar}, ${hasVar} := args["${field.name}"]`);
  lines.push(`\t\tif ${hasVar} {`);
  lines.push(`\t\t\tif ${rawVar} == nil {`);
  lines.push(`\t\t\t\treturn mcp.NewToolResultError(${invalidArrayMessage}), nil`);
  lines.push(`\t\t\t}`);
  lines.push(`\t\t\t${rawItemsVar}, ok := ${rawVar}.([]any)`);
  lines.push(`\t\t\tif !ok {`);
  lines.push(`\t\t\t\treturn mcp.NewToolResultError(${invalidArrayMessage}), nil`);
  lines.push(`\t\t\t}`);
  lines.push(`\t\t\t${itemsVar} := make([]${alias}.${field.elementTypeName}, len(${rawItemsVar}))`);
  lines.push(`\t\t\tfor i, raw := range ${rawItemsVar} {`);
  lines.push(`\t\t\t\tm, ok := raw.(map[string]any)`);
  lines.push(`\t\t\t\tif !ok {`);
  lines.push(
    `\t\t\t\t\treturn mcp.NewToolResultError(fmt.Sprintf("${escapeDQ(`${field.name}[%d] must be an object`)}", i)), nil`,
  );
  lines.push(`\t\t\t\t}`);
  lines.push(`\t\t\t\tvar value ${alias}.${field.elementTypeName}`);
  emitObjectMembers(lines, field.elementFields, "value", "m", `${field.name}[%d]`, true);
  lines.push(`\t\t\t\t${itemsVar}[i] = value`);
  lines.push(`\t\t\t}`);
  lines.push(`\t\t\tbody.${field.goField} = ${itemsVar}`);
  lines.push(`\t\t} else if ${String(field.required)} {`);
  lines.push(`\t\t\treturn mcp.NewToolResultError(${requiredMessage}), nil`);
  lines.push(`\t\t}`);
}

// --- Handler body emission ---

function emitHandlerBody(
  lines: string[],
  op: DerivedOp,
  pathParams: GoParam[],
  queryParams: GoParam[],
  bodyFields: GoBodyField[],
  hasBody: boolean,
  alias: string,
): void {
  for (const p of pathParams) {
    lines.push(`\t\t${p.name}, ok := args["${p.name}"].(${p.typeInfo.assertType})`);
    if (p.typeInfo.assertType === "string") {
      lines.push(`\t\tif !ok || ${p.name} == "" {`);
    } else {
      lines.push(`\t\tif !ok {`);
    }
    lines.push(`\t\t\treturn mcp.NewToolResultError("${escapeDQ(`${p.name} is required`)}"), nil`);
    lines.push(`\t\t}`);
  }

  if (op.hasParams) {
    const inlineFields = pathParams
      .map((p) => `${p.goField}: ${convertExpr(p.name, p.typeInfo)}`)
      .join(", ");
    lines.push(`\t\tparams := ${alias}.${op.paramsType}{${inlineFields}}`);

    for (const qp of queryParams) {
      const assertType = qp.typeInfo.assertType;
      const convertedVar = `converted${qp.goField}`;
      if (assertType === "string") {
        lines.push(`\t\tif v, ok := args["${qp.name}"].(string); ok && v != "" {`);
      } else {
        lines.push(`\t\tif v, ok := args["${qp.name}"].(${assertType}); ok {`);
      }
      emitConvertedValue(
        lines,
        "\t\t\t",
        convertedVar,
        "v",
        qp.typeInfo,
        `"${escapeDQ(`${qp.name} must be a valid value`)}"`,
      );
      if (qp.required) {
        lines.push(`\t\t\tparams.${qp.goField} = ${convertedVar}`);
      } else {
        lines.push(`\t\t\tparams.${qp.goField}.SetTo(${convertedVar})`);
      }
      lines.push(`\t\t}`);
    }
  }

  if (hasBody) {
    lines.push(`\t\tbody := &${alias}.${op.bodyTypeName}{}`);
    for (const field of bodyFields) {
      switch (field.kind) {
        case "scalar":
          emitScalarFieldExtraction(lines, field);
          break;
        case "scalarArray":
          emitScalarArrayFieldExtraction(lines, field);
          break;
        case "object":
          emitObjectFieldExtraction(lines, field, alias);
          break;
        case "objectArray":
          emitObjectArrayFieldExtraction(lines, field, alias);
          break;
      }
    }
  }

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
  const toolData = tools.map(({ httpOp, toolOpts }) => {
    const op = deriveOp(httpOp);
    const camel = snakeToCamel(toolOpts.name);
    const allParams = collectParams(program, httpOp, ALIAS);
    const pathParams = allParams.filter((p) => p.isPathParam);
    const queryParams = allParams.filter((p) => !p.isPathParam);
    const pathParamNames = new Set(pathParams.map((p) => p.name));
    const bodyFields = collectBodyFields(program, httpOp, ALIAS, pathParamNames);
    const hasBody = op.bodyTypeName !== null && bodyFields.length > 0;
    return { op, camel, toolOpts, pathParams, queryParams, bodyFields, hasBody };
  });

  const needsTimeImport = toolData.some(
    ({ pathParams, queryParams, bodyFields }) =>
      pathParams.some((p) => typeNeedsTime(p.typeInfo)) ||
      queryParams.some((p) => typeNeedsTime(p.typeInfo)) ||
      bodyFields.some((f) => bodyFieldNeedsTime(f)),
  );

  const lines: string[] = [];

  lines.push("// Code generated by typespec-mcp-go. DO NOT EDIT.");
  lines.push("");
  lines.push(`package ${PKG}`);
  lines.push("");
  lines.push("import (");
  lines.push('\t"context"');
  lines.push('\t"fmt"');
  if (needsTimeImport) {
    lines.push('\t"time"');
  }
  lines.push("");
  lines.push('\t"github.com/mark3labs/mcp-go/mcp"');
  lines.push('\tmcpserver "github.com/mark3labs/mcp-go/server"');
  lines.push("");
  lines.push(`\t${ALIAS} "${opts.ogenImportPath}"`);
  lines.push(")");
  lines.push("");

  lines.push("// Handlers is the interface MCP tool calls are dispatched through.");
  lines.push("type Handlers interface {");
  lines.push("\t// ConvertError maps a handler error to a user-facing message.");
  lines.push("\tConvertError(ctx context.Context, err error) string");
  for (const { op, hasBody } of toolData) {
    const parts: string[] = ["ctx context.Context"];
    if (hasBody) parts.push(`req *${ALIAS}.${op.bodyTypeName}`);
    if (op.hasParams) parts.push(`params ${ALIAS}.${op.paramsType}`);
    const returnType = op.responseTypeName ? `(*${ALIAS}.${op.responseTypeName}, error)` : "error";
    lines.push(`\t${op.methodName}(${parts.join(", ")}) ${returnType}`);
  }
  lines.push("}");
  lines.push("");

  lines.push("// RegisterTools registers all @mcpTool-annotated operations with the MCP server.");
  lines.push("func RegisterTools(s *mcpserver.MCPServer, h Handlers) {");
  for (const { camel } of toolData) {
    lines.push(`\ts.AddTool(${camel}Tool(), ${camel}Handler(h))`);
  }
  lines.push("}");
  lines.push("");

  lines.push(
    "func withRawProperty(name string, required bool, schema map[string]any) mcp.ToolOption {",
  );
  lines.push("\treturn func(t *mcp.Tool) {");
  lines.push("\t\tif required {");
  lines.push("\t\t\tt.InputSchema.Required = append(t.InputSchema.Required, name)");
  lines.push("\t\t}");
  lines.push("\t\tt.InputSchema.Properties[name] = schema");
  lines.push("\t}");
  lines.push("}");
  lines.push("");

  for (const td of toolData) {
    const { toolOpts, op, camel, pathParams, queryParams, bodyFields, hasBody } = td;

    lines.push(`// --- ${toolOpts.name} ---`);
    lines.push("");
    lines.push(`func ${camel}Tool() mcp.Tool {`);
    lines.push(`\treturn mcp.NewTool("${toolOpts.name}",`);
    lines.push(`\t\tmcp.WithDescription("${escapeDQ(toolOpts.description)}"),`);

    for (const p of [...pathParams, ...queryParams]) {
      lines.push(
        `\t\t${p.mcpType}("${p.name}"${requiredOpt(p.required)}${descOpt(p.description)}${enumOpt(p.typeInfo.enumValues)}),`,
      );
    }

    for (const field of bodyFields) {
      switch (field.kind) {
        case "scalar":
          if (!field.nullable) {
            lines.push(
              `\t\t${field.mcpType}("${field.name}"${requiredOpt(field.required)}${descOpt(field.description)}${enumOpt(field.typeInfo.enumValues)}),`,
            );
          } else {
            lines.push(
              `\t\twithRawProperty("${field.name}", ${String(field.required)}, map[string]any{`,
            );
            emitSimpleSchemaMap(lines, "\t\t\t", field.typeInfo, true, field.description);
            lines.push(`\t\t}),`);
          }
          break;
        case "scalarArray":
          lines.push(
            `\t\twithRawProperty("${field.name}", ${String(field.required)}, map[string]any{`,
          );
          lines.push(`\t\t\t"type": "array",`);
          if (field.description) {
            lines.push(`\t\t\t"description": "${escapeDQ(field.description)}",`);
          }
          lines.push(`\t\t\t"items": map[string]any{`);
          emitSimpleSchemaMap(lines, "\t\t\t\t", field.elementTypeInfo, false, "");
          lines.push(`\t\t\t},`);
          lines.push(`\t\t}),`);
          break;
        case "object":
          lines.push(
            `\t\twithRawProperty("${field.name}", ${String(field.required)}, map[string]any{`,
          );
          lines.push(
            field.nullable ? `\t\t\t"type": []any{"object", "null"},` : `\t\t\t"type": "object",`,
          );
          if (field.description) {
            lines.push(`\t\t\t"description": "${escapeDQ(field.description)}",`);
          }
          lines.push(`\t\t\t"properties": map[string]any{`);
          for (const member of field.members) {
            lines.push(`\t\t\t\t"${member.name}": map[string]any{`);
            emitSimpleSchemaMap(
              lines,
              "\t\t\t\t\t",
              member.typeInfo,
              member.nullable,
              member.description,
            );
            lines.push(`\t\t\t\t},`);
          }
          lines.push(`\t\t\t},`);
          {
            const requiredMembers = field.members
              .filter((m) => m.required)
              .map((m) => `"${m.name}"`);
            if (requiredMembers.length > 0) {
              lines.push(`\t\t\t"required": []string{${requiredMembers.join(", ")}},`);
            }
          }
          lines.push(`\t\t}),`);
          break;
        case "objectArray":
          lines.push(
            `\t\twithRawProperty("${field.name}", ${String(field.required)}, map[string]any{`,
          );
          lines.push(`\t\t\t"type": "array",`);
          if (field.description) {
            lines.push(`\t\t\t"description": "${escapeDQ(field.description)}",`);
          }
          lines.push(`\t\t\t"items": map[string]any{`);
          lines.push(`\t\t\t\t"type": "object",`);
          lines.push(`\t\t\t\t"properties": map[string]any{`);
          for (const member of field.elementFields) {
            lines.push(`\t\t\t\t\t"${member.name}": map[string]any{`);
            emitSimpleSchemaMap(
              lines,
              "\t\t\t\t\t\t",
              member.typeInfo,
              member.nullable,
              member.description,
            );
            lines.push(`\t\t\t\t\t},`);
          }
          lines.push(`\t\t\t\t},`);
          {
            const requiredMembers = field.elementFields
              .filter((m) => m.required)
              .map((m) => `"${m.name}"`);
            if (requiredMembers.length > 0) {
              lines.push(`\t\t\t\t"required": []string{${requiredMembers.join(", ")}},`);
            }
          }
          lines.push(`\t\t\t},`);
          lines.push(`\t\t}),`);
          break;
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
