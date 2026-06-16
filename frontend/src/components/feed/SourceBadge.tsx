import { SourceAvatar } from '@/components/feed/SourceAvatar'
import type { FeedArticle } from '@/lib/feedTypes'

export function SourceBadge({ article }: { article: FeedArticle }) {
  return (
    <div className="flex items-center gap-1.5 shrink-0">
      <SourceAvatar
        source={article.source}
        className="w-4 h-4 rounded-sm ring-1 ring-border/60 shrink-0"
        fallbackClassName="text-[9px]"
      />
      <span
        className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wide"
        style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
      >
        {article.source.name}
      </span>
    </div>
  )
}

export default SourceBadge
