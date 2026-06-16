import { Link } from '@tanstack/react-router'
import { trackEvent } from '@/analytics'
import { ArrowRight, Layers } from 'lucide-react'
import { SourceAvatar } from '@/components/feed/SourceAvatar'
import { SourceBadge } from '@/components/feed/SourceBadge'
import { FeaturedArticle } from '@/components/feed/FeaturedArticle'
import type { FeedCluster, FeedArticle } from '@/lib/feedTypes'

interface StoryClusterProps {
  cluster: FeedCluster
  featured?: boolean
  embedded?: boolean
}

function ArticleRow({ article }: { article: FeedArticle }) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={() => trackEvent('article_click', { source: article.source.name })}
      className="group flex items-start gap-3 py-2.5 border-t border-border/60 first:border-t-0 hover:bg-accent/40 -mx-3 px-3 transition-colors duration-100"
    >
      <SourceAvatar
        source={article.source}
        className="w-5 h-5 rounded-sm ring-1 ring-border/50 shrink-0 mt-0.5"
        fallbackClassName="text-[9px]"
      />
      <div className="flex-1 min-w-0">
        <div className="flex items-baseline gap-2 mb-0.5">
          <span
            className="text-muted-foreground font-semibold uppercase shrink-0"
            style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.06em' }}
          >
            {article.source.name}
          </span>
          {article.publishedAt && (
            <span
              className="text-muted-foreground"
              style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
            >
              {article.publishedAt}
            </span>
          )}
        </div>
        <h4
          className="text-sm font-semibold text-foreground leading-snug line-clamp-2 group-hover:text-editorial transition-colors duration-150"
          style={{ fontFamily: 'var(--font-serif)' }}
        >
          {article.title}
        </h4>
      </div>
    </a>
  )
}

function FeaturedArticleRow({ article, fill = true }: { article: FeedArticle; fill?: boolean }) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={() => trackEvent('article_click', { source: article.source.name })}
      className={`group flex flex-col px-3 py-2 border-t border-border/60 first:border-t-0 hover:bg-accent/40 transition-colors duration-100 ${fill ? 'flex-1 min-h-0' : 'min-h-18'}`}
    >
      <div className="flex items-center gap-1.5 mb-1">
        <SourceAvatar
          source={article.source}
          className="w-3.5 h-3.5 rounded-sm ring-1 ring-border/50 shrink-0"
          fallbackClassName="text-[8px]"
        />
        <span
          className="text-muted-foreground font-semibold uppercase"
          style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.06em' }}
        >
          {article.source.name}
        </span>
      </div>
      <h4
        className={`text-sm font-semibold text-foreground leading-snug group-hover:text-editorial transition-colors duration-150 ${fill ? 'line-clamp-2 flex-1' : 'line-clamp-3'}`}
        style={{ fontFamily: 'var(--font-serif)' }}
      >
        {article.title}
      </h4>
      {article.publishedAt && (
        <span
          className="text-muted-foreground mt-1 shrink-0"
          style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
        >
          {article.publishedAt}
          {article.author && ` · ${article.author}`}
        </span>
      )}
    </a>
  )
}

function CompanionArticle({ article }: { article: FeedArticle }) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={() => trackEvent('article_click', { source: article.source.name })}
      className="group flex flex-col pl-4 border-l border-border/60 hover:bg-accent/40 transition-colors duration-100 rounded-sm"
    >
      <div className="flex items-center gap-1.5 mb-2">
        <SourceAvatar
          source={article.source}
          className="w-3.5 h-3.5 rounded-sm ring-1 ring-border/50 shrink-0"
          fallbackClassName="text-[8px]"
        />
        <span
          className="text-muted-foreground font-semibold uppercase"
          style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.06em' }}
        >
          {article.source.name}
        </span>
      </div>
      <h4
        className="text-base font-bold text-foreground leading-snug line-clamp-4 group-hover:text-editorial transition-colors duration-150"
        style={{ fontFamily: 'var(--font-serif)' }}
      >
        {article.title}
      </h4>
      {article.publishedAt && (
        <span
          className="text-muted-foreground mt-2 shrink-0"
          style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
        >
          {article.publishedAt}
          {article.author && ` · ${article.author}`}
        </span>
      )}
    </a>
  )
}

