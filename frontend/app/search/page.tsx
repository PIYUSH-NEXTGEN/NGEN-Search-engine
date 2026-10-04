import AskClient from "./AskClient";
import { askQuestion } from "@/lib/api";

interface SearchPageProps {
  searchParams: { q?: string };
}

// Server component: asks once on the server for the initial load (answer +
// member cards in a single /api/ask call), then hands the first turn to the
// client wrapper, which owns session_id and follow-up turns. A failed answer
// still shows the page — the client wrapper renders the error plainly.
export default async function SearchPage({ searchParams }: SearchPageProps) {
  const query = searchParams.q ?? "";
  const data = query ? await askQuestion(query).catch(() => null) : null;

  return (
    <main className="mx-auto max-w-2xl px-4 py-10">
      <AskClient
        initialTurn={data}
        initialQuery={query}
        initialError={
          query && !data
            ? "Couldn't generate an answer right now — please try again."
            : null
        }
      />
    </main>
  );
}
