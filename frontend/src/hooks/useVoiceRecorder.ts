import { useCallback, useEffect, useRef, useState } from "react";

export type RecorderStatus = "idle" | "requesting" | "recording" | "recorded";
export type RecorderError = "unsupported" | "denied" | "failed";

// A take is only meant to compare with the card's audio, so it is capped to
// keep a forgotten recording from growing in memory.
export const MAX_RECORDING_MS = 60_000;

export interface VoiceRecorder {
  supported: boolean;
  status: RecorderStatus;
  error: RecorderError | null;
  audioUrl: string | null;
  elapsedMs: number;
  start: () => void;
  stop: () => void;
  discard: () => void;
}

function isSupported(): boolean {
  return (
    typeof navigator !== "undefined" &&
    typeof navigator.mediaDevices?.getUserMedia === "function" &&
    typeof MediaRecorder !== "undefined"
  );
}

// Records the microphone into memory only (a Blob behind an object URL).
// Whenever resetKey changes — e.g. the next card comes up — the take is
// dropped and the microphone released, so a recording never outlives its card.
export function useVoiceRecorder(resetKey: unknown): VoiceRecorder {
  const supported = isSupported();
  const [status, setStatus] = useState<RecorderStatus>("idle");
  const [error, setError] = useState<RecorderError | null>(null);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [elapsedMs, setElapsedMs] = useState(0);

  const recorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const urlRef = useRef<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // Bumped on every reset: async callbacks from an older take compare it to
  // their own value and give up, instead of writing into the new card.
  const generationRef = useRef(0);

  const releaseMicrophone = useCallback(() => {
    if (timerRef.current !== null) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
  }, []);

  const replaceUrl = useCallback((url: string | null) => {
    if (urlRef.current) URL.revokeObjectURL(urlRef.current);
    urlRef.current = url;
    setAudioUrl(url);
  }, []);

  const reset = useCallback(() => {
    generationRef.current += 1;
    const recorder = recorderRef.current;
    recorderRef.current = null;
    if (recorder && recorder.state !== "inactive") recorder.stop();
    releaseMicrophone();
    replaceUrl(null);
    setStatus("idle");
    setError(null);
    setElapsedMs(0);
  }, [releaseMicrophone, replaceUrl]);

  useEffect(() => reset, [resetKey, reset]);

  const stop = useCallback(() => {
    const recorder = recorderRef.current;
    if (recorder && recorder.state !== "inactive") recorder.stop();
  }, []);

  const start = useCallback(async () => {
    if (!supported) {
      setError("unsupported");
      return;
    }
    if (status === "requesting" || status === "recording") return;

    const generation = generationRef.current;
    setError(null);
    setStatus("requesting");

    let stream: MediaStream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch (err) {
      if (generation !== generationRef.current) return;
      const denied = err instanceof DOMException && (err.name === "NotAllowedError" || err.name === "SecurityError");
      setError(denied ? "denied" : "failed");
      setStatus(urlRef.current ? "recorded" : "idle");
      return;
    }
    // The card changed while the browser was asking for permission.
    if (generation !== generationRef.current) {
      stream.getTracks().forEach((track) => track.stop());
      return;
    }

    const recorder = new MediaRecorder(stream);
    const chunks: Blob[] = [];
    recorder.ondataavailable = (e) => {
      if (e.data.size > 0) chunks.push(e.data);
    };
    recorder.onstop = () => {
      if (generation !== generationRef.current) return;
      releaseMicrophone();
      recorderRef.current = null;
      const blob = new Blob(chunks, { type: recorder.mimeType || "audio/webm" });
      replaceUrl(URL.createObjectURL(blob));
      setStatus("recorded");
    };

    streamRef.current = stream;
    recorderRef.current = recorder;
    recorder.start();
    setStatus("recording");

    const startedAt = Date.now();
    setElapsedMs(0);
    timerRef.current = setInterval(() => {
      const elapsed = Date.now() - startedAt;
      setElapsedMs(elapsed);
      if (elapsed >= MAX_RECORDING_MS) stop();
    }, 250);
  }, [supported, status, releaseMicrophone, replaceUrl, stop]);

  const discard = useCallback(() => {
    if (status === "recorded") reset();
  }, [status, reset]);

  return { supported, status, error, audioUrl, elapsedMs, start, stop, discard };
}
