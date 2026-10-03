"use client";

import { useState, FormEvent } from "react";
import { useRouter } from "next/navigation";

interface SearchBarProps {
  initialValue?: string;
}

export default function SearchBar({ initialValue = "" }: SearchBarProps) {
  const [value, setValue] = useState(initialValue);
  const router = useRouter();

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed) return;
    router.push(`/search?q=${encodeURIComponent(trimmed)}`);
  }

  return (
    <form onSubmit={handleSubmit} className="w-full max-w-xl">
      <div className="flex items-center gap-2 rounded-full border border-stone-300 bg-white px-5 py-3 shadow-sm focus-within:border-stone-500">
        <input
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="Search the community..."
          className="w-full bg-transparent text-base text-stone-800 outline-none placeholder:text-stone-400"
          autoFocus
        />
        <button
          type="submit"
          className="rounded-full bg-stone-800 px-4 py-1.5 text-sm text-white transition hover:bg-stone-700"
        >
          Search
        </button>
      </div>
    </form>
  );
}
