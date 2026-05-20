import { Link } from '@tanstack/react-router'
import { clearArticleImage } from '@/api/cluster'
import { Card } from '@/components/ui/card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { ArrowRight, TrendingUp } from 'lucide-react'
import type { TopicArticle } from '@/lib/topicMockData'

interface TopicCardProps {
  topicId: string
  topicName: string
  articles: TopicArticle[]
  showStockData?: boolean
  stockData?: Array<{
    name: string
    value: string
    changePercent: string
    isPositive: boolean
  }>
}

function getSourceInitials(name: string) {
  return name
    .split(' ')
    .map(word => word[0])
    .join('')
    .substring(0, 2)
    .toUpperCase()
}

function getSourceColor(name: string) {
  const colors = [
    'bg-blue-500',
    'bg-green-500',
    'bg-purple-500',
    'bg-orange-500',
    'bg-pink-500',
    'bg-indigo-500',
    'bg-teal-500',
    'bg-red-500',
    'bg-cyan-500',
    'bg-amber-500'
  ]
  const index = name.length % colors.length
  return colors[index]
}

function ArticleItemContent({ article }: { article: TopicArticle }) {
  return (
    <div className="flex gap-3 h-full">
      <div className="flex-1 min-w-0 flex flex-col justify-between">
        {/* Source */}
        <div className="flex items-center gap-2 mb-1.5">
          <Avatar className="w-4 h-4 shrink-0">
            <AvatarImage
              src={article.source.logo_url}
              alt={`${article.source.name} logo`}
              className="bg-background object-contain p-0.5"
            />
            <AvatarFallback className={`text-[9px] ${getSourceColor(article.source.name)} text-white`}>
              {getSourceInitials(article.source.name)}
            </AvatarFallback>
          </Avatar>
          <span className="text-xs text-muted-foreground font-medium truncate">
            {article.source.name}
          </span>
        </div>

        {/* Title */}
        <h4 className="text-sm font-normal text-foreground leading-snug mb-2 group-hover:text-primary transition-colors line-clamp-2">
          {article.title}
        </h4>

        {/* Meta */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground mt-auto">
          <span>{article.publishedAt}</span>
          {article.author && (
            <>
              <span>•</span>
              <span className="truncate">By {article.author}</span>
            </>
          )}
        </div>
      </div>

      {/* Thumbnail */}
      {article.imageUrl && (
        <div className="shrink-0 w-20 h-20 rounded-lg overflow-hidden bg-muted mt-1">
          <img
            src={article.imageUrl}
            alt=""
            className="w-full h-full object-cover"
            onError={(e) => { e.currentTarget.style.visibility = 'hidden'; clearArticleImage(article.id) }}
          />
        </div>
      )}
    </div>
  )
}

function TopicArticleItem({ article }: { article: TopicArticle }) {
  return (
    <a
      href={article.url}
      target="_blank"
      rel="noopener noreferrer"
      className="group block w-full h-32 border-b border-border last:border-0 py-3 hover:bg-accent/30 transition-colors"
    >
      <ArticleItemContent article={article} />
    </a>
  )
}

export default function TopicCard({ topicId, topicName, articles, showStockData, stockData }: TopicCardProps) {
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
            {articles.map(article => (
              <TopicArticleItem key={article.id} article={article} />
            ))}
          </div>
        )}
      </div>
      
      {/* Stock Market Data (for Business topic) */}
      {showStockData && stockData && stockData.length > 0 && (
        <div className="px-4 pb-4">
          <div className="grid grid-cols-3 gap-2">
            {stockData.map((stock, index) => (
              <div
                key={index}
                className="bg-muted/50 rounded-lg p-3 border border-border hover:bg-muted transition-colors"
              >
                <div className="flex items-start gap-1.5 mb-1">
                  <TrendingUp className={`w-3.5 h-3.5 mt-0.5 ${stock.isPositive ? 'text-green-500' : 'text-red-500'}`} />
                  <div className="flex-1 min-w-0">
                    <div className="text-xs font-medium text-foreground truncate">
                      {stock.name}
                    </div>
                    <div className={`text-lg font-semibold ${stock.isPositive ? 'text-green-500' : 'text-red-500'}`}>
                      {stock.changePercent}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {stock.value}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
          <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
            <span>14 hours ago</span>
            <a href="#" className="hover:text-primary transition-colors flex items-center gap-1">
              <span>More from Google Finance</span>
            </a>
          </div>
        </div>
      )}
    </Card>
  )
}
