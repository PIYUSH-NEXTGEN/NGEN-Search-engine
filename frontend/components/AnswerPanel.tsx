import type { AskResponse } from "@/lib/types";

interface AnswerPanelProps {
  turn: AskResponse;
}

// Renders one generated answer. The tinted card marks it as synthesized by
// the model; member cards still render separately below. An irrelevant query
// gets a plain one-liner with no cards (the caller hides them on relevant
// === false).
export default function AnswerPanel({ turn }: AnswerPanelProps) {
  if (!turn.relevant) {
    return (
      <p className="text-stone-500">
        &ldquo;{turn.query}&rdquo; doesn&apos;t seem related to this
        community — try asking about members, skills, roles, or projects.
      </p>
    );
  }

  return (
    <section
      aria-label="Generated answer"
      className="rounded-xl border border-stone-200 bg-stone-100 px-5 py-4"
    >
      <p className="text-xs font-medium uppercase tracking-wide text-stone-400">
        Generated answer
      </p>
      <p className="mt-2 text-base leading-relaxed text-stone-800">
        {turn.answer}
      </p>
    </section>
  );
}
