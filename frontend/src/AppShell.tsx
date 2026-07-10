import { NavLink } from "react-router-dom";
import type { ReactNode } from "react";
import { useAuth } from "./lib/auth";
import { ClayButton } from "./design-system/ClayButton";

type AppShellProps = {
  appId: "finance" | "geochat" | "fitness";
  title: string;
  children: ReactNode;
};

// One shared shell for all 3 apps: same nav chrome, each app just sets
// data-app on <body> so the CSS variables in clay.css pick the right
// accent color.
export function AppShell({ appId, title, children }: AppShellProps) {
  const { user, logout } = useAuth();

  return (
    <div className="clay-app" data-app={appId}>
      <nav className="clay-nav">
        <span className="clay-nav__brand">{title}</span>
        <NavLink to="/finance" className={({ isActive }) => (isActive ? "active" : "")}>
          Finance
        </NavLink>
        <NavLink to="/geochat" className={({ isActive }) => (isActive ? "active" : "")}>
          Geochat
        </NavLink>
        <NavLink to="/fitness" className={({ isActive }) => (isActive ? "active" : "")}>
          Fitness
        </NavLink>
        <ClayButton variant="ghost" onClick={() => logout()}>
          Log out {user ? `(${user.username})` : ""}
        </ClayButton>
      </nav>
      <main className="clay-main">{children}</main>
    </div>
  );
}
