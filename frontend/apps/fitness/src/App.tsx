import { Route, Routes } from "react-router-dom";
import { AuthProvider, RequireAuth } from "@shared/lib/auth";
import { LoginPage } from "@shared/LoginPage";
import { FitnessScreen } from "./FitnessScreen";

export function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage appTitle="Fitness Tracker" />} />
        <Route
          path="/*"
          element={
            <RequireAuth>
              <FitnessScreen />
            </RequireAuth>
          }
        />
      </Routes>
    </AuthProvider>
  );
}
