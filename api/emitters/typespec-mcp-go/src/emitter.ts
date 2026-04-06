import { type EmitContext } from "@typespec/compiler";
import { getAllHttpServices } from "@typespec/http";
import { getMcpTool } from "./decorator.js";
import { generateGoFile, type ToolInfo, type EmitterOptions } from "./codegen.js";
import * as path from "path";
import * as fs from "fs/promises";

export async function $onEmit(context: EmitContext): Promise<void> {
  const { program } = context;
  const rawOpts = context.options as Record<string, unknown>;

  const ogenImportPath = rawOpts["ogenImportPath"];
  if (typeof ogenImportPath !== "string" || !ogenImportPath) {
    program.reportDiagnostic({
      code: "typespec-mcp-go/missing-option",
      severity: "error",
      message: `@tend/typespec-mcp-go requires the "ogenImportPath" option (Go import path for ogen-generated package).`,
      target: { kind: "NoTarget" } as any,
    });
    return;
  }

  const opts: EmitterOptions = { ogenImportPath };

  // Collect all operations with @mcpTool
  const tools: ToolInfo[] = [];

  const [services] = getAllHttpServices(program);
  for (const service of services) {
    for (const httpOp of service.operations) {
      const toolOpts = getMcpTool(program, httpOp.operation);
      if (toolOpts) {
        tools.push({ op: httpOp.operation, httpOp, toolOpts });
      }
    }
  }

  if (tools.length === 0) return;

  const outputDir = context.emitterOutputDir;
  await fs.mkdir(outputDir, { recursive: true });

  const content = generateGoFile(program, tools, opts);
  await fs.writeFile(path.join(outputDir, "tools_gen.go"), content);
}
