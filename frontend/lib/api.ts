import type { SearchResponse } from "./types";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export async function searchMembers(query: string): Promise<SearchResponse> {
  const url = new URL("/api/search", API_BASE_URL);
  url.searchParams.set("q", query);

  const res = await fetch(url.toString(), { cache: "no-store" });

  if (!res.ok) {
    throw new Error(`Search request failed with status ${res.status}`);
  }

  return res.json();
}
