export function ClusterPageSkeleton() {
  return (
    <div className="mx-auto max-w-272 animate-pulse space-y-6">
      <div className="space-y-2 border-b-2 border-foreground pb-3">
        <div className="h-3 w-24 rounded-sm bg-muted" />
        <div className="h-10 w-3/4 rounded-sm bg-muted" />
      </div>
      <div className="h-4 w-2/3 rounded-sm bg-muted" />
      <div className="w-full rounded-sm bg-muted" style={{ aspectRatio: '21/9' }} />
      <div className="space-y-2">
        <div className="h-7 w-full rounded-sm bg-muted" />
        <div className="h-7 w-4/5 rounded-sm bg-muted" />
        <div className="h-4 w-full rounded-sm bg-muted" />
      </div>
      {[1, 2, 3].map((i) => (
        <div key={i} className="flex gap-4 border-t border-border/60 py-5">
          <div className="h-7 w-8 shrink-0 rounded-sm bg-muted" />
          <div className="flex-1 space-y-2">
            <div className="h-3 w-28 rounded-sm bg-muted" />
            <div className="h-5 w-3/4 rounded-sm bg-muted" />
          </div>
          <div className="h-14 w-20 shrink-0 rounded-sm bg-muted" />
        </div>
      ))}
    </div>
  )
}

export default ClusterPageSkeleton
