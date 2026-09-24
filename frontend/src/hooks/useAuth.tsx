import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import i18n, { SUPPORTED_LANGUAGES, type SupportedLanguage } from "../i18n";
import { login as loginRequest, logout as logoutRequest, me } from "../services/sessionService";
import type { User } from "../types";

function isSupportedLanguage(value: unknown): value is SupportedLanguage {
  return typeof value === "string" && (SUPPORTED_LANGUAGES as readonly string[]).includes(value);
}

// Applies the language saved on the account, if any; otherwise the
// browser-detected language (already active) is left alone.
function applyAccountLanguage(u: User) {
  if (isSupportedLanguage(u.language)) {
    void i18n.changeLanguage(u.language);
  }
}

type AuthStatus = "loading" | "authenticated" | "anonymous";

interface AuthContextValue {
  status: AuthStatus;
  user: User | null;
  login: (email: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    me()
      .then((u) => {
        setUser(u);
        applyAccountLanguage(u);
        setStatus("authenticated");
      })
      .catch(() => setStatus("anonymous"));
  }, []);

  const login = useCallback(async (email: string) => {
    const { user: loggedInUser } = await loginRequest(email);
    setUser(loggedInUser);
    applyAccountLanguage(loggedInUser);
    setStatus("authenticated");
  }, []);

  const logout = useCallback(async () => {
    await logoutRequest();
    setUser(null);
    setStatus("anonymous");
  }, []);

  return <AuthContext.Provider value={{ status, user, login, logout }}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
