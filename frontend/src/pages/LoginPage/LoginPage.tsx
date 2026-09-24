import { useState, type FormEvent } from "react";
import { Navigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "../../hooks/useAuth";
import { useApiErrorMessage } from "../../i18n/useApiErrorMessage";

export function LoginPage() {
  const { status, login } = useAuth();
  const { t } = useTranslation();
  const apiErrorMessage = useApiErrorMessage();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  if (status === "authenticated") {
    return <Navigate to="/dashboard" replace />;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await login(email);
    } catch (err) {
      setError(apiErrorMessage(err, "login.genericError"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="app-shell">
      <img src="/logo-light.png" alt="Luma" className="brand-logo brand-logo-light brand-logo-hero" />
      <img src="/logo-dark.png" alt="Luma" className="brand-logo brand-logo-dark brand-logo-hero" />
      <p>{t("login.tagline")}</p>
      <p className="hint">{t("login.hint")}</p>

      <form onSubmit={handleSubmit} className="form">
        <label htmlFor="email">{t("login.emailLabel")}</label>
        <input
          id="email"
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={submitting}
          placeholder={t("login.emailPlaceholder")}
        />
        {error && <p className="error" role="alert">{error}</p>}
        <button type="submit" disabled={submitting || email.trim() === ""}>
          {submitting ? t("login.continuing") : t("login.continueButton")}
        </button>
      </form>
    </main>
  );
}
