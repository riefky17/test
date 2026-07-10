import { Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider, RequireAuth } from "./lib/auth";
import { resolveAppFromHostname } from "./lib/appHost";
import { LoginPage } from "./LoginPage";
import { FinanceApp } from "./apps/finance/FinanceApp";
import { GeochatApp } from "./apps/geochat/GeochatApp";
import { FitnessApp } from "./apps/fitness/FitnessApp";

export function App() {
  const homeApp = resolveAppFromHostname(window.location.hostname);

  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/finance/*"
          element={
            <RequireAuth>
              <FinanceApp />
            </RequireAuth>
          }
        />
        <Route
          path="/geochat/*"
          element={
            <RequireAuth>
              <GeochatApp />
            </RequireAuth>
          }
        />
        <Route
          path="/fitness/*"
          element={
            <RequireAuth>
              <FitnessApp />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to={`/${homeApp}`} replace />} />
      </Routes>
    </AuthProvider>
  );
}