function SeeMoreLink({ clusterId }: { clusterId: number | string }) {
  return (
    <div className="mt-2 pt-2 border-t border-border/60">
      <Link
        to="/cluster/$clusterId"
        params={{ clusterId: String(clusterId) }}
        className="group flex items-center justify-center gap-2 px-3 py-2 md:py-1.5 rounded-sm border border-border/60 bg-muted/40 hover:bg-muted hover:border-editorial/40 transition-all duration-150"
        style={{ fontFamily: 'var(--font-mono)', letterSpacing: '0.05em' }}
      >
        <Layers className="w-3 h-3 text-editorial shrink-0" />
        <span className="text-xs font-semibold uppercase tracking-wider text-foreground">Сите перспективи</span>
        <ArrowRight className="w-3 h-3 text-muted-foreground transition-transform duration-150 group-hover:translate-x-0.5 group-hover:text-editorial" />
      </Link>
    </div>
  )
}

function HorizontalArticleCard({ article }: { article: FeedArticle }) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={() => trackEvent('article_click', { source: article.source.name })}
      className="group flex flex-col w-52 shrink-0 bg-card border border-border rounded-sm p-3 hover:border-editorial/40 transition-colors duration-150"
    >
      <div className="flex items-center gap-1.5 mb-2">
        <SourceAvatar
          source={article.source}
          className="w-4 h-4 rounded-sm shrink-0"
          fallbackClassName="text-[8px]"
        />
        <span
          className="text-foreground font-bold uppercase truncate"
          style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.06em' }}
        >
          {article.source.name}
        </span>
      </div>
      <h4
        className="text-sm font-semibold text-foreground leading-snug line-clamp-3 group-hover:text-editorial transition-colors duration-150 flex-1"
        style={{ fontFamily: 'var(--font-serif)' }}
      >
        {article.title}
      </h4>
      {article.publishedAt && (
        <span
          className="text-muted-foreground mt-2 shrink-0"
          style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
        >
          {article.publishedAt}
        </span>
      )}
    </a>
  )
}

function HorizontalArticleScroll({ articles }: { articles: FeedArticle[] }) {
  return (
    <div className="relative -mx-4">
      <div className="pointer-events-none absolute right-0 top-0 bottom-0 w-12 z-10 bg-linear-to-l from-card to-transparent" />
      <div className="overflow-x-auto scrollbar-hide px-4">
        <div className="flex gap-3 pb-1">
          {articles.map((article) => (
            <HorizontalArticleCard key={article.id} article={article} />
          ))}
        </div>
      </div>
    </div>
  )
}

