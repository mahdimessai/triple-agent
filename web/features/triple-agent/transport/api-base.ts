const DEFAULT_API_BASE_URL = "http://localhost:8080";

export function apiBaseUrl(): string {
  return (process.env.NEXT_PUBLIC_TRIPLE_AGENT_API_URL ?? DEFAULT_API_BASE_URL).replace(/\/$/, "");
}

export function apiUrl(path: string): string {
  return `${apiBaseUrl()}${path}`;
}
