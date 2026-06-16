import StoryCluster from '@/components/feed/StoryCluster'
import type { FeedCluster } from '@/lib/feedTypes'

interface TopicFeedSectionProps {
  topicName: string
  clusters: FeedCluster[]
}

export default function TopicFeedSection({ topicName, clusters }: TopicFeedSectionProps) {
  return (
    <section className="space-y-6">
      <div className="flex items-end justify-between border-b-2 border-foreground pb-3">
        <div className="flex items-end gap-3">
          <div
            className="w-1 self-stretch bg-editorial flex-shrink-0"
            style={{ minHeight: '2.5rem' }}
          />
          <div>
            <h1
              className="text-3xl md:text-4xl font-black text-foreground leading-none"
              style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic' }}
            >
              {topicName}
            </h1>
            <p
              className="text-xs text-muted-foreground mt-1 uppercase tracking-widest"
              style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
            >
              Последни вести и перспективи
            </p>
          </div>
        </div>
      </div>

      <div className="max-w-4xl mx-auto space-y-6">
        {clusters.map((cluster) => (
          <StoryCluster key={cluster.id} cluster={cluster} featured />
        ))}
      </div>
    </section>
  )
}
