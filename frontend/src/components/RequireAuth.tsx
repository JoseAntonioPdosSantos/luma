import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "../hooks/useAuth";

export function RequireAuth({ children }: { children: ReactNode }) {
  const { status } = useAuth();
  const { t } = useTranslation();

  if (status === "loading") {
    return <p className="loading">{t("requireAuth.loading")}</p>;
  }
  if (status === "anonymous") {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
