import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// Geochat is its own deployed bundle -- served directly by geochat-svc
// on its own hostname in production. This dev server only exists for
// local work.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@shared": path.resolve(__dirname, "../../shared"),
    },
  },
  server: {
    port: 5174,
    strictPort: true,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:3002",
        ws: true,
      },
    },
  },
  build: {
    outDir: "dist",
  },
});
