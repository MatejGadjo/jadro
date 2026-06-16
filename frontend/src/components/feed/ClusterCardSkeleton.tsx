import { Skeleton } from '@/components/ui/skeleton'

function FeaturedRowSkeleton() {
  return (
    <div className="flex flex-col flex-1 px-3 py-2 border-t border-border/60 first:border-t-0 min-h-0">
      <div className="flex items-center gap-1.5 mb-1">
        <Skeleton className="w-3.5 h-3.5 rounded-sm shrink-0" />
        <Skeleton className="h-2 w-16" />
      </div>
      <Skeleton className="h-3.5 w-full mb-1" />
      <Skeleton className="h-3.5 w-4/5 mb-auto" />
      <Skeleton className="h-2 w-14 mt-1 shrink-0" />
    </div>
  )
}


function HorizontalCardSkeleton() {
  return (
    <div className="flex flex-col w-52 shrink-0 bg-card border border-border rounded-sm p-3">
      <div className="flex items-center gap-1.5 mb-2">
        <Skeleton className="w-4 h-4 rounded-sm shrink-0" />
        <Skeleton className="h-2 w-16" />
      </div>
      <Skeleton className="h-3.5 w-full mb-1" />
      <Skeleton className="h-3.5 w-4/5 mb-1" />
      <Skeleton className="h-3.5 w-3/5 mb-auto" />
      <Skeleton className="h-2 w-12 mt-2" />
    </div>
  )
}

export function TopicSectionHeaderSkeleton() {
  return (
    <div className="flex items-end gap-3 border-b-2 border-foreground pb-3">
      <Skeleton className="w-1 rounded-none min-h-10 self-stretch" />
      <div className="space-y-2">
        <Skeleton className="h-9 w-48" />
        <Skeleton className="h-2.5 w-36" />
      </div>
    </div>
  )
}

// Desktop featured-cluster layout: label row + col-span-2 image left + col-span-3 article rows right.
// Mirrors StoryCluster's featured desktop grid; reused by HomePageSkeleton.
export function DesktopClusterSkeleton() {
  return (
    <>
      {/* Cluster label row */}
      <div className="flex items-center gap-3 mb-3">
        <Skeleton className="h-4 w-0.5 rounded-none" />
        <Skeleton className="h-3 w-28" />
      </div>

      <div className="grid grid-cols-5 gap-4">
        <div className="col-span-2 flex flex-col gap-2">
          <Skeleton className="aspect-video w-full rounded-sm" />
          <div className="flex items-center gap-1.5">
            <Skeleton className="w-4 h-4 rounded-sm shrink-0" />
            <Skeleton className="h-2.5 w-20" />
          </div>
          <div className="space-y-1.5 flex-1">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
          </div>
          <Skeleton className="h-2.5 w-24" />
          <div className="mt-2 pt-2 border-t border-border/60">
            <Skeleton className="h-7 w-full rounded-sm" />
          </div>
        </div>

        <div className="col-span-3 flex flex-col border-l border-border/60">
          <FeaturedRowSkeleton />
          <FeaturedRowSkeleton />
          <FeaturedRowSkeleton />
        </div>
      </div>
    </>
  )
}

export default function ClusterCardSkeleton() {
  return (
    <div className="bg-card border border-border rounded-sm overflow-hidden p-4 md:p-5">
      {/* Desktop */}
      <div className="hidden md:block">
        <DesktopClusterSkeleton />
      </div>

      {/* Mobile: featured article + horizontal scroll strip */}
      <div className="md:hidden space-y-3">
        <Skeleton className="aspect-video w-full rounded-sm" />
        <div className="flex items-center gap-1.5">
          <Skeleton className="w-4 h-4 rounded-sm shrink-0" />
          <Skeleton className="h-2.5 w-20" />
        </div>
        <div className="space-y-1.5">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-5/6" />
        </div>
        {/* Horizontal scroll strip */}
        <div className="flex gap-3 overflow-hidden -mx-4 px-4">
          <HorizontalCardSkeleton />
          <HorizontalCardSkeleton />
          <HorizontalCardSkeleton />
        </div>
        {/* SeeMoreLink */}
        <div className="pt-2 border-t border-border/60">
          <Skeleton className="h-9 w-full rounded-sm" />
        </div>
      </div>
    </div>
  )
}
