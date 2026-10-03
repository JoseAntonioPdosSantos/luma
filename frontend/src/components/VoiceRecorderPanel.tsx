import { useTranslation } from "react-i18next";
import type { VoiceRecorder } from "../hooks/useVoiceRecorder";
import { MicIcon, StopIcon, TrashIcon } from "./Icons";

function formatElapsed(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
}

// Record / stop / listen / discard for the user's own take. The recording
// lives only in memory: the owner of the recorder drops it when the card changes.
export function VoiceRecorderPanel({ recorder }: { recorder: VoiceRecorder }) {
  const { t } = useTranslation();
  const { supported, status, error, audioUrl, elapsedMs, start, stop, discard } = recorder;

  if (!supported) {
    // Browsers only expose the microphone on https or localhost, so opening
    // the app by IP or hostname over plain http hides it.
    const insecure = typeof window !== "undefined" && !window.isSecureContext;
    return (
      <p className="hint voice-recorder-note">
        {insecure ? t("study.recordInsecure", { url: "http://localhost:5173" }) : t("study.recordUnsupported")}
      </p>
    );
  }

  return (
    <div className="voice-recorder">
      {status === "recording" ? (
        <button className="secondary voice-recorder-toggle recording" onClick={stop} aria-keyshortcuts="R">
          <StopIcon />
          {t("study.stopRecording")}
          <span className="voice-recorder-elapsed" role="timer">
            {formatElapsed(elapsedMs)}
          </span>
          <kbd className="key-hint" aria-hidden="true">R</kbd>
        </button>
      ) : (
        <button
          className="secondary voice-recorder-toggle"
          onClick={start}
          disabled={status === "requesting"}
          aria-keyshortcuts="R"
        >
          <MicIcon />
          {status === "recorded" ? t("study.recordAgain") : t("study.recordVoice")}
          <kbd className="key-hint" aria-hidden="true">R</kbd>
        </button>
      )}

      {status === "recorded" && audioUrl && (
        <div className="voice-recorder-take">
          <span className="hint">{t("study.yourRecording")}</span>
          <audio controls src={audioUrl} className="study-audio" aria-label={t("study.yourRecording")}>
            {t("study.audioUnsupported")}
          </audio>
          <button className="link-button voice-recorder-discard" onClick={discard}>
            <TrashIcon />
            {t("study.discardRecording")}
          </button>
        </div>
      )}

      {error && (
        <p className="error" role="alert">
          {error === "denied" ? t("study.recordDenied") : t("study.recordFailed")}
        </p>
      )}
    </div>
  );
}
