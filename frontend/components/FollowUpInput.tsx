"use client";

import { FormEvent, useState } from "react";
import { askQuestion } from "@/lib/api";
import type { AskResponse } from "@/lib/types";

interface FollowUpInputProps {
  sessionId: string;
  disabled?: boolean;
  onAnswer: (turn: AskResponse) => void;
  onError: (message: string) => void;
}

// Follow-up box shown under the answer. Sends the stored session_id so the
// backend resolves pronouns ("tell me more about her") from prior turns.
// The parent appends the new turn below the old ones — nothing navigates.
export default function FollowUpInput({
  sessionId,
  disabled = false,
  onAnswer,
  onError,
}: FollowUpInputProps) {
  const [value, setValue] = useState("");
  const [pending, setPending] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed || pending || disabled) return;
    setPending(true);
    try {
      const turn = await askQuestion(trimmed, sessionId);
      setValue("");
      onAnswer(turn);
    } catch {
      onError("Follow-up failed — please try again.");
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="mt-6 w-full">
      <div className="flex items-center gap-2 rounded-full border border-stone-300 bg-white px-5 py-2.5 shadow-sm focus-within:border-stone-500">
        <input
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="Ask a follow-up..."
          disabled={pending || disabled}
          className="w-full bg-transparent text-sm text-stone-800 outline-none placeholder:text-stone-400 disabled:text-stone-300"
        />
        <button
          type="submit"
          disabled={pending || disabled || !value.trim()}
          className="rounded-full bg-stone-800 px-4 py-1.5 text-sm text-white transition hover:bg-stone-700 disabled:bg-stone-300"
        >
          {pending ? "Asking..." : "Ask"}
        </button>
      </div>
    </form>
  );
}
