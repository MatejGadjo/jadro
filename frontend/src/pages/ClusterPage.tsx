import { useParams } from '@tanstack/react-router'
import { clearArticleImage } from '@/api/cluster'
import { format, isValid, parseISO } from 'date-fns'
import { mk } from 'date-fns/locale'
import { Clock3, Newspaper } from 'lucide-react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { useCluster } from '@/hooks/useCluster'

type ClusterArticle = NonNullable<ReturnType<typeof useCluster>['data']>['articles'][number]

function getSourceInitials(name: string) {
  return name
    .split(' ')
    .map((word) => word[0])
    .join('')
    .substring(0, 2)
    .toUpperCase()
}

function formatArticleDate(publishedAt: string | null | undefined, fetchedAt: string) {
  const candidate = publishedAt ?? fetchedAt
  const parsed = parseISO(candidate)
  if (!isValid(parsed)) return 'Recently'
  return format(parsed, 'd MMM yyyy', { locale: mk })
}

function SourceAvatar({ article }: { article: ClusterArticle }) {
  return (
    <Avatar className="h-4 w-4 rounded-sm ring-1 ring-border/50 shrink-0">
      <AvatarImage
        src={article.source.logo_url}
        alt={article.source.name}
        className="rounded-sm bg-card object-contain p-0.5"
      />
      <AvatarFallback className="rounded-sm bg-editorial text-[8px] font-bold text-white">
        {getSourceInitials(article.source.name)}
      </AvatarFallback>
    </Avatar>
  )
}


function ArticleItem({
  article,
  displayNumber,
}: {
  article: ClusterArticle
  displayNumber: number
}) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      className="group flex flex-col overflow-hidden rounded-md border border-border/60 bg-card shadow-sm transition-colors duration-100 hover:bg-accent/30
                 md:flex-row md:items-start md:gap-4 md:rounded-none md:border-0 md:border-t md:bg-transparent md:shadow-none md:-mx-5 md:px-5 md:py-5"
    >
      {/* Thumbnail — full-width on top for mobile, small right-side on desktop */}
      {article.image_url && (
        <div
          className="w-full aspect-video shrink-0 overflow-hidden bg-muted
                     md:aspect-auto md:h-16 md:w-24 md:self-start md:rounded-sm md:order-last"
        >
          <img
            src={article.image_url}
            alt={article.title}
            className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.04]"
            onError={(e) => { e.currentTarget.style.visibility = 'hidden'; clearArticleImage(article.id) }}
          />
        </div>
      )}

      {/* Text content */}
      <div className="min-w-0 flex-1 p-3.5 md:p-0 md:order-2">
        {/* Source row — number pill + avatar + name + date all on one line */}
        <div className="mb-2 flex items-center gap-1.5 md:mb-1.5 md:gap-2">
          {/* Number pill: mobile only */}
          <span
            className="shrink-0 select-none rounded-sm bg-editorial/10 px-1.5 py-0.5 font-black leading-none text-editorial md:hidden"
            style={{ fontFamily: 'var(--font-mono)', fontSize: '9px', letterSpacing: '0.06em' }}
            aria-hidden
          >
            {String(displayNumber).padStart(2, '0')}
          </span>
          <SourceAvatar article={article} />
          <span
            style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.08em' }}
            className="truncate font-semibold uppercase text-muted-foreground"
          >
            {article.source.name}
          </span>
          <span className="text-muted-foreground/50">·</span>
          <span
            style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
            className="shrink-0 text-muted-foreground"
          >
            {formatArticleDate(article.published_at, article.fetched_at)}
          </span>
        </div>

        <h3
          className="line-clamp-2 text-base font-bold leading-snug text-foreground transition-colors duration-150 group-hover:text-editorial md:text-[1.05rem]"
          style={{ fontFamily: 'var(--font-serif)' }}
        >
          {article.title}
        </h3>

        {article.snippet && (
          <p className="mt-1.5 hidden line-clamp-2 text-sm leading-6 text-muted-foreground md:block">
            {article.snippet}
          </p>
        )}
      </div>

      {/* Large italic number — desktop only */}
      <span
        className="mt-0.5 hidden shrink-0 select-none font-black leading-none text-foreground/20 transition-colors duration-150 group-hover:text-editorial/60 md:block md:order-first"
        style={{
          fontFamily: 'var(--font-serif)',
          fontStyle: 'italic',
          fontSize: '1.6rem',
          lineHeight: '1.1',
          minWidth: '2.5rem',
          textAlign: 'right',
        }}
        aria-hidden
      >
        {String(displayNumber).padStart(2, '0')}
      </span>
    </a>
  )
}

function LoadingSkeleton() {
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

export default function ClusterPage() {
  const { clusterId } = useParams({ strict: false })
  const { data, isLoading, error } = useCluster(clusterId)

  if (isLoading) return <LoadingSkeleton />

  if (error) {
    return (
      <div className="mx-auto max-w-272 py-12 text-center">
        <p className="text-sm text-destructive">
          Грешка при вчитување: {error instanceof Error ? error.message : 'Непозната грешка'}
        </p>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="mx-auto max-w-272 py-12 text-center">
        <p className="italic text-muted-foreground" style={{ fontFamily: 'var(--font-serif)' }}>
          Нема достапни податоци за оваа група.
        </p>
      </div>
    )
  }

  const { cluster, articles } = data
  const clusterTitle = cluster.display_title || cluster.headline || 'Главни Вести'

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
              className="text-base font-black leading-snug text-foreground md:text-4xl md:leading-tight"
              style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic' }}
            >
              {clusterTitle}
            </h1>
          </div>
        </div>
      </div>

      {/* Summary + meta pills — stacked on mobile, side-by-side on desktop */}
      <div className="mb-4 flex flex-col gap-4 md:mb-8 md:flex-row md:flex-wrap md:items-start">
        {cluster.summary && (
          <p className="flex-1 border-l-2 border-editorial/40 pl-4 text-base leading-7 text-muted-foreground line-clamp-3 md:line-clamp-none md:min-w-50">
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
              Ажурирано {formatArticleDate(cluster.updated_at, cluster.created_at)}
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
            <ArticleItem
              key={article.id}
              article={article}
              displayNumber={i + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}
