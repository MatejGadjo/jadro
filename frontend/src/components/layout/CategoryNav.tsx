import { Link, useParams, useRouterState } from '@tanstack/react-router'
import { TOPIC_ITEMS } from '@/lib/topics'

const navItems = [
  { id: 'home', label: 'Почетна', path: '/' },
]

export default function CategoryNav() {
  const { topic } = useParams({ strict: false })
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const isHomeActive = pathname === '/'

  return (
    <nav className="bg-foreground">
      <div className="container mx-auto px-4">
        <div className="overflow-x-auto scrollbar-hide">
          <div className="flex items-stretch min-w-max lg:min-w-0 lg:justify-center">
            {navItems.map((item) => {
              const isActive = item.path === '/' ? isHomeActive : pathname === item.path
              return (
                <Link
                  key={item.id}
                  to={item.path}
                  className={[
                    'whitespace-nowrap px-4 py-2.5 text-xs font-semibold tracking-widest uppercase transition-colors duration-150 border-b-2',
                    'relative',
                    isActive
                      ? 'text-card bg-editorial border-editorial'
                      : 'text-card/60 border-transparent hover:text-card hover:border-card/30',
                  ].join(' ')}
                  style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.1em' }}
                >
                  {item.label}
                </Link>
              )
            })}

            {/* Divider */}
            <div className="w-px bg-card/20 my-1.5" />

            {TOPIC_ITEMS.map((topicItem) => {
              const isActive = topic === topicItem.id
              return (
                <Link
                  key={topicItem.id}
                  to="/$topic"
                  params={{ topic: topicItem.id }}
                  className={[
                    'whitespace-nowrap px-4 py-2.5 text-xs font-semibold tracking-widest uppercase transition-colors duration-150 border-b-2',
                    isActive
                      ? 'text-card bg-editorial border-editorial'
                      : 'text-card/60 border-transparent hover:text-card hover:border-card/30',
                  ].join(' ')}
                  style={{ fontFamily: 'var(--font-mono)', fontSize: '10px', letterSpacing: '0.1em' }}
                >
                  {topicItem.label}
                </Link>
              )
            })}
          </div>
        </div>
      </div>
    </nav>
  )
}
