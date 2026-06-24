import { useParams } from '@tanstack/react-router'
import axios from 'axios'
import { Clock3, Newspaper } from 'lucide-react'
import { useCluster } from '@/hooks/useCluster'
import { toRelativeTime } from '@/lib/clusterArticles'
import { ClusterArticleItem } from '@/components/cluster/ClusterArticleItem'
import { ClusterPageSkeleton } from '@/components/cluster/ClusterPageSkeleton'
import { NotFound } from '@/components/NotFound'
import { topicSlugFromValue } from '@/lib/topics'

export default function ClusterPage() {
  const { clusterId } = useParams({ strict: false })

  // A non-numeric id (e.g. /cluster/abc) can never resolve — show the 404 page
  // without firing a request.
  if (clusterId === undefined || !Number.isInteger(Number(clusterId))) {
    return <NotFound title="Веста не постои" />
  }

  return <ClusterPageContent clusterId={clusterId} />
}

function ClusterPageContent({ clusterId }: { clusterId: string }) {
  const { data, isLoading, error } = useCluster(clusterId)

  if (isLoading) return <ClusterPageSkeleton />

  if (error) {
    // A missing cluster (deleted/expired or never existed) is a 404, not an
    // error — show the branded 404. Real failures keep the error message.
    if (axios.isAxiosError(error) && error.response?.status === 404) {
      return <NotFound title="Веста не постои" />
    }
    return (
      <div className="mx-auto max-w-272 py-12 text-center">
        <p className="text-sm text-destructive">
          Грешка при вчитување: {error instanceof Error ? error.message : 'Непозната грешка'}
        </p>
      </div>
    )
  }

  if (!data) {
    return <NotFound title="Веста не постои" />
  }

  const { cluster, articles } = data
  const clusterTitle = cluster.display_title || cluster.headline || 'Главни Вести'
  const topicSlug = topicSlugFromValue(cluster.topic)

  return (
    <div className="mx-auto max-w-272">
      {/* Section header */}
      <div className="mb-6 border-b-2 border-foreground pb-3">
        <div className="flex items-end gap-3">
          <div
            className="w-1 shrink-0 bg-editorial"
            style={{ minHeight: '2.5rem', alignSelf: 'stretch' }}
          />
          <div className="min-w-0 flex-1">
            <div
              style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.12em' }}
              className="mb-1.5 uppercase text-muted-foreground"
            >
              Целосна покриеност · {articles.length} извори
            </div>
            <h1
              className="font-black text-foreground"
              style={{
                fontFamily: 'var(--font-serif)',
                fontStyle: 'italic',
                // Inline size overrides the global `h1 { font-size: 2.8em }` rule
                // in index.css (unlayered → beats Tailwind utilities). clamp()
                // scales the headline down on mobile while keeping the large
                // size on wide desktops.
                fontSize: 'clamp(1.5rem, 5.5vw, 2.8rem)',
                lineHeight: 1.15,
              }}
            >
              {clusterTitle}
            </h1>
          </div>
        </div>
      </div>

      {/* Summary + meta pills — stacked on mobile, side-by-side on desktop */}
      <div className="mb-4 flex flex-col gap-4 md:mb-8 md:flex-row md:flex-wrap md:items-start">
        {cluster.summary && (
          <p className="flex-1 border-l-2 border-editorial/40 pl-4 text-sm leading-6 text-muted-foreground line-clamp-3 md:text-base md:leading-7 md:line-clamp-none md:min-w-50">
            {cluster.summary}
          </p>
        )}
        <div className="flex flex-row gap-2 md:flex-col md:shrink-0">
          <div className="inline-flex items-center gap-1.5 rounded-sm border border-border/70 bg-card px-2.5 py-1">
            <Newspaper className="h-3 w-3 text-editorial" />
            <span
              style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.05em' }}
              className="text-muted-foreground"
            >
              {articles.length} написи
            </span>
          </div>
          <div className="inline-flex items-center gap-1.5 rounded-sm border border-border/70 bg-card px-2.5 py-1">
            <Clock3 className="h-3 w-3 text-muted-foreground" />
            <span
              style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.04em' }}
              className="text-muted-foreground"
            >
              Ажурирано {toRelativeTime(cluster.updated_at, cluster.created_at)}
            </span>
          </div>
        </div>
      </div>

      {/* Article list */}
      {articles.length === 0 ? (
        <p
          className="py-12 text-center italic text-muted-foreground"
          style={{ fontFamily: 'var(--font-serif)' }}
        >
          Нема написи во оваа група.
        </p>
      ) : (
        <div className="flex flex-col gap-3 md:block md:gap-0">
          {articles.map((article, i) => (
            <ClusterArticleItem
              key={article.id}
              article={article}
              displayNumber={i + 1}
              topic={topicSlug}
            />
          ))}
        </div>
      )}
    </div>
  )
}
