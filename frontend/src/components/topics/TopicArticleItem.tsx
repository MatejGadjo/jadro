import { clearArticleImage } from '@/api/cluster'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { getSourceColor, getSourceInitials } from '@/lib/source'
import type { TopicArticle } from '@/lib/feedTypes'

function ArticleItemContent({ article }: { article: TopicArticle }) {
  return (
    <div className="flex gap-3 h-full">
      <div className="flex-1 min-w-0 flex flex-col justify-between">
        {/* Source */}
        <div className="flex items-center gap-2 mb-1.5">
          <Avatar className="w-4 h-4 shrink-0">
            <AvatarImage
              src={article.source.logo_url}
              alt={`${article.source.name} logo`}
              className="bg-background object-contain p-0.5"
            />
            <AvatarFallback className={`text-[9px] ${getSourceColor(article.source.name)} text-white`}>
              {getSourceInitials(article.source.name)}
            </AvatarFallback>
          </Avatar>
          <span className="text-xs text-muted-foreground font-medium truncate">
            {article.source.name}
          </span>
        </div>

        {/* Title */}
        <h4
          className="text-sm font-semibold text-foreground leading-snug mb-2 group-hover:text-primary transition-colors line-clamp-2"
          style={{ fontFamily: 'var(--font-sans)' }}
        >
          {article.title}
        </h4>

        {/* Meta */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground mt-auto">
          <span>{article.publishedAt}</span>
          {article.author && (
            <>
              <span>•</span>
              <span className="truncate">By {article.author}</span>
            </>
          )}
        </div>
      </div>

      {/* Thumbnail */}
      {article.imageUrl && (
        <div className="shrink-0 w-20 h-20 rounded-lg overflow-hidden bg-muted mt-1">
          <img
            src={article.imageUrl}
            alt=""
            className="w-full h-full object-cover"
            onError={(e) => { e.currentTarget.style.visibility = 'hidden'; clearArticleImage(article.id) }}
          />
        </div>
      )}
    </div>
  )
}

export function TopicArticleItem({ article }: { article: TopicArticle }) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      className="group block w-full h-32 border-b border-border last:border-0 py-3 hover:bg-accent/30 transition-colors"
    >
      <ArticleItemContent article={article} />
    </a>
  )
}

export default TopicArticleItem
