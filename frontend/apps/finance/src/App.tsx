import { Route, Routes } from "react-router-dom";
import { AuthProvider, RequireAuth } from "@shared/lib/auth";
import { LoginPage } from "@shared/LoginPage";
import { FinanceScreen } from "./FinanceScreen";

export function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage appTitle="Finance Tracker" />} />
        <Route
          path="/*"
          element={
            <RequireAuth>
              <FinanceScreen />
            </RequireAuth>
          }
        />
      </Routes>
    </AuthProvider>
  );
}
