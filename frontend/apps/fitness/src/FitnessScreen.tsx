import { useEffect, useState, type FormEvent } from "react";
import { AppFrame } from "@shared/AppFrame";
import { ClayCard } from "@shared/design-system/ClayCard";
import { ClayButton } from "@shared/design-system/ClayButton";
import { ClayInput } from "@shared/design-system/ClayInput";
import { api } from "@shared/lib/api";
import { errorMessage } from "@shared/lib/auth";

type Plan = { id: number; name: string; style: string; created_at: string };
type Session = { id: number; plan_id: number | null; notes: string; performed_at: string };

const STYLE_LABELS: Record<string, string> = {
  general: "General",
  "ppl-evidence-based": "PPL, evidence-based",
  "posterior-chain-focus": "Posterior-chain focus",
};

export function FitnessScreen() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [name, setName] = useState("");
  const [style, setStyle] = useState("general");
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    const [p, s] = await Promise.all([
      api.get<Plan[]>("/api/fitness/plans"),
      api.get<Session[]>("/api/fitness/sessions"),
    ]);
    setPlans(p);
    setSessions(s);
  }

  useEffect(() => {
    refresh().catch((err) => setError(errorMessage(err)));
  }, []);

  async function handleCreatePlan(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (!name.trim()) {
      setError("Plan name is required");
      return;
    }
    try {
      await api.post("/api/fitness/plans", { name, style, exercises: [] });
      setName("");
      await refresh();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function handleDeletePlan(id: number) {
    try {
      await api.delete(`/api/fitness/plans/${id}`);
      await refresh();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function handleLogSession(planId: number | null) {
    try {
      await api.post("/api/fitness/sessions", { plan_id: planId, notes: "" });
      await refresh();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <AppFrame appId="fitness" title="Fitness Tracker">
      <ClayCard flat>
        <h3 style={{ marginTop: 0 }}>New plan</h3>
        <p style={{ color: "var(--clay-text-muted)", marginTop: 0 }}>
          Styles here follow general training principles (progressive overload, push/pull/legs splits,
          posterior-chain hypertrophy work) -- not a transcription of any specific paid program.
        </p>
        <form onSubmit={handleCreatePlan} style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
          <ClayInput placeholder="Plan name" value={name} onChange={(e) => setName(e.target.value)} />
          <select className="clay-input" value={style} onChange={(e) => setStyle(e.target.value)}>
            {Object.entries(STYLE_LABELS).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
          {error && <div style={{ color: "var(--clay-danger)" }}>{error}</div>}
          <ClayButton type="submit">Create plan</ClayButton>
        </form>
      </ClayCard>

      <ClayCard>
        <h3 style={{ marginTop: 0 }}>Your plans</h3>
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          {plans.map((p) => (
            <div key={p.id} style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <div>
                <strong>{p.name}</strong>{" "}
                <span style={{ color: "var(--clay-text-muted)" }}>({STYLE_LABELS[p.style] ?? p.style})</span>
              </div>
              <div style={{ display: "flex", gap: "0.5rem" }}>
                <ClayButton variant="ghost" onClick={() => handleLogSession(p.id)}>
                  Log session
                </ClayButton>
                <ClayButton variant="danger" onClick={() => handleDeletePlan(p.id)}>
                  Delete
                </ClayButton>
              </div>
            </div>
          ))}
          {plans.length === 0 && <p style={{ color: "var(--clay-text-muted)" }}>No plans yet.</p>}
        </div>
      </ClayCard>

      <ClayCard>
        <h3 style={{ marginTop: 0 }}>Recent sessions</h3>
        <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
          {sessions.map((s) => (
            <div key={s.id} style={{ color: "var(--clay-text-muted)" }}>
              {new Date(s.performed_at).toLocaleString()}
              {s.plan_id ? ` — plan #${s.plan_id}` : ""}
            </div>
          ))}
          {sessions.length === 0 && <p style={{ color: "var(--clay-text-muted)" }}>No sessions logged yet.</p>}
        </div>
      </ClayCard>
    </AppFrame>
  );
}
