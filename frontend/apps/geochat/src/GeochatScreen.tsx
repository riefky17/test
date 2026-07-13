import { useEffect, useRef, useState } from "react";
import { AppFrame } from "@shared/AppFrame";
import { ClayCard } from "@shared/design-system/ClayCard";
import { ClayButton } from "@shared/design-system/ClayButton";
import { ClayInput } from "@shared/design-system/ClayInput";
import { api, wsURL } from "@shared/lib/api";
import { errorMessage } from "@shared/lib/auth";

type WsEvent = {
  kind: "message" | "location" | "sos";
  user_id?: number;
  username?: string;
  body?: string;
  latitude?: number;
  longitude?: number;
  created_at?: string;
};

type LocationPermission = "unknown" | "prompt" | "granted" | "denied" | "unsupported";

// How often a live-shared position is re-sent over the websocket.
// watchPosition fires on every GPS update, which is far more often than
// this app needs -- 3 family members, not a fleet tracker.
const LOCATION_SHARE_INTERVAL_MS = 20_000;

export function GeochatScreen() {
  const [events, setEvents] = useState<WsEvent[]>([]);
  const [draft, setDraft] = useState("");
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sosStatus, setSosStatus] = useState<string | null>(null);
  const [locationPermission, setLocationPermission] = useState<LocationPermission>("unknown");
  const [sharingLocation, setSharingLocation] = useState(false);

  const socketRef = useRef<WebSocket | null>(null);
  const watchIdRef = useRef<number | null>(null);
  const lastPositionRef = useRef<{ latitude: number; longitude: number } | null>(null);
  const lastSentAtRef = useRef(0);

  useEffect(() => {
    api
      .get<WsEvent[]>("/api/geochat/messages")
      .then((history) => setEvents(history.reverse().map((m) => ({ ...m, kind: "message" as const }))))
      .catch((err) => setError(errorMessage(err)));

    const socket = new WebSocket(wsURL("/api/geochat/ws"));
    socketRef.current = socket;
    socket.onopen = () => setConnected(true);
    socket.onclose = () => setConnected(false);
    socket.onmessage = (msg) => {
      const event: WsEvent = JSON.parse(msg.data);
      setEvents((prev) => [...prev, event]);
    };

    if (!navigator.geolocation) {
      setLocationPermission("unsupported");
    } else if (navigator.permissions?.query) {
      navigator.permissions
        .query({ name: "geolocation" as PermissionName })
        .then((status) => {
          setLocationPermission(status.state as LocationPermission);
          status.onchange = () => setLocationPermission(status.state as LocationPermission);
        })
        .catch(() => setLocationPermission("prompt"));
    } else {
      setLocationPermission("prompt");
    }

    return () => {
      socket.close();
      if (watchIdRef.current !== null) navigator.geolocation.clearWatch(watchIdRef.current);
    };
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

  // Requesting the browser's own Geolocation permission is the prompt
  // the user actually sees -- this app has no separate consent flow of
  // its own, and no data ever leaves this web messenger to a third
  // party. Once granted, watchPosition keeps a "living" position so an
  // SOS trigger can always attach the freshest coordinates without
  // waiting on a fresh permission round-trip.
  function enableLocationSharing() {
    if (!navigator.geolocation) {
      setLocationPermission("unsupported");
      return;
    }

    watchIdRef.current = navigator.geolocation.watchPosition(
      (pos) => {
        setLocationPermission("granted");
        setSharingLocation(true);
        lastPositionRef.current = { latitude: pos.coords.latitude, longitude: pos.coords.longitude };

        const now = Date.now();
        if (now - lastSentAtRef.current >= LOCATION_SHARE_INTERVAL_MS) {
          lastSentAtRef.current = now;
          send({ kind: "location", latitude: pos.coords.latitude, longitude: pos.coords.longitude });
        }
      },
      (err) => {
        setLocationPermission(err.code === err.PERMISSION_DENIED ? "denied" : "prompt");
        setSharingLocation(false);
        setError(err.message);
      },
      { enableHighAccuracy: false, maximumAge: 15_000 },
    );
  }

  function disableLocationSharing() {
    if (watchIdRef.current !== null) {
      navigator.geolocation.clearWatch(watchIdRef.current);
      watchIdRef.current = null;
    }
    setSharingLocation(false);
  }

  async function triggerSOS() {
    setError(null);
    setSosStatus(null);
    const pos = lastPositionRef.current;

    // Primary and only delivery path: broadcast over the open
    // websocket to this same web messenger -- there's no separate bot
    // or third-party service in the loop.
    send({ kind: "sos", body: "I need help", latitude: pos?.latitude, longitude: pos?.longitude });

    try {
      const res = await api.post<{ recipients_online: number }>("/api/geochat/sos", {
        message: "I need help",
        latitude: pos?.latitude,
        longitude: pos?.longitude,
      });
      setSosStatus(
        res.recipients_online > 0
          ? `Sent — ${res.recipients_online} other family member${res.recipients_online > 1 ? "s" : ""} online right now.`
          : "Sent and logged, but no one else was online at this moment.",
      );
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <AppFrame appId="geochat" title="Geochat">
      <ClayCard flat>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <span style={{ color: connected ? "var(--clay-accent)" : "var(--clay-text-muted)" }}>
            {connected ? "● connected" : "○ reconnecting…"}
          </span>
          <ClayButton className="clay-sos-button" onClick={triggerSOS}>
            🆘 SOS
          </ClayButton>
        </div>
        {sosStatus && <div style={{ marginTop: "0.5rem", fontSize: "0.9rem" }}>{sosStatus}</div>}
      </ClayCard>

      <ClayCard flat>
        {locationPermission === "granted" && sharingLocation && (
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <span>📍 Sharing your live location with the family</span>
            <ClayButton variant="ghost" onClick={disableLocationSharing}>
              Stop
            </ClayButton>
          </div>
        )}
        {locationPermission !== "granted" && locationPermission !== "unsupported" && (
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: "0.75rem" }}>
            <span style={{ color: "var(--clay-text-muted)" }}>
              {locationPermission === "denied"
                ? "Location permission was denied — enable it in your browser's site settings to share location."
                : "Share your location so the family can see where you are, and so SOS can attach your position."}
            </span>
            {locationPermission !== "denied" && (
              <ClayButton onClick={enableLocationSharing}>Enable location</ClayButton>
            )}
          </div>
        )}
        {(locationPermission === "granted" && !sharingLocation) && (
          <ClayButton onClick={enableLocationSharing}>Enable location</ClayButton>
        )}
        {locationPermission === "unsupported" && (
          <span style={{ color: "var(--clay-text-muted)" }}>This browser doesn't support location sharing.</span>
        )}
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
        </div>
        {error && <div style={{ color: "var(--clay-danger)", marginTop: "0.5rem" }}>{error}</div>}
      </ClayCard>
    </AppFrame>
  );
}
