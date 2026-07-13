import { useState, type FormEvent } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { ClayCard } from "./design-system/ClayCard";
import { ClayInput } from "./design-system/ClayInput";
import { ClayButton } from "./design-system/ClayButton";
import { useAuth, isApiError } from "./lib/auth";

type LoginPageProps = {
  appTitle: string;
};

// Each app has its own login page at its own hostname (there's no
// central login domain), but they all authenticate against the same
// auth.users table and set the same cross-subdomain session cookie --
// so logging in on one app also logs you into the other two.
export function LoginPage({ appTitle }: LoginPageProps) {
  const { user, login } = useAuth();
  const location = useLocation();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  if (user) {
    const from = (location.state as { from?: Location })?.from?.pathname ?? "/";
    return <Navigate to={from} replace />;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await login(username, password);
    } catch (err) {
      setError(isApiError(err) ? err.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="clay-main" style={{ justifyContent: "center", minHeight: "100vh" }}>
      <ClayCard>
        <h1 style={{ marginTop: 0 }}>{appTitle}</h1>
        <p style={{ color: "var(--clay-text-muted)" }}>Papa · Mami · Echa</p>
        <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
          <ClayInput
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            required
          />
          <ClayInput
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
          {error && <div style={{ color: "var(--clay-danger)" }}>{error}</div>}
          <ClayButton type="submit" disabled={submitting}>
            {submitting ? "Signing in…" : "Sign in"}
          </ClayButton>
        </form>
      </ClayCard>
    </div>
  );
}
