import { Link } from '@tanstack/react-router'
import { TOPIC_ITEMS } from '@/lib/topics'

// NotFound is the branded 404 page shown for unknown routes, invalid topic
// slugs, and missing clusters. Rendered inside MainLayout via the layout
// route's notFoundComponent, so it keeps the site header/footer.
export function NotFound({
  title = 'Страницата не постои',
  message = 'Страницата што ја баравте не постои или е преместена.',
}: {
  title?: string
  message?: string
}) {
  return (
    <div className="mx-auto max-w-272 py-16 text-center md:py-24">
      <div
        className="text-editorial"
        style={{
          fontFamily: 'var(--font-serif)',
          fontStyle: 'italic',
          fontWeight: 900,
          fontSize: 'clamp(3.5rem, 14vw, 6rem)',
          lineHeight: 1,
        }}
      >
        404
      </div>

      <h1
        className="mt-3 text-foreground"
        style={{
          fontFamily: 'var(--font-serif)',
          fontStyle: 'italic',
          fontWeight: 900,
          fontSize: 'clamp(1.375rem, 5vw, 2rem)',
          lineHeight: 1.2,
        }}
      >
        {title}
      </h1>

      <p className="mx-auto mt-3 max-w-prose text-sm leading-6 text-muted-foreground">
        {message}
      </p>

      <div className="mt-8">
        <Link
          to="/"
          className="inline-flex items-center rounded-sm bg-editorial px-4 py-2 text-sm font-semibold text-white transition-colors duration-150 hover:bg-editorial/90"
        >
          Назад на почетна
        </Link>
      </div>

      <div className="mt-10 flex flex-wrap justify-center gap-2">
        {TOPIC_ITEMS.map((topic) => (
          <Link
            key={topic.id}
            to="/$topic"
            params={{ topic: topic.id }}
            className="rounded-sm border border-border/70 px-3 py-1 uppercase text-muted-foreground transition-colors duration-150 hover:border-editorial hover:text-editorial"
            style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.08em' }}
          >
            {topic.label}
          </Link>
        ))}
      </div>
    </div>
  )
}

export default NotFound
