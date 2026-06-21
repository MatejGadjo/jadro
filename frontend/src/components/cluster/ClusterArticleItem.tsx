import { clearArticleImage } from '@/api/cluster'
import { trackEvent } from '@/analytics'
import { SourceAvatar } from '@/components/feed/SourceAvatar'
import { toRelativeTime } from '@/lib/clusterArticles'
import type { ClusterResponse } from '@/api/types'

type ClusterArticle = ClusterResponse['articles'][number]

export function ClusterArticleItem({
  article,
  displayNumber,
  topic,
}: {
  article: ClusterArticle
  displayNumber: number
  topic?: string
}) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={() => trackEvent('article_click', { source: article.source.name, ...(topic ? { topic } : {}) })}
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
          <SourceAvatar
            source={article.source}
            className="h-4 w-4 rounded-sm ring-1 ring-border/50 shrink-0"
            fallbackClassName="text-[8px]"
          />
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
            {toRelativeTime(article.published_at, article.fetched_at)}
          </span>
        </div>

        <h3
          className="line-clamp-2 text-base font-bold leading-snug text-foreground transition-colors duration-150 group-hover:text-editorial md:text-[1.05rem]"
          style={{ fontFamily: 'var(--font-sans)' }}
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

export default ClusterArticleItem
