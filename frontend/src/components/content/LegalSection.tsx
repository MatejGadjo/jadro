import type { ReactNode } from 'react'

interface LegalSectionProps {
  /** Clause number — legal pages are genuinely a numbered, referenceable sequence. */
  index: number
  title: string
  children: ReactNode
}

/**
 * A single numbered clause for the legal pages (Приватност / Услови).
 * The clause number echoes the mono numbering used in the "Како функционира"
 * pipeline on the About page, keeping the static pages visually related.
 */
export default function LegalSection({ index, title, children }: LegalSectionProps) {
  const num = String(index).padStart(2, '0')

  return (
    <section className="grid grid-cols-[2.5rem_1fr] md:grid-cols-[3.5rem_1fr] gap-x-3 border-t border-border pt-6">
      <span
        className="text-xl md:text-2xl font-black text-editorial tabular-nums leading-none select-none"
        style={{ fontFamily: 'var(--font-mono)' }}
      >
        {num}
      </span>
      <div>
        <h2
          className="text-xl md:text-2xl font-bold text-foreground mb-3"
          style={{ fontFamily: 'var(--font-serif)' }}
        >
          {title}
        </h2>
        <div className="space-y-3 text-[15px] leading-relaxed text-foreground/85">
          {children}
        </div>
      </div>
    </section>
  )
}
