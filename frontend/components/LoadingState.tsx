export default function LoadingState() {
  return (
    <div className="space-y-4 py-6">
      {[0, 1, 2].map((i) => (
        <div key={i} className="animate-pulse space-y-2">
          <div className="h-4 w-40 rounded bg-stone-200" />
          <div className="h-3 w-64 rounded bg-stone-100" />
        </div>
      ))}
    </div>
  );
}
