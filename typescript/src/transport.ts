/**
 * Best-effort JSON POST. Telemetry/platform writes must never break the
 * originating request, so failures are logged and swallowed (returns null).
 */

export async function postJson(
  url: string,
  payload: unknown,
  opts: { headers?: Record<string, string>; timeoutMs?: number; label?: string } = {},
): Promise<Response | null> {
  const label = opts.label ?? "platform";
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), opts.timeoutMs ?? 5000);
  try {
    const resp = await fetch(url, {
      method: "POST",
      headers: { "content-type": "application/json", ...(opts.headers ?? {}) },
      body: JSON.stringify(payload),
      signal: controller.signal,
    });
    if (!resp.ok) {
      console.warn(`[streamoid:${label}] POST ${url} -> ${resp.status}`);
      return null;
    }
    return resp;
  } catch (err) {
    console.warn(`[streamoid:${label}] POST ${url} failed:`, (err as Error)?.message);
    return null;
  } finally {
    clearTimeout(timer);
  }
}

export function authHeaders(token?: string): Record<string, string> {
  return token ? { authorization: `Bearer ${token}` } : {};
}
