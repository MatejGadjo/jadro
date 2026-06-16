import { Link } from '@tanstack/react-router'
import { Card } from '@/components/ui/card'
import { ArrowRight } from 'lucide-react'
import { TopicArticleItem } from '@/components/topics/TopicArticleItem'
import type { TopicArticle } from '@/lib/feedTypes'

interface TopicCardProps {
  topicId: string
  topicName: string
  articles: TopicArticle[]
}

export default function TopicCard({ topicId, topicName, articles }: TopicCardProps) {
  return (
    <Card className="bg-card border-border overflow-hidden flex flex-col h-full gap-0">
      {/* Header */}
      <div className="p-4 pb-2 border-b border-border">
        <Link
          to="/$topic"
          params={{ topic: topicId }}
          className="flex items-center justify-between group"
        >
          <h3
            className="text-3xl md:text-4xl font-black text-foreground leading-none group-hover:text-primary transition-colors"
            style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic' }}
          >
            {topicName}
          </h3>
          <ArrowRight className="w-4 h-4 text-muted-foreground group-hover:text-primary group-hover:translate-x-0.5 transition-all duration-150" />
        </Link>
      </div>

      {/* Articles */}
      <div className="flex-1 px-4">
        {articles.length === 0 ? (
          <div className="mt-4 rounded-lg border border-dashed border-border bg-muted/30 px-4 py-6 text-sm text-muted-foreground">
            Моментално нема достапни написи.
          </div>
        ) : (
          <div className="space-y-0">
            {articles.map((article) => (
              <TopicArticleItem key={article.id} article={article} />
            ))}
          </div>
        )}
      </div>
    </Card>
  )
}
