import { useState, useEffect } from 'react'
import { useQueries } from '@tanstack/react-query'
import { useFeed } from '@/hooks/useFeed'
import { fetchCluster } from '@/api/cluster'
import { queryKeys } from '@/api/queryKeys'
import { mapClusterResponseArticles } from '@/lib/clusterArticles'
import StoryCluster from '@/components/feed/StoryCluster'
import ClusterCardSkeleton from '@/components/feed/ClusterCardSkeleton'
import { Loader2, ArrowDown } from 'lucide-react'
import type { MockCluster } from '@/lib/mockData'

const PER_PAGE = 10

interface Props {
  featuredCount: number
  featuredClusterIds: Set<number>
}

export default function MobileTopStoriesFeed({ featuredCount, featuredClusterIds }: Props) {
  const [page, setPage] = useState(0)
  const [allClusterIds, setAllClusterIds] = useState<number[]>([])
  const [hasMore, setHasMore] = useState(true)
  const [visibleSlice, setVisibleSlice] = useState(() => Math.max(1, PER_PAGE - featuredCount))

  const { data, isLoading } = useFeed('', page)

  useEffect(() => {
    if (!data?.clusters) return
    const newIds = data.clusters
      .map((c) => c.id)
      .filter((id) => !featuredClusterIds.has(id))
    setAllClusterIds((prev) => {
      const existing = new Set(prev)
      const fresh = newIds.filter((id) => !existing.has(id))
      return fresh.length > 0 ? [...prev, ...fresh] : prev
    })
    // Use raw count (before dedup) to reflect actual backend pagination state
    setHasMore(data.clusters.length >= PER_PAGE)
  }, [data, featuredClusterIds])

  const clusterQueries = useQueries({
    queries: allClusterIds.map((id) => ({
      queryKey: queryKeys.cluster(id),
      queryFn: () => fetchCluster(id),
      staleTime: 5 * 60 * 1000,
      retry: 2,
    })),
  })

  const resolvedClusters: MockCluster[] = clusterQueries
    .filter((q) => Boolean(q.data))
    .map((q) => ({
      id: q.data!.cluster.id,
      headline: q.data!.cluster.headline,
      displayTitle: q.data!.cluster.display_title,
      topic: q.data!.cluster.topic,
      articles: mapClusterResponseArticles(q.data!.articles),
    }))

  const visibleClusters = resolvedClusters.slice(0, visibleSlice)
  const isFetchingMore = isLoading && page > 0
  const isInitialLoading = isLoading && allClusterIds.length === 0
  const canShowMore = visibleSlice < resolvedClusters.length || (hasMore && resolvedClusters.length <= visibleSlice)

  function handleLoadMore() {
    if (visibleSlice < resolvedClusters.length) {
      setVisibleSlice((v) => v + PER_PAGE)
    } else {
      setPage((p) => p + 1)
      setVisibleSlice((v) => v + PER_PAGE)
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
