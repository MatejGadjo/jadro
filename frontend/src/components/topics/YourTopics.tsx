import TopicCard from './TopicCard'
import type { TopicSection } from '@/lib/feedTypes'

interface YourTopicsProps {
  sections: TopicSection[]
}

export default function YourTopics({ sections }: YourTopicsProps) {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h2
            className="text-3xl md:text-4xl font-black text-foreground leading-none mb-2"
            style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic' }}
          >
            Теми
          </h2>
        </div>
      </div>

      {/* Topics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {sections.length === 0 ? (
          <div className="md:col-span-2 lg:col-span-3 rounded-lg border border-border bg-card p-6 text-sm text-muted-foreground">
            Моментално нема достапни теми.
          </div>
        ) : (
          sections.map((topic) => (
            <TopicCard
              key={topic.id}
              topicId={topic.id}
              topicName={topic.name}
              articles={topic.articles}
            />
          ))
        )}
      </div>
    </div>
  )
}
