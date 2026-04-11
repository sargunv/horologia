import tailwindcss from "@tailwindcss/vite";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { env } from "node:process";
import { defineConfig } from "vite-plus";

const serverPort = env.SERVER_PORT ?? "8080";
const webPort = env.WEB_PORT ?? "5173";

export default defineConfig({
  plugins: [
    TanStackRouterVite({
      routesDirectory: "./src/routes",
      addExtensions: true,
      routeTreeFileHeader: ["/* eslint-disable */", "// noinspection JSUnusedGlobalSymbols"],
    }),
    react(),
    tailwindcss(),
  ],
  lint: {
    ignorePatterns: ["dist/**", "src/routeTree.gen.ts"],
    plugins: ["eslint", "typescript", "unicorn", "oxc", "import", "react", "promise", "vitest"],
    rules: {
      "import/no-duplicates": "error",
      "import/no-self-import": "error",
      "import/no-empty-named-blocks": "error",
      "import/first": "error",
      "import/no-mutable-exports": "error",
      "import/no-cycle": "error",
      "@typescript-eslint/consistent-type-assertions": ["error", { assertionStyle: "never" }],
      "typescript/no-explicit-any": "error",
      "typescript/no-unsafe-argument": "error",
      "typescript/no-unsafe-assignment": "error",
      "typescript/no-unsafe-call": "error",
      "typescript/no-unsafe-function-type": "error",
      "typescript/no-unsafe-member-access": "error",
      "typescript/no-unsafe-return": "error",
      "typescript/no-unsafe-type-assertion": "error",
    },
    options: {
      typeAware: true,
      typeCheck: true,
    },
  },
  server: {
    port: Number(webPort),
    strictPort: true,
    proxy: {
      "/api": {
        target: `http://localhost:${serverPort}`,
      },
      "/auth": {
        target: `http://localhost:${serverPort}`,
      },
      "/oauth": {
        target: `http://localhost:${serverPort}`,
      },
      "/.well-known": {
        target: `http://localhost:${serverPort}`,
      },
      "/mcp/.well-known": {
        target: `http://localhost:${serverPort}`,
      },
      "/healthz": {
        target: `http://localhost:${serverPort}`,
      },
    },
  },
});
