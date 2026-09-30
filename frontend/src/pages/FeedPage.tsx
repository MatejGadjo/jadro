import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import FeedEmpty from '@/components/feed/FeedEmpty'
import ClusterCardSkeleton, { TopicSectionHeaderSkeleton } from '@/components/feed/ClusterCardSkeleton'
import TopicFeedSection from '@/components/topics/TopicFeedSection'
import type { FeedCluster } from '@/lib/feedTypes'
import type { ClusterDetail } from '@/api/types'
import { useFeed } from '@/hooks/useFeed'
import { mapClusterResponseArticles } from '@/lib/clusterArticles'
import { topicLabelFromSlug, topicValueFromSlug } from '@/lib/topics'
import { FEED_PAGE_SIZE } from '@/lib/constants'
import { Loader2, ArrowDown } from 'lucide-react'

export default function FeedPage() {
  const { topic } = useParams({ strict: false })
  const topicSlug = typeof topic === 'string' ? topic : undefined
  const topicName = topicLabelFromSlug(topicSlug)
  const topicValue = topicValueFromSlug(topicSlug)

  const [prevTopicSlug, setPrevTopicSlug] = useState(topicSlug)
  const [page, setPage] = useState(0)
  const [accumulated, setAccumulated] = useState<ClusterDetail[]>([])
  const [hasMore, setHasMore] = useState(true)

  // Synchronous reset when topic changes — avoids key-based remount flash
  const topicChanged = prevTopicSlug !== topicSlug
  const activePage = topicChanged ? 0 : page
  const activeDetails = topicChanged ? [] : accumulated
  if (topicChanged) {
    setPrevTopicSlug(topicSlug)
    setPage(0)
    setAccumulated([])
    setHasMore(true)
  }

  const { data, isLoading, error } = useFeed(topicValue ?? '', activePage, Boolean(topicValue))

  // Accumulate cluster details (with embedded articles) as pages load.
  // Adjusted during render when a new page arrives (same pattern as the topic
  // reset above) instead of in an effect, which would render twice.
  const [seenData, setSeenData] = useState(data)
  if (data !== seenData) {
    setSeenData(data)
    const details = data?.cluster_details
    if (details) {
      setAccumulated((prev) => {
        const existing = new Set(prev.map((d) => d.cluster.id))
        const fresh = details.filter((d) => !existing.has(d.cluster.id))
        return fresh.length > 0 ? [...prev, ...fresh] : prev
      })
      setHasMore(details.length >= FEED_PAGE_SIZE)
    }
  }

  const isLoadingMore = isLoading && activePage > 0

  const topicClusters: FeedCluster[] = activeDetails.map((detail) => ({
    id: detail.cluster.id,
    headline: detail.cluster.headline,
    displayTitle: detail.cluster.display_title,
    topic: detail.cluster.topic,
    articles: mapClusterResponseArticles(detail.articles),
  }))

  const showSkeleton = isLoading && activePage === 0 && topicClusters.length === 0

  return (
    <div className="space-y-6">
      {showSkeleton && (
        <div className="max-w-4xl mx-auto space-y-6">
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

      {!isLoading && !error && data !== undefined && (data.cluster_details?.length ?? 0) === 0 && <FeedEmpty />}

      {topicClusters.length > 0 && (
        <>
          <TopicFeedSection topicName={topicName} clusters={topicClusters} />

          {hasMore && (
            <div className="flex justify-center pt-2 pb-6">
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={isLoadingMore}
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
