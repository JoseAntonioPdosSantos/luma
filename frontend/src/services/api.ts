// Falls back to the page's own hostname (with the backend's port) instead
// of a fixed "localhost", so the same build works whether it's opened as
// localhost or from another device on the LAN via the host's IP.
const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? `${window.location.protocol}//${window.location.hostname}:8080`;

export class ApiError extends Error {
  code: string;
  status: number;
  // Stable, machine-readable identifier for exactly which validation or
  // lookup failed (e.g. "deck.name.tooLong"), meant for translating the
  // message shown to the user. "" for a response the backend didn't send
  // (a network failure, or one with no recognizable envelope).
  key: string;

  constructor(code: string, message: string, status: number, key: string = "") {
    super(message);
    this.code = code;
    this.status = status;
    this.key = key;
  }
}

interface ErrorEnvelope {
  error: { code: string; key?: string; message: string; requestId: string };
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const isJson = response.headers.get("content-type")?.includes("application/json");
  const body = isJson ? await response.json() : undefined;

  if (!response.ok) {
    const envelope = body as ErrorEnvelope | undefined;
    throw new ApiError(
      envelope?.error?.code ?? "UNKNOWN_ERROR",
      envelope?.error?.message ?? `Request failed with status ${response.status}`,
      response.status,
      envelope?.error?.key ?? "",
    );
  }

  return body as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PUT", body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PATCH", body: body ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
  // postFile bypasses the default JSON content type: the body is the raw
  // file, sent with its own content type (e.g. "audio/mpeg").
  postFile: <T>(path: string, contentType: string, file: Blob) =>
    request<T>(path, { method: "POST", body: file, headers: { "Content-Type": contentType } }),
};

// The backend returns audio URLs as API-relative paths (e.g.
// "/api/v1/flashcards/{id}/audio"). Since the frontend is typically
// served from a different origin than the API, an <audio>/<img> element
// needs the full URL, not a path resolved against the frontend's own
// origin.
export function apiUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}
