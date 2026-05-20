import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

function ArticleRowSkeleton() {
  return (
    <div className="flex items-start gap-3 py-2.5 border-t border-border/60 first:border-t-0">
      <Skeleton className="w-5 h-5 rounded-sm flex-shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0 space-y-1.5">
        <Skeleton className="h-2.5 w-16" />
        <Skeleton className="h-3.5 w-full" />
        <Skeleton className="h-3.5 w-4/5" />
      </div>
    </div>
  )
}

export default function HomePageSkeleton() {
  return (
    <Card className="bg-card border-border gap-0">
      <div className="p-4">
        {/* Desktop: two-column 5-grid layout */}
        <div className="hidden md:grid md:grid-cols-5 gap-5">
          {/* Left col: hero */}
          <div className="col-span-3 space-y-3">
            <Skeleton className="aspect-[16/9] w-full rounded" />
            <div className="flex items-center gap-2">
              <Skeleton className="w-3.5 h-3.5 rounded-sm flex-shrink-0" />
              <Skeleton className="h-2.5 w-20" />
            </div>
            <div className="space-y-2">
              <Skeleton className="h-7 w-full" />
              <Skeleton className="h-7 w-5/6" />
              <Skeleton className="h-7 w-2/3" />
            </div>
            <Skeleton className="h-2.5 w-28" />
          </div>

          {/* Right col: article rows */}
          <div className="col-span-2 flex flex-col">
            {Array.from({ length: 4 }).map((_, i) => (
              <ArticleRowSkeleton key={i} />
            ))}
          </div>
        </div>

        {/* Mobile: stacked */}
        <div className="md:hidden space-y-3">
          <Skeleton className="aspect-[16/9] w-full rounded" />
          <div className="flex items-center gap-2">
            <Skeleton className="w-3.5 h-3.5 rounded-sm flex-shrink-0" />
            <Skeleton className="h-2.5 w-20" />
          </div>
          <div className="space-y-2">
            <Skeleton className="h-7 w-full" />
            <Skeleton className="h-7 w-3/4" />
          </div>
          <div>
            {Array.from({ length: 3 }).map((_, i) => (
              <ArticleRowSkeleton key={i} />
            ))}
          </div>
        </div>
      </div>
    </Card>
  )
}
