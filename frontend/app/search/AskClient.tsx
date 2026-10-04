"use client";

import { useState } from "react";
import SearchBar from "@/components/SearchBar";
import MemberCard from "@/components/MemberCard";
import AnswerPanel from "@/components/AnswerPanel";
import FollowUpInput from "@/components/FollowUpInput";
import type { AskResponse } from "@/lib/types";

interface AskClientProps {
  initialTurn: AskResponse | null;
  initialQuery: string;
  initialError: string | null;
}

// Client wrapper: the page stays a server component and hands it the first
// turn (or an error). session_id lives here in React state; each follow-up
// appends a turn below the previous ones so the thread stays visible.
export default function AskClient({
  initialTurn,
  initialQuery,
  initialError,
}: AskClientProps) {
  const [turns, setTurns] = useState<AskResponse[]>(
    initialTurn ? [initialTurn] : [],
  );
  const [sessionId, setSessionId] = useState<string | null>(
    initialTurn ? initialTurn.session_id : null,
  );
  const [error, setError] = useState<string | null>(initialError);

  const latest = turns[turns.length - 1] ?? null;

  function handleAnswer(turn: AskResponse) {
    setSessionId(turn.session_id);
    setTurns((prev) => [...prev, turn]);
    setError(null);
  }

  return (
    <>
      <SearchBar initialValue={initialQuery} />

      <div className="mt-8 space-y-8">
        {error && <p className="text-stone-500">{error}</p>}

        {!initialQuery && turns.length === 0 && !error && (
          <p className="text-stone-400">Enter a search to get started.</p>
        )}

        {turns.map((turn, i) => (
          <div key={`${turn.session_id}-${i}`} className="space-y-4">
            <AnswerPanel turn={turn} />
            {turn.relevant && turn.results.length > 0 && (
              <div>
                {turn.results.map((member) => (
                  <MemberCard key={member.id} member={member} />
                ))}
              </div>
            )}
            {turn.relevant && turn.results.length === 0 && (
              <p className="text-sm text-stone-400">
                No member cards matched this one, but the answer above still
                uses earlier context.
              </p>
            )}
          </div>
        ))}

        {latest && latest.relevant && sessionId && (
          <FollowUpInput
            sessionId={sessionId}
            onAnswer={handleAnswer}
            onError={setError}
          />
        )}
      </div>
    </>
  );
}
