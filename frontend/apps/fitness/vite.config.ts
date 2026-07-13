import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// Fitness Tracker is its own deployed bundle -- served directly by
// fitness-svc on its own hostname in production. This dev server only
// exists for local work.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@shared": path.resolve(__dirname, "../../shared"),
    },
  },
  server: {
    port: 5175,
    strictPort: true,
    proxy: {
      "/api": "http://127.0.0.1:3003",
    },
  },
  build: {
    outDir: "dist",
  },
});
