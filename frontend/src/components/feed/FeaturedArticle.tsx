import { clearArticleImage } from '@/api/cluster'
import { trackEvent } from '@/analytics'
import { SourceAvatar } from '@/components/feed/SourceAvatar'
import type { FeedArticle } from '@/lib/feedTypes'

export function FeaturedArticle({ article, topic }: { article: FeedArticle; topic?: string }) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={() => trackEvent('article_click', { source: article.source.name, ...(topic ? { topic } : {}) })}
      className="group flex flex-col h-full"
    >
      {article.imageUrl && (
        <div className="relative w-full aspect-video overflow-hidden bg-muted mb-2 rounded-sm">
          <img
            src={article.imageUrl}
            alt={article.title}
            className="w-full h-full object-cover group-hover:scale-[1.02] transition-transform duration-500"
            onError={(e) => { e.currentTarget.style.visibility = 'hidden'; clearArticleImage(article.id) }}
          />
        </div>
      )}

      {/* Source below image */}
      <div className="flex items-center gap-1.5 mb-1.5">
        <SourceAvatar
          source={article.source}
          className="w-4 h-4 rounded-sm ring-1 ring-border/60 shrink-0"
          fallbackClassName="text-[9px]"
        />
        <span
          className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wide"
          style={{ fontFamily: 'var(--font-mono)' }}
        >
          {article.source.name}
        </span>
      </div>

      <h3
        className="text-base font-bold text-foreground leading-snug group-hover:text-editorial transition-colors duration-150 flex-1"
        style={{ fontFamily: 'var(--font-sans)' }}
      >
        {article.title}
      </h3>

      {article.publishedAt && (
        <div className="mt-1.5">
          <span
            className="text-muted-foreground"
            style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
          >
            {article.publishedAt}
            {article.author && ` · ${article.author}`}
          </span>
        </div>
      )}
    </a>
  )
}

export default FeaturedArticle
