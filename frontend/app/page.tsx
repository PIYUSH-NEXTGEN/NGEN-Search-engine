import SearchBar from "@/components/SearchBar";

export default function HomePage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 px-4">
      <h1 className="text-3xl font-medium text-stone-800">
        Search the community
      </h1>
      <SearchBar />
    </main>
  );
}
