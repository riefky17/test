// Each app (Finance/Geochat/Fitness) is its own build, deployed to its
// own hostname in production (finance.yourdomain.com, etc. -- see
// deploy/haproxy/haproxy.cfg, which routes each hostname to a
// different backend). So "switching apps" from inside one of them
// means a real cross-origin link, not client-side routing.
//
// In local dev there's no subdomain to swap -- each app's Vite dev
// server just runs on its own port instead (see each app's
// vite.config.ts), so DEV_PORTS mirrors that.
export type AppId = "finance" | "geochat" | "fitness";

export const APP_LABELS: Record<AppId, string> = {
  finance: "Finance",
  geochat: "Geochat",
  fitness: "Fitness",
};

const DEV_PORTS: Record<AppId, number> = {
  finance: 5173,
  geochat: 5174,
  fitness: 5175,
};

export function siblingAppUrl(appId: AppId): string {
  const { protocol, hostname } = window.location;

  if (hostname === "localhost" || hostname === "127.0.0.1") {
    return `${protocol}//${hostname}:${DEV_PORTS[appId]}`;
  }

  const labels = hostname.split(".");
  labels[0] = appId;
  return `${protocol}//${labels.join(".")}`;
}
