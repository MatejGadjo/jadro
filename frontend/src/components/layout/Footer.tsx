import { Link } from '@tanstack/react-router'
import { TOPIC_ITEMS } from '@/lib/topics'

export default function Footer() {
  const currentYear = new Date().getFullYear()

  return (
    <footer className="bg-card border-t-2 border-foreground mt-auto">

      {/* ── Desktop: 3-column grid ── */}
      <div className="hidden sm:block container mx-auto px-4 py-10">
        <div className="grid grid-cols-3 gap-10">

          {/* Brand column */}
          <div className="flex flex-col gap-3">
            <Link to="/" className="group flex items-baseline gap-1 w-fit">
              <span
                className="text-3xl font-black text-foreground group-hover:text-editorial transition-colors duration-150 leading-none"
                style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic', letterSpacing: '-0.02em' }}
              >
                jadro
              </span>
              <span
                className="text-3xl font-black text-editorial leading-none"
                style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic', letterSpacing: '-0.02em' }}
              >
                .mk
              </span>
            </Link>
            <p
              className="text-[11px] leading-relaxed text-muted-foreground max-w-50"
              style={{ fontFamily: 'var(--font-mono)' }}
            >
              Агрегатор на македонски вести. Повеќе извори, една слика.
            </p>
          </div>

          {/* Navigation column */}
          <div className="flex flex-col gap-3">
            <span
              className="text-[9px] tracking-[0.25em] uppercase text-muted-foreground"
              style={{ fontFamily: 'var(--font-mono)' }}
            >
              Навигација
            </span>
            <nav className="flex flex-col gap-2">
              <Link to="/" className="text-xs text-muted-foreground hover:text-editorial transition-colors duration-150 w-fit" style={{ fontFamily: 'var(--font-mono)' }}>Почетна</Link>
              <Link to="/about" className="text-xs text-muted-foreground hover:text-editorial transition-colors duration-150 w-fit" style={{ fontFamily: 'var(--font-mono)' }}>За нас</Link>
              <Link to="/privacy" className="text-xs text-muted-foreground hover:text-editorial transition-colors duration-150 w-fit" style={{ fontFamily: 'var(--font-mono)' }}>Приватност</Link>
              <Link to="/terms" className="text-xs text-muted-foreground hover:text-editorial transition-colors duration-150 w-fit" style={{ fontFamily: 'var(--font-mono)' }}>Услови</Link>
            </nav>
          </div>

          {/* Topics column */}
          <div className="flex flex-col gap-3">
            <span
              className="text-[9px] tracking-[0.25em] uppercase text-muted-foreground"
              style={{ fontFamily: 'var(--font-mono)' }}
            >
              Теми
            </span>
            <nav className="grid grid-cols-2 gap-x-4 gap-y-2">
              {TOPIC_ITEMS.map((topic) => (
                <Link
                  key={topic.value}
                  to="/$topic"
                  params={{ topic: topic.value }}
                  className="text-xs text-muted-foreground hover:text-editorial transition-colors duration-150 w-fit"
                  style={{ fontFamily: 'var(--font-mono)' }}
                >
                  {topic.label}
                </Link>
              ))}
            </nav>
          </div>

        </div>
      </div>

      {/* ── Mobile: dateline strip — everything on one scrollable line ── */}
      <div className="sm:hidden overflow-x-auto scrollbar-hide">
        <div className="flex items-center gap-0 px-4 py-3 whitespace-nowrap w-max min-w-full justify-center">
          <Link to="/" className="group flex items-baseline gap-0.5 shrink-0 pr-3">
            <span
              className="text-base font-black text-foreground group-hover:text-editorial transition-colors duration-150 leading-none"
              style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic', letterSpacing: '-0.02em' }}
            >
              jadro
            </span>
            <span
              className="text-base font-black text-editorial leading-none"
              style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic', letterSpacing: '-0.02em' }}
            >
              .mk
            </span>
          </Link>

          {[
            { to: '/', label: 'Почетна' },
            { to: '/about', label: 'За нас' },
            { to: '/privacy', label: 'Приватност' },
            { to: '/terms', label: 'Услови' },
          ].map(({ to, label }) => (
            <span key={to} className="flex items-center">
              <span className="text-[10px] text-muted-foreground/40 px-2" style={{ fontFamily: 'var(--font-mono)' }}>·</span>
              <Link
                to={to}
                className="text-[10px] text-muted-foreground hover:text-editorial transition-colors duration-150"
                style={{ fontFamily: 'var(--font-mono)' }}
              >
                {label}
              </Link>
            </span>
          ))}

          <span className="text-[10px] text-muted-foreground/40 px-2" style={{ fontFamily: 'var(--font-mono)' }}>·</span>
          <span className="text-[10px] text-muted-foreground/50" style={{ fontFamily: 'var(--font-mono)' }}>
            © {currentYear}
          </span>
        </div>
      </div>

      {/* Bottom bar — desktop only */}
      <div className="border-t border-border">
        <div className="hidden sm:flex container mx-auto px-4 py-3 flex-row items-center justify-between gap-2">
          <span
            className="text-[10px] text-muted-foreground"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            © {currentYear} jadro.mk
          </span>
          <span
            className="text-[10px] text-muted-foreground text-center"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            Содржината припаѓа на изворните медиуми. jadro.mk само ги агрегира.
          </span>
        </div>
      </div>

    </footer>
  )
}
