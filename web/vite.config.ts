import tailwindcss from "@tailwindcss/vite";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite-plus";

export default defineConfig({
  plugins: [TanStackRouterVite({ routesDirectory: "./src/routes" }), react(), tailwindcss()],
  lint: {
    ignorePatterns: ["dist/**", "src/routeTree.gen.ts"],
    plugins: ["import"],
    rules: {
      "import/no-duplicates": "error",
      "import/no-self-import": "error",
      "import/no-empty-named-blocks": "error",
      "import/first": "error",
      "import/no-mutable-exports": "error",
      "import/no-cycle": "error",
    },
    options: {
      typeAware: true,
      typeCheck: true,
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        rewrite: (path) => path.replace(/^\/api/, ""),
      },
    },
  },
});
