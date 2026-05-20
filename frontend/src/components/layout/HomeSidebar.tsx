import { useLatestArticles } from '@/hooks/useLatestArticles'
import type { LatestArticle } from '@/api/types'

const mono: React.CSSProperties = { fontFamily: 'var(--font-mono)' }

function mkTimeAgo(dateStr: string): string {
  const mins = Math.floor((Date.now() - new Date(dateStr).getTime()) / 60_000)
  if (mins < 1)  return 'пред момент'
  if (mins === 1) return 'пред 1 мин'
  if (mins < 60)  return `пред ${mins} мин`
  const hrs = Math.floor(mins / 60)
  return hrs === 1 ? 'пред 1 час' : `пред ${hrs} часа`
}

function SectionHeader({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2.5 mb-3 pb-2.5 border-b border-border">
      <div className="w-0.5 self-stretch bg-editorial shrink-0" style={{ minHeight: '1.1rem' }} />
      <span
        className="text-foreground uppercase font-black"
        style={{ ...mono, fontSize: '11px', letterSpacing: '0.12em' }}
      >
        {label}
      </span>
    </div>
  )
}

function ArticleRow({ article }: { article: LatestArticle }) {
  const rawTime = article.published_at ?? article.fetched_at

  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      className="group block py-2.5 border-b border-border/50 last:border-b-0 -mx-4 px-4 hover:bg-accent/40 transition-colors duration-100"
    >
      {/* Topic */}
      <span
        className="text-editorial uppercase font-bold"
        style={{ ...mono, fontSize: '9px', letterSpacing: '0.1em' }}
      >
        {article.topic}
      </span>

      {/* Title */}
      <p
        className="text-sm font-semibold text-foreground leading-snug line-clamp-2 group-hover:text-editorial transition-colors duration-100 mt-0.5 mb-1"
        style={{ fontFamily: 'var(--font-serif)' }}
      >
        {article.title}
      </p>

      {/* Source + time */}
      <div className="flex items-center justify-between gap-2">
        <span
          className="text-muted-foreground font-bold uppercase truncate"
          style={{ ...mono, fontSize: '10px', letterSpacing: '0.08em' }}
        >
          {article.source_name}
        </span>
        <span
          className="text-muted-foreground shrink-0"
          style={{ ...mono, fontSize: '9px' }}
        >
          {mkTimeAgo(rawTime)}
        </span>
      </div>
    </a>
  )
}

function SkeletonRow() {
  return (
    <div className="py-2.5 border-b border-border/50 last:border-b-0 -mx-4 px-4 animate-pulse">
      {/* Topic placeholder */}
      <div className="h-2 w-14 bg-muted rounded-sm mb-1.5" />
      {/* Title placeholder — two lines */}
      <div className="h-3 w-full bg-muted rounded-sm mb-1" />
      <div className="h-3 w-3/4 bg-muted rounded-sm mb-2" />
      {/* Source + time placeholder */}
      <div className="flex items-center justify-between">
        <div className="h-2 w-24 bg-muted rounded-sm" />
        <div className="h-2 w-14 bg-muted rounded-sm" />
      </div>
    </div>
  )
}

export default function HomeSidebar() {
  const { data, isLoading, isError } = useLatestArticles(5)

  return (
    <aside className="hidden lg:flex flex-col self-start">
      <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden">

        {/* ── Најнови вести feed ── */}
        <div className="p-4 pb-2">
          <SectionHeader label="Најнови вести" />

          {isLoading && (
            <div>
              {Array.from({ length: 5 }, (_, i) => (
                <SkeletonRow key={i} />
              ))}
            </div>
          )}

          {isError && (
            <p className="text-destructive py-4 text-center" style={{ ...mono, fontSize: '10px' }}>
              Неуспешно вчитување
            </p>
          )}

          {!isLoading && !isError && data && (
            <div>
              {data.articles.map((article) => (
                <ArticleRow key={article.id} article={article} />
              ))}
            </div>
          )}
        </div>

      </div>
    </aside>
  )
}
