import type { MemberResult } from "@/lib/types";

export default function MemberCard({ member }: { member: MemberResult }) {
  return (
    <div className="border-b border-stone-200 py-5 last:border-none">
      <h3 className="text-lg font-medium text-stone-900">{member.full_name}</h3>
      {member.headline && (
        <p className="mt-0.5 text-sm text-stone-500">{member.headline}</p>
      )}
      {member.bio && (
        <p className="mt-2 max-w-2xl text-sm leading-relaxed text-stone-700">
          {member.bio}
        </p>
      )}
      {member.location && (
        <p className="mt-2 text-xs text-stone-400">{member.location}</p>
      )}
    </div>
  );
}
