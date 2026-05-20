import { Link, useParams } from '@tanstack/react-router'

const topics = [
  { id: 'politika', label: 'Politika' },
  { id: 'sport', label: 'Sport' },
  { id: 'tehnologija', label: 'Tehnologija' },
  { id: 'ekonomija', label: 'Ekonomija' },
  { id: 'kultura', label: 'Kultura' },
  { id: 'zdravlje', label: 'Zdravlje' },
]

export default function TopicSidebar() {
  const { topic } = useParams({ strict: false })

  return (
    <aside className="bg-card rounded-lg shadow-sm p-4">
      <nav className="space-y-1">
        {topics.map((topicItem) => {
          const isActive = topic === topicItem.id
          return (
            <Link
              key={topicItem.id}
              to="/$topic"
              params={{ topic: topicItem.id }}
              className={`block px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-primary/10 text-primary border-l-4 border-primary'
                  : 'text-foreground hover:bg-accent hover:text-accent-foreground'
              }`}
            >
              {topicItem.label}
            </Link>
          )
        })}
      </nav>
    </aside>
  )
}
