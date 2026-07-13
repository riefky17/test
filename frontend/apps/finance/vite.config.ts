import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// Finance Tracker is its own deployed bundle -- built as static files
// and served directly by finance-svc (see internal/spa), on its own
// hostname in production. This dev server only exists for local work.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@shared": path.resolve(__dirname, "../../shared"),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": "http://127.0.0.1:3001",
    },
  },
  build: {
    outDir: "dist",
  },
});
