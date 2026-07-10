// In production HAProxy sends finance.yourdomain.com / geochat.yourdomain.com
// / fitness.yourdomain.com to 3 separate backends, but all 3 serve the
// same static SPA build. So when someone lands on "/" we still need to
// know which app to show -- inferred from the hostname first (subdomain
// prefix), falling back to Finance for local dev (localhost, IP, etc.)
// where there's only one hostname to work with.
const APP_IDS = ["finance", "geochat", "fitness"] as const;
export type AppId = (typeof APP_IDS)[number];

export function resolveAppFromHostname(hostname: string): AppId {
  const subdomain = hostname.split(".")[0];
  return (APP_IDS as readonly string[]).includes(subdomain) ? (subdomain as AppId) : "finance";
}
