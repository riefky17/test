import { useEffect, useState, type FormEvent } from "react";
import { AppFrame } from "@shared/AppFrame";
import { ClayCard } from "@shared/design-system/ClayCard";
import { ClayButton } from "@shared/design-system/ClayButton";
import { ClayInput } from "@shared/design-system/ClayInput";
import { api } from "@shared/lib/api";
import { errorMessage } from "@shared/lib/auth";
import { formatIDR } from "@shared/lib/currency";

type Category = { id: number; name: string; kind: "income" | "expense"; icon: string };
type Transaction = {
  id: number;
  category_id: number;
  category_name: string;
  amount_idr: number;
  note: string;
  occurred_at: string;
};
type Summary = { income_idr: number; expense_idr: number; balance_idr: number };

export function FinanceScreen() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [categoryId, setCategoryId] = useState<number | "">("");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    const [cats, txs, sum] = await Promise.all([
      api.get<Category[]>("/api/finance/categories"),
      api.get<Transaction[]>("/api/finance/transactions"),
      api.get<Summary>("/api/finance/summary"),
    ]);
    setCategories(cats);
    setTransactions(txs);
    setSummary(sum);
    if (categoryId === "" && cats.length > 0) setCategoryId(cats[0].id);
  }

  useEffect(() => {
    refresh().catch((err) => setError(errorMessage(err)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleAdd(e: FormEvent) {
    e.preventDefault();
    setError(null);
    const parsed = Number(amount);
    if (!categoryId || !parsed) {
      setError("Pick a category and enter a non-zero amount");
      return;
    }
    const category = categories.find((c) => c.id === categoryId);
    const signedAmount = category?.kind === "expense" ? -Math.abs(parsed) : Math.abs(parsed);

    try {
      await api.post("/api/finance/transactions", {
        category_id: categoryId,
        amount_idr: signedAmount,
        note,
      });
      setAmount("");
      setNote("");
      await refresh();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function handleDelete(id: number) {
    try {
      await api.delete(`/api/finance/transactions/${id}`);
      await refresh();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <AppFrame appId="finance" title="Finance Tracker">
      {summary && (
        <ClayCard>
          <h2 style={{ marginTop: 0 }}>Balance: {formatIDR(summary.balance_idr)}</h2>
          <div style={{ display: "flex", gap: "1.5rem", color: "var(--clay-text-muted)" }}>
            <span>Income: {formatIDR(summary.income_idr)}</span>
            <span>Expense: {formatIDR(summary.expense_idr)}</span>
          </div>
        </ClayCard>
      )}

      <ClayCard flat>
        <h3 style={{ marginTop: 0 }}>Add transaction</h3>
        <form onSubmit={handleAdd} style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
          <select
            className="clay-input"
            value={categoryId}
            onChange={(e) => setCategoryId(Number(e.target.value))}
          >
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.icon} {c.name} ({c.kind})
              </option>
            ))}
          </select>
          <ClayInput
            type="number"
            placeholder="Amount (IDR)"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <ClayInput placeholder="Note (optional)" value={note} onChange={(e) => setNote(e.target.value)} />
          {error && <div style={{ color: "var(--clay-danger)" }}>{error}</div>}
          <ClayButton type="submit">Add</ClayButton>
        </form>
      </ClayCard>

      <ClayCard>
        <h3 style={{ marginTop: 0 }}>Recent transactions</h3>
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          {transactions.map((t) => (
            <div
              key={t.id}
              style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}
            >
              <div>
                <strong>{t.category_name}</strong>
                {t.note && <span style={{ color: "var(--clay-text-muted)" }}> — {t.note}</span>}
              </div>
              <div style={{ display: "flex", gap: "0.75rem", alignItems: "center" }}>
                <span style={{ color: t.amount_idr < 0 ? "var(--clay-danger)" : "inherit" }}>
                  {formatIDR(t.amount_idr)}
                </span>
                <ClayButton variant="ghost" onClick={() => handleDelete(t.id)}>
                  ✕
                </ClayButton>
              </div>
            </div>
          ))}
          {transactions.length === 0 && <p style={{ color: "var(--clay-text-muted)" }}>No transactions yet.</p>}
        </div>
      </ClayCard>
    </AppFrame>
  );
}
