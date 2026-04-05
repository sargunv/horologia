import { type EmitContext } from "@typespec/compiler";
import { getAllHttpServices } from "@typespec/http";
import { getMcpTool } from "./decorator.js";
import { generateGoFile, type ToolInfo } from "./codegen.js";
import * as path from "path";
import * as fs from "fs/promises";

export async function $onEmit(context: EmitContext): Promise<void> {
  const { program } = context;

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

  const content = generateGoFile(program, tools);
  await fs.writeFile(path.join(outputDir, "tools_gen.go"), content);
}
