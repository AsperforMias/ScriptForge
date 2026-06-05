import type { ApiEnvelope } from "../types/api";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export function buildApiUrl(path: string) {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;

  return `${API_BASE_URL}${normalizedPath}`;
}

export async function requestJson<T>(path: string, init: RequestInit = {}) {
  const response = await fetch(buildApiUrl(path), {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });

  const payload = (await response.json()) as ApiEnvelope<T>;

  if (!response.ok || payload.error) {
    throw payload.error ?? {
      code: "internal_error",
      message: "Request failed",
      details: {},
    };
  }

  return payload;
}

export async function requestText(path: string, init: RequestInit = {}) {
  const response = await fetch(buildApiUrl(path), init);
  const text = await response.text();

  if (!response.ok) {
    throw new Error(text || "Request failed");
  }

  return text;
}

export { API_BASE_URL };
