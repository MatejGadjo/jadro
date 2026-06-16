import { Link } from '@tanstack/react-router'
import type { Cluster } from '../../api/types'

interface FeedCardProps {
  cluster: Cluster
}

export default function FeedCard({ cluster }: FeedCardProps) {
  const headline = cluster.display_title || 'Топ вести'

  return (
    <Link
      to="/cluster/$clusterId"
      params={{ clusterId: cluster.id.toString() }}
      className="group block border border-border bg-card hover:border-editorial/40 transition-colors duration-150 rounded-sm overflow-hidden"
    >
      <div className="flex items-stretch">
        {/* Editorial left accent bar */}
        <div className="w-1 flex-shrink-0 bg-border group-hover:bg-editorial transition-colors duration-150" />
        <div className="flex-1 px-4 py-3">
          <h2
            className="text-lg font-bold text-foreground leading-snug group-hover:text-editorial transition-colors duration-150"
            style={{ fontFamily: 'var(--font-serif)' }}
          >
            {headline}
          </h2>
          {cluster.summary && (
            <p className="text-sm text-muted-foreground mt-1.5 leading-relaxed line-clamp-2 hidden sm:block">
              {cluster.summary}
            </p>
          )}
        </div>
      </div>
    </Link>
  )
}
