import type { ReactNode } from "react";
import { useAuth } from "./lib/auth";
import { APP_LABELS, siblingAppUrl, type AppId } from "./lib/siblingApps";
import { ClayButton } from "./design-system/ClayButton";

type AppFrameProps = {
  appId: AppId;
  title: string;
  children: ReactNode;
};

// Shared chrome for all 3 apps: same nav bar, same claymorphism shell.
// Unlike a single-SPA nav, the links to the other 2 apps are plain
// cross-origin <a> tags -- each app is its own deployed bundle on its
// own hostname, sharing nothing at runtime except the session cookie.
export function AppFrame({ appId, title, children }: AppFrameProps) {
  const { user, logout } = useAuth();
  const otherApps = (Object.keys(APP_LABELS) as AppId[]).filter((id) => id !== appId);

  return (
    <div className="clay-app" data-app={appId}>
      <nav className="clay-nav">
        <span className="clay-nav__brand">{title}</span>
        {otherApps.map((id) => (
          <a key={id} href={siblingAppUrl(id)}>
            {APP_LABELS[id]}
          </a>
        ))}
        <ClayButton variant="ghost" onClick={() => logout()}>
          Log out {user ? `(${user.username})` : ""}
        </ClayButton>
      </nav>
      <main className="clay-main">{children}</main>
    </div>
  );
}
