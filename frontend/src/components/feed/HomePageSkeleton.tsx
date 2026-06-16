import { Fragment } from 'react'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
import { DesktopClusterSkeleton } from '@/components/feed/ClusterCardSkeleton'

function ArticleRowSkeleton() {
  return (
    <div className="flex items-start gap-3 py-2.5 border-t border-border/60 first:border-t-0">
      <Skeleton className="w-5 h-5 rounded-sm shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0 space-y-1.5">
        <Skeleton className="h-2.5 w-16" />
        <Skeleton className="h-3.5 w-full" />
        <Skeleton className="h-3.5 w-4/5" />
      </div>
    </div>
  )
}

// Rendered inside HomePage's Card — mirrors the loaded structure:
// two featured StoryClusters (embedded) separated by a Separator.
export default function HomePageSkeleton() {
  return (
    <>
      {/* Desktop: two featured-cluster blocks */}
      <div className="hidden md:block">
        {Array.from({ length: 2 }).map((_, i) => (
          <Fragment key={i}>
            {i > 0 && <Separator />}
            <div className="p-4">
              <div className="p-4 md:p-5">
                <DesktopClusterSkeleton />
              </div>
            </div>
          </Fragment>
        ))}
      </div>

      {/* Mobile: stacked */}
      <div className="md:hidden p-4 space-y-3">
        <Skeleton className="aspect-video w-full rounded" />
        <div className="flex items-center gap-2">
          <Skeleton className="w-3.5 h-3.5 rounded-sm shrink-0" />
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
    </>
  )
}
