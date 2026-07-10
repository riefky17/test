import { useEffect, useRef, useState } from "react";
import { AppShell } from "../../AppShell";
import { ClayCard } from "../../design-system/ClayCard";
import { ClayButton } from "../../design-system/ClayButton";
import { ClayInput } from "../../design-system/ClayInput";
import { api, wsURL } from "../../lib/api";
import { errorMessage } from "../../lib/auth";

type WsEvent = {
  kind: "message" | "location" | "sos";
  user_id?: number;
  username?: string;
  body?: string;
  latitude?: number;
  longitude?: number;
  created_at?: string;
};

export function GeochatApp() {
  const [events, setEvents] = useState<WsEvent[]>([]);
  const [draft, setDraft] = useState("");
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    api
      .get<WsEvent[]>("/api/geochat/messages")
      .then((history) =>
        setEvents(history.reverse().map((m) => ({ ...m, kind: "message" as const }))),
      )
      .catch((err) => setError(errorMessage(err)));

    const socket = new WebSocket(wsURL("/api/geochat/ws"));
    socketRef.current = socket;
    socket.onopen = () => setConnected(true);
    socket.onclose = () => setConnected(false);
    socket.onmessage = (msg) => {
      const event: WsEvent = JSON.parse(msg.data);
      setEvents((prev) => [...prev, event]);
    };

    return () => socket.close();
  }, []);

  function send(event: Partial<WsEvent>) {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify(event));
    }
  }

  function handleSend() {
    if (!draft.trim()) return;
    send({ kind: "message", body: draft });
    setDraft("");
  }

  function shareLocation() {
    if (!navigator.geolocation) {
      setError("Geolocation is not available in this browser");
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => send({ kind: "location", latitude: pos.coords.latitude, longitude: pos.coords.longitude }),
      (err) => setError(err.message),
    );
  }

  async function triggerSOS() {
    setError(null);
    const finish = (lat?: number, lng?: number) => {
      // Primary channel: the open websocket, if it's up.
      send({ kind: "sos", body: "I need help", latitude: lat, longitude: lng });
      // Fallback channel: plain REST call, which the geochat-svc
      // backend fans out to Telegram independent of the websocket --
      // this is what still works if the socket (or the browser tab)
      // is the thing that's broken.
      api
        .post<{ telegram_ok: boolean }>("/api/geochat/sos", { message: "I need help", latitude: lat, longitude: lng })
        .catch((err) => setError(errorMessage(err)));
    };

    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(
        (pos) => finish(pos.coords.latitude, pos.coords.longitude),
        () => finish(),
      );
    } else {
      finish();
    }
  }

  return (
    <AppShell appId="geochat" title="Geochat">
      <ClayCard flat>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <span style={{ color: connected ? "var(--clay-accent)" : "var(--clay-text-muted)" }}>
            {connected ? "● connected" : "○ reconnecting…"}
          </span>
          <ClayButton className="clay-sos-button" onClick={triggerSOS}>
            🆘 SOS
          </ClayButton>
        </div>
      </ClayCard>

      <ClayCard>
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem", maxHeight: 360, overflowY: "auto" }}>
          {events.map((e, i) => (
            <div key={i}>
              {e.kind === "message" && (
                <div>
                  <strong>{e.username}:</strong> {e.body}
                </div>
              )}
              {e.kind === "location" && (
                <div style={{ color: "var(--clay-text-muted)" }}>
                  📍 {e.username} shared location ({e.latitude?.toFixed(4)}, {e.longitude?.toFixed(4)})
                </div>
              )}
              {e.kind === "sos" && (
                <div style={{ color: "var(--clay-danger)", fontWeight: 700 }}>
                  🆘 {e.username}: {e.body}
                </div>
              )}
            </div>
          ))}
          {events.length === 0 && <p style={{ color: "var(--clay-text-muted)" }}>No messages yet.</p>}
        </div>
      </ClayCard>

      <ClayCard flat>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <ClayInput
            placeholder="Message"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSend()}
          />
          <ClayButton onClick={handleSend}>Send</ClayButton>
          <ClayButton variant="ghost" onClick={shareLocation}>
            📍
          </ClayButton>
        </div>
        {error && <div style={{ color: "var(--clay-danger)", marginTop: "0.5rem" }}>{error}</div>}
      </ClayCard>
    </AppShell>
  );
}
