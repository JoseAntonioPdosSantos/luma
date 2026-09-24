// The browser's IANA time zone (e.g. "America/Manaus"). Sent to endpoints
// that depend on what "today" is, so the server counts the user's own days
// rather than UTC days.
export function browserTimeZone(): string | undefined {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || undefined;
  } catch {
    return undefined;
  }
}

// "tz=America%2FManaus" (or an empty string when the zone is unknown), ready
// to be joined to other query parameters.
export function timeZoneParam(): string {
  const tz = browserTimeZone();
  return tz ? `tz=${encodeURIComponent(tz)}` : "";
}
