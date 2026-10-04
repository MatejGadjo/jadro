import { useState } from 'react'
import { useFeed } from '@/hooks/useFeed'
import { mapClusterResponseArticles } from '@/lib/clusterArticles'
import StoryCluster from '@/components/feed/StoryCluster'
import ClusterCardSkeleton from '@/components/feed/ClusterCardSkeleton'
import { Loader2, ArrowDown } from 'lucide-react'
import type { FeedCluster } from '@/lib/feedTypes'
import type { ClusterDetail } from '@/api/types'
import { FEED_PAGE_SIZE } from '@/lib/constants'

interface Props {
  featuredCount: number
  featuredClusterIds: Set<number>
}

export default function MobileTopStoriesFeed({ featuredCount, featuredClusterIds }: Props) {
  const [page, setPage] = useState(0)
  const [accumulated, setAccumulated] = useState<ClusterDetail[]>([])
  const [hasMore, setHasMore] = useState(true)
  const [visibleSlice, setVisibleSlice] = useState(() => Math.max(1, FEED_PAGE_SIZE - featuredCount))

  const { data, isLoading } = useFeed('', page)

  // Accumulate pages during render when new data (or a new featured set)
  // arrives, rather than in an effect that would trigger a second render.
  const [seen, setSeen] = useState({ data, featuredClusterIds })
  if (data !== seen.data || featuredClusterIds !== seen.featuredClusterIds) {
    setSeen({ data, featuredClusterIds })
    const details = data?.cluster_details
    if (details) {
      const fresh = details.filter((d) => !featuredClusterIds.has(d.cluster.id))
      setAccumulated((prev) => {
        const existing = new Set(prev.map((d) => d.cluster.id))
        const unseen = fresh.filter((d) => !existing.has(d.cluster.id))
        return unseen.length > 0 ? [...prev, ...unseen] : prev
      })
      // Use raw count (before dedup) to reflect actual backend pagination state
      setHasMore(details.length >= FEED_PAGE_SIZE)
    }
  }

  const resolvedClusters: FeedCluster[] = accumulated.map((d) => ({
    id: d.cluster.id,
    headline: d.cluster.headline,
    displayTitle: d.cluster.display_title,
    topic: d.cluster.topic,
    articles: mapClusterResponseArticles(d.articles),
  }))

  const visibleClusters = resolvedClusters.slice(0, visibleSlice)
  const isFetchingMore = isLoading && page > 0
  const isInitialLoading = isLoading && accumulated.length === 0
  const canShowMore = visibleSlice < resolvedClusters.length || (hasMore && resolvedClusters.length <= visibleSlice)

  function handleLoadMore() {
    if (visibleSlice < resolvedClusters.length) {
      setVisibleSlice((v) => v + FEED_PAGE_SIZE)
    } else {
      setPage((p) => p + 1)
      setVisibleSlice((v) => v + FEED_PAGE_SIZE)
    }
  }

  if (isInitialLoading) {
    return (
      <div className="space-y-4 mt-4">
        <ClusterCardSkeleton />
        <ClusterCardSkeleton />
      </div>
    )
  }

  return (
    <div className="space-y-4 mt-4">
      {visibleClusters.map((cluster) => (
        <StoryCluster
          key={cluster.id}
          cluster={cluster}
          featured={Boolean(cluster.articles[0]?.imageUrl)}
          embedded={false}
        />
      ))}

      {canShowMore && (
        <div className="pt-1 pb-2 flex justify-center">
          <button
            onClick={handleLoadMore}
            disabled={isFetchingMore}
            className="group flex items-center gap-2 px-4 py-2 rounded-sm border border-border/60 bg-muted/40 hover:bg-muted hover:border-editorial/40 transition-all duration-150 disabled:opacity-50"
            style={{ fontFamily: 'var(--font-mono)', letterSpacing: '0.05em' }}
          >
            {isFetchingMore ? (
              <Loader2 className="w-3 h-3 text-editorial shrink-0 animate-spin" />
            ) : (
              <ArrowDown className="w-3 h-3 text-editorial shrink-0 transition-transform duration-150 group-hover:translate-y-0.5" />
            )}
            <span className="text-xs font-semibold uppercase tracking-wider text-foreground">
              {isFetchingMore ? 'Вчитување...' : 'Вчитај повеќе'}
            </span>
          </button>
        </div>
      )}
    </div>
  )
}