export default function StoryCluster({ cluster, featured = false, embedded = false }: StoryClusterProps) {
  const [mainArticle, ...otherArticles] = cluster.articles
  const clusterTitle = cluster.displayTitle || 'Топ приказни'

  const containerClasses = embedded
    ? ''
    : 'bg-card border border-border rounded-sm overflow-hidden hover:border-editorial/30 transition-colors duration-150'

  const innerClasses = 'p-4 md:p-5'

  if (featured && mainArticle?.imageUrl) {
    return (
      <div className={containerClasses}>
        <div className={innerClasses}>
          {/* Cluster label — desktop only */}
          <div className="hidden md:flex items-center gap-3 mb-3">
            <div className="h-4 w-0.5 bg-editorial shrink-0" />
            <Link
              to="/cluster/$clusterId"
              params={{ clusterId: String(cluster.id) }}
              className="group inline-flex items-center gap-1.5"
            >
              <span
                className="text-xs font-bold uppercase text-foreground group-hover:text-editorial transition-colors"
                style={{ fontFamily: 'var(--font-mono)', letterSpacing: '0.12em' }}
              >
                {clusterTitle}
              </span>
              <ArrowRight className="w-3.5 h-3.5 text-muted-foreground group-hover:text-editorial group-hover:translate-x-0.5 transition-all duration-150" />
            </Link>
          </div>

          {/* Desktop: two-column layout, adapts to secondary article count */}
          {(() => {
            const secondaryArticles = otherArticles.slice(0, 3)
            const count = secondaryArticles.length

            if (count === 0) {
              return (
                <div className="hidden md:flex flex-col">
                  <FeaturedArticle article={mainArticle} />
                  <SeeMoreLink clusterId={cluster.id} />
                </div>
              )
            }

            if (count === 1) {
              return (
                <div className="hidden md:grid md:grid-cols-2 gap-4">
                  <div className="flex flex-col">
                    <FeaturedArticle article={mainArticle} />
                    <SeeMoreLink clusterId={cluster.id} />
                  </div>
                  <CompanionArticle article={secondaryArticles[0]} />
                </div>
              )
            }

            return (
              <div className="hidden md:grid md:grid-cols-5 gap-4">
                <div className="col-span-2 flex flex-col">
                  <FeaturedArticle article={mainArticle} />
                  <SeeMoreLink clusterId={cluster.id} />
                </div>
                <div className="col-span-3 flex flex-col border-l border-border/60">
                  {secondaryArticles.map((article) => (
                    <FeaturedArticleRow key={article.id} article={article} fill={count === 3} />
                  ))}
                </div>
              </div>
            )
          })()}

          {/* Mobile: main article + horizontal scroll strip */}
          <div className="md:hidden">
            <FeaturedArticle article={mainArticle} />
            {otherArticles.length > 0 && (
              <div className="mt-3">
                <HorizontalArticleScroll articles={otherArticles.slice(0, 3)} />
              </div>
            )}
            <SeeMoreLink clusterId={cluster.id} />
          </div>
        </div>
      </div>
    )
  }

  // Default non-featured layout
  return (
    <div className={containerClasses}>
      <div className={innerClasses}>
        {/* Cluster label — desktop only */}
        <div className="hidden md:flex items-center gap-3 mb-3">
          <div className="h-4 w-0.5 bg-border group-hover:bg-editorial shrink-0" />
          <Link
            to="/cluster/$clusterId"
            params={{ clusterId: String(cluster.id) }}
            className="group inline-flex items-center gap-1.5"
          >
            <span
              className="text-xs font-bold uppercase text-muted-foreground group-hover:text-foreground transition-colors"
              style={{ fontFamily: 'var(--font-mono)', letterSpacing: '0.12em' }}
            >
              {clusterTitle}
            </span>
            <ArrowRight className="w-3.5 h-3.5 text-muted-foreground/50 group-hover:text-foreground group-hover:translate-x-0.5 transition-all duration-150" />
          </Link>
        </div>

        {mainArticle && (
          <div className="mb-1">
            <a href={mainArticle.url} target="_blank" rel="noopener noreferrer" onClick={() => trackEvent('article_click', { source: mainArticle.source.name })} className="group block">
              <div className="flex items-center gap-2 mb-1.5">
                <SourceBadge article={mainArticle} />
              </div>
              <h3
                className="text-lg font-bold text-foreground leading-tight group-hover:text-editorial transition-colors duration-150"
                style={{ fontFamily: 'var(--font-serif)' }}
              >
                {mainArticle.title}
              </h3>
            </a>
          </div>
        )}

        {otherArticles.length > 0 && (
          <>
            {/* Desktop: stacked rows */}
            <div className="hidden md:block mt-1">
              {otherArticles.slice(0, 3).map((article) => (
                <ArticleRow key={article.id} article={article} />
              ))}
            </div>
            {/* Mobile: horizontal scroll strip */}
            <div className="md:hidden mt-3">
              <HorizontalArticleScroll articles={otherArticles.slice(0, 3)} />
            </div>
          </>
        )}

        <SeeMoreLink clusterId={cluster.id} />
      </div>
    </div>
  )
}
