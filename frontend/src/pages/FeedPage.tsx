import { useState, useEffect, useRef } from 'react'
import { useParams } from '@tanstack/react-router'
import { useQueries } from '@tanstack/react-query'
import FeedEmpty from '@/components/feed/FeedEmpty'
import ClusterCardSkeleton, { TopicSectionHeaderSkeleton } from '@/components/feed/ClusterCardSkeleton'
import TopicFeedSection from '@/components/topics/TopicFeedSection'
import { fetchCluster } from '@/api/cluster'
import { queryKeys } from '@/api/queryKeys'
import type { MockCluster } from '@/lib/mockData'
import { useFeed } from '@/hooks/useFeed'
import { mapClusterResponseArticles } from '@/lib/clusterArticles'
import { topicLabelFromSlug, topicValueFromSlug } from '@/lib/topics'
import { Loader2, ArrowDown } from 'lucide-react'

const FEED_PAGE_SIZE = 10

export default function FeedPage() {
  const { topic } = useParams({ strict: false })
  const topicSlug = typeof topic === 'string' ? topic : undefined
  const topicName = topicLabelFromSlug(topicSlug)
  const topicValue = topicValueFromSlug(topicSlug)

  const [page, setPage] = useState(0)
  const [allClusterIds, setAllClusterIds] = useState<number[]>([])
  const [hasMore, setHasMore] = useState(true)
  const prevTopicRef = useRef(topicValue)

  // Reset when topic changes
  useEffect(() => {
    if (prevTopicRef.current !== topicValue) {
      prevTopicRef.current = topicValue
      setPage(0)
      setHasMore(true)
      setAllClusterIds([])
    }
  }, [topicValue])

  const { data, isLoading, error } = useFeed(topicValue ?? '', page, Boolean(topicValue))

  // Accumulate cluster IDs as pages load
  useEffect(() => {
    if (!data?.clusters) return
    const newIds = data.clusters.map((c) => c.id)
    setAllClusterIds((prev) => {
      const existing = new Set(prev)
      const fresh = newIds.filter((id) => !existing.has(id))
      return fresh.length > 0 ? [...prev, ...fresh] : prev
    })
    setHasMore(newIds.length >= FEED_PAGE_SIZE)
  }, [data])

  const clusterQueries = useQueries({
    queries: allClusterIds.map((clusterID) => ({
      queryKey: queryKeys.cluster(clusterID),
      queryFn: () => fetchCluster(clusterID),
      staleTime: 5 * 60 * 1000,
      retry: 2,
      enabled: Boolean(topicValue),
    })),
  })

  const hasClusterError = clusterQueries.some((query) => query.isError)
  const isClustersLoading = clusterQueries.some((query) => query.isLoading)
  const isLoadingMore = isLoading && page > 0

  const topicClusters: MockCluster[] = []
  clusterQueries.forEach((query) => {
    if (!query.data) return
    const mappedArticles: MockCluster['articles'] = mapClusterResponseArticles(query.data.articles)
    topicClusters.push({
      id: query.data.cluster.id,
      headline: query.data.cluster.headline,
      displayTitle: query.data.cluster.display_title,
      topic: query.data.cluster.topic,
      articles: mappedArticles,
    })
  })

  const showSkeleton = (isLoading || isClustersLoading) && page === 0 && topicClusters.length === 0

  return (
    <div className="space-y-6">
      {showSkeleton && (
        <div className="space-y-6">
          <TopicSectionHeaderSkeleton />
          <div className="space-y-6">
            <ClusterCardSkeleton />
            <ClusterCardSkeleton />
            <ClusterCardSkeleton />
          </div>
        </div>
      )}

      {error && (
        <p className="text-sm text-destructive">
          Неуспешно вчитување на групи за оваа тема.
        </p>
      )}

      {!isLoading && !error && hasClusterError && (
        <p className="text-sm text-destructive">
          Неуспешно вчитување детали за групата.
        </p>
      )}

      {!isLoading && !error && !isClustersLoading && !hasClusterError && data !== undefined && (data.clusters?.length ?? 0) === 0 && <FeedEmpty />}

      {topicClusters.length > 0 && (
        <>
          <TopicFeedSection topicName={topicName} clusters={topicClusters} />

          {hasMore && (
            <div className="flex justify-center pt-2 pb-6">
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={isLoadingMore || isClustersLoading}
                className="group flex items-center gap-2 px-4 py-2 rounded-sm border border-border/60 bg-muted/40 hover:bg-muted hover:border-editorial/40 transition-all duration-150 disabled:opacity-50"
                style={{ fontFamily: 'var(--font-mono)', letterSpacing: '0.05em' }}
              >
                {isLoadingMore ? (
                  <Loader2 className="w-3 h-3 text-editorial shrink-0 animate-spin" />
                ) : (
                  <ArrowDown className="w-3 h-3 text-editorial shrink-0 transition-transform duration-150 group-hover:translate-y-0.5" />
                )}
                <span className="text-xs font-semibold uppercase tracking-wider text-foreground">
                  {isLoadingMore ? 'Вчитување...' : 'Вчитај повеќе'}
                </span>
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
