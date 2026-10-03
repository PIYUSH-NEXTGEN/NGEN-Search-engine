import SearchBar from "@/components/SearchBar";
import MemberCard from "@/components/MemberCard";
import { searchMembers } from "@/lib/api";

interface SearchPageProps {
  searchParams: { q?: string };
}

// Server component: fetches results on the server for the initial load.
// (Phase 2 will add the generated paragraph answer above these cards.)
export default async function SearchPage({ searchParams }: SearchPageProps) {
  const query = searchParams.q ?? "";
  const data = query ? await searchMembers(query) : null;

  return (
    <main className="mx-auto max-w-2xl px-4 py-10">
      <SearchBar initialValue={query} />

      <div className="mt-8">
        {!query && (
          <p className="text-stone-400">Enter a search to get started.</p>
        )}

        {query && data && data.results.length === 0 && (
          <p className="text-stone-500">
            No matches found for &ldquo;{query}&rdquo;.
          </p>
        )}

        {query && data && data.results.length > 0 && (
          <div>
            {data.results.map((member) => (
              <MemberCard key={member.id} member={member} />
            ))}
          </div>
        )}
      </div>
    </main>
  );
}
