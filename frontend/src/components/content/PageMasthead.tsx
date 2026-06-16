interface PageMastheadProps {
  eyebrow: string
  title: string
  lede?: string
  dateline?: string
}

/**
 * Editorial page header shared by the static content pages (За нас / Приватност / Услови).
 * Mirrors the broadsheet masthead used on the homepage: mono eyebrow, serif italic
 * display title and the crimson editorial accent bar.
 */
export default function PageMasthead({ eyebrow, title, lede, dateline }: PageMastheadProps) {
  return (
    <header className="mb-10">
      <div className="flex items-stretch gap-3 border-b-2 border-foreground pb-4">
        <div className="w-1 self-stretch bg-editorial shrink-0" style={{ minHeight: '3rem' }} />
        <div>
          <p
            className="text-[10px] uppercase tracking-[0.28em] text-muted-foreground mb-2"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            {eyebrow}
          </p>
          <h1
            className="text-4xl md:text-5xl font-black text-foreground leading-[1.05]"
            style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic' }}
          >
            {title}
          </h1>
        </div>
      </div>

      {lede && (
        <p
          className="mt-5 text-lg md:text-xl leading-relaxed text-foreground/90 max-w-2xl"
          style={{ fontFamily: 'var(--font-serif)' }}
        >
          {lede}
        </p>
      )}

      {dateline && (
        <p
          className="mt-4 text-[10px] uppercase tracking-[0.22em] text-muted-foreground"
          style={{ fontFamily: 'var(--font-mono)' }}
        >
          {dateline}
        </p>
      )}
    </header>
  )
}
