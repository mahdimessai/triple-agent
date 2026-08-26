export function apiBaseUrl(): string {
  return (process.env.NEXT_PUBLIC_TRIPLE_AGENT_API_URL ?? "").replace(/\/$/, "");
}

export function apiUrl(path: string): string {
  return `${apiBaseUrl()}${path}`;
}
