import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Built as static files and served directly by HAProxy (see
// deploy/haproxy/haproxy.cfg) -- no Node process needed at runtime.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // Dev-only convenience: routes each app's /api calls to its own
      // service port so `npm run dev` works without HAProxy in front.
      "/api/finance": "http://127.0.0.1:3001",
      "/api/geochat": {
        target: "http://127.0.0.1:3002",
        ws: true,
      },
      "/api/fitness": "http://127.0.0.1:3003",
    },
  },
  build: {
    outDir: "dist",
  },
});
