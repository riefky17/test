import { Route, Routes } from "react-router-dom";
import { AuthProvider, RequireAuth } from "@shared/lib/auth";
import { LoginPage } from "@shared/LoginPage";
import { GeochatScreen } from "./GeochatScreen";

export function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage appTitle="Geochat" />} />
        <Route
          path="/*"
          element={
            <RequireAuth>
              <GeochatScreen />
            </RequireAuth>
          }
        />
      </Routes>
    </AuthProvider>
  );
}
